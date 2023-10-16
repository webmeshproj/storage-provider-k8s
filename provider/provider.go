/*
Copyright 2023 Avi Zimmerman <avi.zimmerman@gmail.com>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package provider contains the storage provider implementation for Kubernetes.
package provider

import (
	"context"
	"fmt"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-logr/logr"
	v1 "github.com/webmeshproj/api/v1"
	"github.com/webmeshproj/webmesh/pkg/storage"
	"github.com/webmeshproj/webmesh/pkg/storage/errors"
	"github.com/webmeshproj/webmesh/pkg/storage/types"
	corev1 "k8s.io/api/core/v1"
	coordinationv1client "k8s.io/client-go/kubernetes/typed/coordination/v1"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/handler"

	"github.com/webmeshproj/storage-provider-k8s/provider/database"
	"github.com/webmeshproj/storage-provider-k8s/provider/manager"
)

//go:generate go run sigs.k8s.io/controller-tools/cmd/controller-gen@latest rbac:roleName=webmesh-storage-role paths="./..." output:rbac:artifacts:config=../deploy

const (
	// LeaderElectionID is the name of the leader election lease.
	LeaderElectionID = "storage-provider-k8s-leader-election"
)

// Ensure we satisfy the provider interface.
var _ storage.Provider = &Provider{}

// Options are the options for configuring the provider.
type Options struct {
	// NodeID is the ID of the node.
	NodeID string
	// ListenPort is the port the storage API is listening on, if any.
	ListenPort int
	// MetricsPort is the address to bind the metrics endpoint to.
	MetricsPort int
	// ProbePort is the address to bind the health probe endpoint to.
	ProbePort int
	// ShutdownTimeout is the timeout for shutting down the provider.
	ShutdownTimeout time.Duration
	// Namespace is the namespace to use for leader election and storage.
	Namespace string
	// LeaderElectionLeaseDuration is the duration of the leader election lease.
	LeaderElectionLeaseDuration time.Duration
	// LeaderElectionRenewDeadline is the duration of the leader election lease renewal deadline.
	LeaderElectionRenewDeadline time.Duration
	// LeaderElectionRetryPeriod is the duration of the leader election lease retry period.
	LeaderElectionRetryPeriod time.Duration
}

// Provider is the storage provider implementation for Kubernetes.
type Provider struct {
	Options
	started   atomic.Bool
	laddr     net.Addr
	lport     uint16
	mgr       manager.Manager
	db        storage.MeshDB
	datastore *database.Database
	storage   *Storage
	consensus *Consensus
	leaders   *leaderelection.LeaderElector
	errc      chan error
	subs      map[string]Subscription
	subsmu    sync.Mutex
	stop      context.CancelFunc
	log       logr.Logger
	mu        sync.Mutex
}

// New creates a new Provider.
func New(options Options) (*Provider, error) {
	mgr, err := manager.New(manager.Options{
		MetricsPort:     options.MetricsPort,
		ProbePort:       options.ProbePort,
		ShutdownTimeout: options.ShutdownTimeout,
	})
	if err != nil {
		return nil, fmt.Errorf("create controller manager: %w", err)
	}
	return NewWithManager(mgr, options)
}

// NewObserverWithConfig creates a new observing storage provider with the given rest config.
// An observing storage provider can still write to storage depending on the given configuration,
// but it will not be able to mutate or participate in consensus.
func NewObserverWithConfig(cfg *rest.Config, options Options) (*Provider, error) {
	mgr, err := manager.NewFromConfig(cfg, manager.Options{
		MetricsPort:     options.MetricsPort,
		ProbePort:       options.ProbePort,
		ShutdownTimeout: options.ShutdownTimeout,
	})
	if err != nil {
		return nil, fmt.Errorf("create controller manager: %w", err)
	}
	return NewObserverWithManager(mgr, options)
}

// NewObserverWithManager creates an observing storage provider with the given manager.
func NewObserverWithManager(mgr manager.Manager, options Options) (*Provider, error) {
	p := &Provider{
		Options: options,
		mgr:     mgr,
		subs:    make(map[string]Subscription),
		errc:    make(chan error, 1),
		log:     ctrl.Log.WithName("storage-provider"),
	}
	p.storage = &Storage{Provider: p}
	p.consensus = &Consensus{provider: p, isObserver: true}
	err := p.setupWithManager(mgr, options, true)
	if err != nil {
		return nil, fmt.Errorf("setup provider with manager: %w", err)
	}
	return p, nil
}

// NewWithManager creates a new Provider with the given manager.
func NewWithManager(mgr manager.Manager, options Options) (*Provider, error) {
	p := &Provider{
		Options: options,
		mgr:     mgr,
		subs:    make(map[string]Subscription),
		errc:    make(chan error, 1),
		log:     ctrl.Log.WithName("storage-provider"),
	}
	p.storage = &Storage{Provider: p}
	p.consensus = &Consensus{provider: p, isObserver: false}
	err := p.setupWithManager(mgr, options, false)
	if err != nil {
		return nil, fmt.Errorf("setup provider with manager: %w", err)
	}
	return p, nil
}

func (p *Provider) setupWithManager(mgr ctrl.Manager, opts Options, isObserver bool) error {
	laddr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("[::]:%d", opts.ListenPort))
	if err != nil {
		return fmt.Errorf("resolve listen address: %w", err)
	}
	p.laddr = laddr
	p.lport = uint16(laddr.Port)
	if p.Namespace == "" {
		var err error
		p.Namespace, err = getInClusterNamespace()
		if err != nil {
			return fmt.Errorf("get in-cluster namespace: %w", err)
		}
	}
	// Register the reconciler with the manager.
	err = ctrl.
		NewControllerManagedBy(p.mgr).
		Named(p.NodeID).
		Watches(&corev1.Secret{}, handler.EnqueueRequestsFromMapFunc(p.enqueueObjectIfOwner)).
		Complete(p)
	if err != nil {
		return fmt.Errorf("register controller: %w", err)
	}
	// Register the database with the manager
	p.db, p.datastore, err = database.New(mgr, database.Options{
		NodeID:     types.NodeID(opts.NodeID),
		Namespace:  opts.Namespace,
		ListenAddr: laddr,
	})
	if err != nil {
		return fmt.Errorf("create database: %w", err)
	}
	// If we are an observer, we are done here.
	if isObserver {
		return nil
	}
	// Create clients for leader election
	cfg := rest.CopyConfig(p.mgr.GetConfig())
	corev1client, err := corev1client.NewForConfig(cfg)
	if err != nil {
		return fmt.Errorf("create corev1 client: %w", err)
	}
	coordinationClient, err := coordinationv1client.NewForConfig(cfg)
	if err != nil {
		return fmt.Errorf("create coordinationv1 client: %w", err)
	}
	// Create the leader election lock.
	rlock, err := resourcelock.New(
		"leases",
		opts.Namespace,
		LeaderElectionID,
		corev1client,
		coordinationClient,
		resourcelock.ResourceLockConfig{
			Identity:      opts.NodeID,
			EventRecorder: p,
		},
	)
	if err != nil {
		return fmt.Errorf("create leader election resource lock: %w", err)
	}
	// Create the leader elector.
	p.leaders, err = leaderelection.NewLeaderElector(leaderelection.LeaderElectionConfig{
		Name:          "webmesh-storage-leader",
		Lock:          rlock,
		LeaseDuration: opts.LeaderElectionLeaseDuration,
		RenewDeadline: opts.LeaderElectionRenewDeadline,
		RetryPeriod:   opts.LeaderElectionRetryPeriod,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(ctx context.Context) {
				p.log.Info("Acquired leader lease")
			},
			OnStoppedLeading: func() {
				p.log.Info("Lost leader lease")
			},
			OnNewLeader: func(identity string) {
				p.log.Info("New leader elected", "identity", identity)
			},
		},
		ReleaseOnCancel: true,
	})
	if err != nil {
		return fmt.Errorf("create leader elector: %w", err)
	}
	return nil
}

// Start starts the provider.
func (p *Provider) Start(ctx context.Context) error {
	return p.StartManaged(ctx)
}

// StartUnmanaged starts the provider assuming it controls the controller manager.
func (p *Provider) StartManaged(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.started.Load() {
		return errors.ErrStarted
	}
	defer p.started.Store(true)
	ctx, p.stop = context.WithCancel(ctx)
	// Start the controller manager
	go func() {
		p.log.Info("Starting controller manager and storage provider")
		if err := p.mgr.Start(ctx); err != nil {
			p.log.Error(err, "Problem running controller manager")
			p.errc <- fmt.Errorf("run controller manager: %w", err)
		}
	}()
	// Start the leader elector
	if !p.consensus.isObserver {
		go p.RunLeaderElection(ctx)
	}
	return nil
}

// StartUnmanaged starts the provider assuming it does not control the controller manager.
func (p *Provider) StartUnmanaged(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.started.Load() {
		return errors.ErrStarted
	}
	defer p.started.Store(true)
	ctx, p.stop = context.WithCancel(ctx)
	if !p.consensus.isObserver {
		go p.RunLeaderElection(ctx)
	}
	return nil
}

// RunLeaderElection runs the leader election loop.
func (p *Provider) RunLeaderElection(ctx context.Context) {
	// Start the leader elector
	p.log.Info("Starting leader election")
	for {
		p.leaders.Run(ctx)
		select {
		case <-ctx.Done():
			return
		default:
			p.log.Info("Leader election lease lost, restarting election")
		}
	}
}

// MeshStorage returns the underlying MeshStorage instance. The provider does not
// need to guarantee consistency on read operations.
func (p *Provider) MeshStorage() storage.MeshStorage {
	return p.storage
}

// MeshDB returns the underlying MeshDB instance. The provider does not
// need to guarantee consistency on read operations.
func (p *Provider) MeshDB() storage.MeshDB {
	return p.db
}

// Datastore returns the underlying datastore instance. This is useful
// for embedding the provider in other components.
func (p *Provider) Datastore() *database.Database {
	return p.datastore
}

// Consensus returns the underlying Consensus instance.
func (p *Provider) Consensus() storage.Consensus {
	return p.consensus
}

// Bootstrap should bootstrap the provider for first-time usage.
func (p *Provider) Bootstrap(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.started.Load() {
		return errors.ErrClosed
	}
	// Check if peers exist already
	peers, err := p.consensus.getPeers(ctx)
	if err != nil {
		return err
	}
	if len(peers) > 0 {
		// Peers exist, we are already bootstrapped
		return errors.ErrAlreadyBootstrapped
	}
	// We create ourselves as a voter.
	return p.consensus.AddVoter(ctx, types.StoragePeer{StoragePeer: &v1.StoragePeer{
		Id:            p.NodeID,
		Address:       fmt.Sprintf("%s:%d", p.NodeID, p.lport),
		ClusterStatus: v1.ClusterStatus_CLUSTER_LEADER,
	}})
}

// Status returns the status of the storage provider.
func (p *Provider) Status() *v1.StorageStatus {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.started.Load() {
		// This is a special case currently where we just return ourself.
		return &v1.StorageStatus{
			IsWritable:    false,
			ClusterStatus: v1.ClusterStatus_CLUSTER_NODE,
			Peers: []*v1.StoragePeer{
				{
					Id:            p.NodeID,
					Address:       p.laddr.String(),
					ClusterStatus: v1.ClusterStatus_CLUSTER_NODE,
				},
			},
			Message: errors.ErrClosed.Error(),
		}
	}
	peers, err := p.consensus.GetPeers(context.Background())
	if err != nil {
		p.log.Error(err, "Failed to get peers")
	}
	// If the peer list doesn't contain us its the same as the special case above.
	if !p.consensus.containsPeer(peers, p.NodeID) {
		return &v1.StorageStatus{
			IsWritable:    false,
			ClusterStatus: v1.ClusterStatus_CLUSTER_NODE,
			Peers: []*v1.StoragePeer{
				{
					Id:            p.NodeID,
					Address:       p.laddr.String(),
					ClusterStatus: v1.ClusterStatus_CLUSTER_NODE,
				},
			},
			Message: errors.ErrNotVoter.Error(),
		}
	}
	return &v1.StorageStatus{
		IsWritable: p.leaders.IsLeader(),
		ClusterStatus: func() v1.ClusterStatus {
			if p.leaders.IsLeader() {
				return v1.ClusterStatus_CLUSTER_LEADER
			}
			return v1.ClusterStatus_CLUSTER_VOTER
		}(),
		Peers: func() []*v1.StoragePeer {
			var out []*v1.StoragePeer
			for _, peer := range peers {
				pr := peer
				if peer.Id == p.NodeID && p.consensus.IsLeader() {
					pr.ClusterStatus = v1.ClusterStatus_CLUSTER_LEADER
				}
				out = append(out, pr.StoragePeer)
			}
			return out
		}(),
		Message: func() string {
			if err != nil {
				return err.Error()
			}
			return ""
		}(),
	}
}

// ListenPort should return the TCP port that the storage provider is listening on.
func (p *Provider) ListenPort() uint16 {
	return p.lport
}

// Close closes the provider.
func (p *Provider) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.started.Load() {
		return errors.ErrClosed
	}
	p.subsmu.Lock()
	defer p.subsmu.Unlock()
	for subID, sub := range p.subs {
		sub.cancel()
		delete(p.subs, subID)
	}
	p.stop()
	return nil
}

const inClusterNamespacePath = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"

func getInClusterNamespace() (string, error) {
	// Check whether the namespace file exists.
	// If not, we are not running in cluster so can't guess the namespace.
	if _, err := os.Stat(inClusterNamespacePath); os.IsNotExist(err) {
		return "", fmt.Errorf("not running in-cluster, please specify LeaderElectionNamespace")
	} else if err != nil {
		return "", fmt.Errorf("error checking namespace file: %w", err)
	}

	// Load the namespace file and return its content
	namespace, err := os.ReadFile(inClusterNamespacePath)
	if err != nil {
		return "", fmt.Errorf("error reading namespace file: %w", err)
	}
	return string(namespace), nil
}
