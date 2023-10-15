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

// Package database implements a MeshDB using Kubernetes custom resources.
package database

import (
	"context"
	"fmt"
	"net"
	"net/netip"

	v1 "github.com/webmeshproj/api/v1"
	"github.com/webmeshproj/webmesh/pkg/storage"
	"github.com/webmeshproj/webmesh/pkg/storage/errors"
	"github.com/webmeshproj/webmesh/pkg/storage/meshdb"
	"github.com/webmeshproj/webmesh/pkg/storage/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	storagev1 "github.com/webmeshproj/storage-provider-k8s/api/storage/v1"
	"github.com/webmeshproj/storage-provider-k8s/provider/manager"
)

// Ensure we implement the interface.
var _ storage.MeshDataStore = &Database{}

// Database is a MeshDB implementation using Kubernetes custom resources.
type Database struct {
	cli     client.Client
	laddr   *net.TCPAddr
	graph   *GraphStore
	rbac    *RBAC
	state   *MeshState
	network *Networking
}

// Options are the options for the database.
type Options struct {
	NodeID     types.NodeID
	Namespace  string
	ListenAddr *net.TCPAddr
}

// New returns a new MeshDB instance. It will create a new Database
// and then wrap it in a meshdb.MeshDB.
func New(mgr manager.Manager, opts Options) (storage.MeshDB, *Database, error) {
	db, err := NewDB(mgr, opts)
	if err != nil {
		return nil, nil, err
	}
	return meshdb.New(db), db, nil
}

// NewFromClient returns a database from the given client. It does not
// intialize any controllers.
func NewFromClient(cli client.Client, opts Options) *Database {
	return &Database{
		cli:     cli,
		laddr:   opts.ListenAddr,
		graph:   NewGraphStore(cli, opts.Namespace),
		rbac:    NewRBAC(cli, opts.Namespace),
		state:   NewMeshState(cli, opts.Namespace),
		network: NewNetworking(cli, opts.Namespace),
	}
}

// NewDB returns a new MeshDataStore instance.
func NewDB(mgr manager.Manager, opts Options) (*Database, error) {
	db := NewFromClient(mgr.GetClient(), opts)
	err := ctrl.
		NewControllerManagedBy(mgr).
		Named("meshdb-k8s").
		// Register the main reconciler for the peer CRD.
		For(&storagev1.Peer{}).
		// Watch the Routes as well and queue reconciles for the related peer.
		Watches(&storagev1.Route{}, handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, o client.Object) []reconcile.Request {
			labels := o.GetLabels()
			if peerID, ok := labels[RouteNodeLabel]; ok && peerID != "" {
				return []reconcile.Request{{NamespacedName: client.ObjectKey{Name: peerID, Namespace: opts.Namespace}}}
			}
			return nil
		})).
		// Watch edges as well and queue reconciles for the related peers.
		Watches(&storagev1.MeshEdge{}, handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, o client.Object) []reconcile.Request {
			labels := o.GetLabels()
			var out []reconcile.Request
			sourceID, ok := labels[storagev1.EdgeSourceLabel]
			if ok && sourceID != "" {
				out = append(out, reconcile.Request{NamespacedName: client.ObjectKey{Name: sourceID, Namespace: opts.Namespace}})
			}
			targetID, ok := labels[storagev1.EdgeTargetLabel]
			if ok && targetID != "" {
				out = append(out, reconcile.Request{NamespacedName: client.ObjectKey{Name: targetID, Namespace: opts.Namespace}})
			}
			return out
		})).
		Complete(db)
	if err != nil {
		return nil, fmt.Errorf("register controller: %w", err)
	}
	return db, nil
}

// GraphStore returns the interface for querying the peer graph.
func (db *Database) GraphStore() storage.GraphStore {
	return db.graph
}

// RBAC returns the interface for conditionmanaging RBAC policies in the mesh.
func (db *Database) RBAC() storage.RBAC {
	return db.rbac
}

// MeshState returns the interface for querying mesh state.
func (db *Database) MeshState() storage.MeshState {
	return db.state
}

// Networking returns the interface for managing networking in the mesh.
func (db *Database) Networking() storage.Networking {
	return db.network
}

// Close closes the database.
func (db *Database) Close() error {
	db.graph.submu.Lock()
	defer db.graph.submu.Unlock()
	for id, sub := range db.graph.subs {
		sub.cancel()
		delete(db.graph.subs, id)
	}
	return nil
}

// Reconcile is called for every update to a peer, route, or edge.
func (db *Database) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	db.graph.submu.RLock()
	defer db.graph.submu.RUnlock()
	var peer storagev1.Peer
	err := db.graph.cli.Get(ctx, req.NamespacedName, &peer)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			return ctrl.Result{}, fmt.Errorf("get peer: %w", err)
		}
		// We'll notify an empty peer
		peer.MeshNode = types.MeshNode{
			MeshNode: &v1.MeshNode{Id: req.Name},
		}
	}
	for id, sub := range db.graph.subs {
		select {
		case <-sub.ctx.Done():
			// Delete the subscription
			delete(db.graph.subs, id)
			continue
		default:
		}
		sub.fn([]types.MeshNode{peer.MeshNode})
	}
	return ctrl.Result{}, nil
}

// GetPeerByIP returns the peer with the given IP address.
func (db *Database) GetPeerByIPv4Addr(ctx context.Context, addr netip.Prefix) (types.MeshNode, error) {
	var peers storagev1.PeerList
	err := db.cli.List(context.Background(), &peers,
		client.InNamespace(db.graph.namespace),
		client.MatchingLabels{
			storagev1.NodeIPv4Label: HashLabelValue(addr.String()),
		},
	)
	if err != nil {
		return types.MeshNode{}, fmt.Errorf("list peers: %w", err)
	}
	if len(peers.Items) == 0 {
		return types.MeshNode{}, errors.ErrNodeNotFound
	}
	return peers.Items[0].MeshNode, nil
}

// GetPeerByIP returns the peer with the given IP address.
func (db *Database) GetPeerByIPv6Addr(ctx context.Context, addr netip.Prefix) (types.MeshNode, error) {
	var peers storagev1.PeerList
	err := db.cli.List(context.Background(), &peers,
		client.InNamespace(db.graph.namespace),
		client.MatchingLabels{
			storagev1.NodeIPv6Label: HashLabelValue(addr.String()),
		},
	)
	if err != nil {
		return types.MeshNode{}, fmt.Errorf("list peers: %w", err)
	}
	if len(peers.Items) == 0 {
		return types.MeshNode{}, errors.ErrNodeNotFound
	}
	return peers.Items[0].MeshNode, nil
}
