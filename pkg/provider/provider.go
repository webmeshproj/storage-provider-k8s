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
	"time"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
	v1 "github.com/webmeshproj/api/v1"
	"github.com/webmeshproj/webmesh/pkg/storage"
	corev1 "k8s.io/api/core/v1"
	coordinationv1client "k8s.io/client-go/kubernetes/typed/coordination/v1"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/handler"

	"github.com/webmeshproj/storage-provider-k8s/pkg/manager"
)

const (
	// LeaderElectionID is the name of the leader election lease.
	LeaderElectionID = "storage-provider-k8s-leader-election"
)

// Ensure we satisfy the provider interface.
var _ storage.Provider = &Provider{}

// Options are the options for configuring the provider.
type Options struct {
	// ListenAddr is the address to bind the webhook server to.
	ListenAddr string
	// MetricsAddr is the address to bind the metrics endpoint to.
	MetricsAddr string
	// ProbeAddr is the address to bind the health probe endpoint to.
	ProbeAddr string
	// ShutdownTimeout is the timeout for shutting down the provider.
	ShutdownTimeout time.Duration
	// LeaderElectionNamespace is the namespace to use for leader election.
	LeaderElectionNamespace string
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
	lport     uint16
	mgr       manager.Manager
	storage   *Storage
	consensus *Consensus
	leaders   *leaderelection.LeaderElector
	errc      chan error
	stop      context.CancelFunc
	log       logr.Logger
}

// New creates a new Provider.
func New(options Options) (*Provider, error) {
	p := &Provider{
		Options: options,
		errc:    make(chan error, 1),
		log:     ctrl.Log.WithName("storage-provider"),
	}
	p.storage = &Storage{Provider: p}
	p.consensus = &Consensus{Provider: p}
	laddr, err := net.ResolveTCPAddr("tcp", options.ListenAddr)
	if err != nil {
		return nil, fmt.Errorf("resolve listen address: %w", err)
	}
	p.lport = uint16(laddr.Port)
	if p.LeaderElectionNamespace == "" {
		var err error
		p.LeaderElectionNamespace, err = getInClusterNamespace()
		if err != nil {
			return nil, fmt.Errorf("get in-cluster namespace: %w", err)
		}
	}
	// Create the controller manager.
	p.mgr, err = manager.New(manager.Options{
		WebhookPort:     laddr.Port,
		MetricsAddr:     options.MetricsAddr,
		ProbeAddr:       options.ProbeAddr,
		ShutdownTimeout: options.ShutdownTimeout,
	})
	if err != nil {
		return nil, fmt.Errorf("create controller manager: %w", err)
	}
	// Register the reconciler with the manager.
	err = ctrl.
		NewControllerManagedBy(p.mgr).
		Watches(&corev1.Secret{}, handler.EnqueueRequestsFromMapFunc(p.enqueueObjectIfOwner)).
		Complete(p)
	if err != nil {
		return nil, fmt.Errorf("register controller: %w", err)
	}
	// Create clients for leader election
	cfg := rest.CopyConfig(p.mgr.GetConfig())
	corev1client, err := corev1client.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("create corev1 client: %w", err)
	}
	coordinationClient, err := coordinationv1client.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("create coordinationv1 client: %w", err)
	}
	// Leader id, needs to be unique
	id, err := os.Hostname()
	if err != nil {
		return nil, err
	}
	// Create the leader election lock.
	rlock, err := resourcelock.New(
		"leases",
		options.LeaderElectionNamespace,
		LeaderElectionID,
		corev1client,
		coordinationClient,
		resourcelock.ResourceLockConfig{
			Identity:      id + "_" + uuid.NewString(),
			EventRecorder: p,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("create leader election resource lock: %w", err)
	}
	// Create the leader elector.
	p.leaders, err = leaderelection.NewLeaderElector(leaderelection.LeaderElectionConfig{
		Name:          "webmesh-storage-leader",
		Lock:          rlock,
		LeaseDuration: options.LeaderElectionLeaseDuration,
		RenewDeadline: options.LeaderElectionRenewDeadline,
		RetryPeriod:   options.LeaderElectionRetryPeriod,
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
		return nil, fmt.Errorf("create leader elector: %w", err)
	}
	return p, nil
}

// Start starts the provider.
func (p *Provider) Start(ctx context.Context) error {
	ctx, p.stop = context.WithCancel(ctrl.SetupSignalHandler())
	// Start the controller manager
	go func() {
		p.log.Info("Starting controller manager and storage provider")
		if err := p.mgr.Start(ctx); err != nil {
			p.log.Error(err, "Problem running controller manager")
			p.errc <- fmt.Errorf("run controller manager: %w", err)
		}
	}()
	// Start the leader elector
	go func() {
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
	}()
	return nil
}

// MeshStorage returns the underlying MeshStorage instance. The provider does not
// need to guarantee consistency on read operations.
func (p *Provider) MeshStorage() storage.MeshStorage {
	return p.storage
}

// Consensus returns the underlying Consensus instance.
func (p *Provider) Consensus() storage.Consensus {
	return p.consensus
}

// Bootstrap should bootstrap the provider for first-time usage.
func (p *Provider) Bootstrap(context.Context) error {
	// No need to bootstrap on k8s.
	return nil
}

// Status returns the status of the storage provider.
func (p *Provider) Status() *v1.StorageStatus {
	return &v1.StorageStatus{}
}

// ListenPort should return the TCP port that the storage provider is listening on.
func (p *Provider) ListenPort() uint16 {
	return p.lport
}

// Close closes the provider.
func (p *Provider) Close() error {
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
