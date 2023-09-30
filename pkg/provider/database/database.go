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

	"github.com/webmeshproj/webmesh/pkg/storage"
	"github.com/webmeshproj/webmesh/pkg/storage/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	storagev1 "github.com/webmeshproj/storage-provider-k8s/api/storage/v1"
	"github.com/webmeshproj/storage-provider-k8s/pkg/manager"
)

// Ensure we implement the interface.
var _ storage.MeshDB = &Database{}

// Database is a MeshDB implementation using Kubernetes custom resources.
type Database struct {
	peers   *Peers
	graph   types.PeerGraph
	rbac    *RBAC
	state   *MeshState
	network *Networking
}

// New returns a new Database instance.
func New(mgr manager.Manager, namespace string) (*Database, error) {
	db := &Database{
		peers:   NewPeers(mgr.GetClient(), namespace),
		graph:   types.NewGraphWithStore(NewGraphStore(mgr.GetClient(), namespace)),
		rbac:    NewRBAC(mgr.GetClient(), namespace),
		state:   NewMeshState(mgr.GetClient(), namespace),
		network: NewNetworking(mgr.GetClient(), namespace),
	}
	err := ctrl.
		NewControllerManagedBy(mgr).
		// Register the main reconciler for the peer CRD.
		For(&storagev1.Peer{}).
		// Watch the Routes as well and queue reconciles for the related peer.
		Watches(&storagev1.Route{}, handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, o client.Object) []reconcile.Request {
			labels := o.GetLabels()
			if peerID, ok := labels[RouteNodeLabel]; ok {
				return []reconcile.Request{{NamespacedName: client.ObjectKey{Name: peerID, Namespace: namespace}}}
			}
			return nil
		})).
		// Watch edges as well and queue reconciles for the related peers.
		Watches(&storagev1.MeshEdge{}, handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, o client.Object) []reconcile.Request {
			labels := o.GetLabels()
			sourceID, ok := labels[EdgeSourceLabel]
			var out []reconcile.Request
			if ok {
				out = append(out, reconcile.Request{NamespacedName: client.ObjectKey{Name: sourceID, Namespace: namespace}})
			}
			targetID, ok := labels[EdgeTargetLabel]
			if ok {
				out = append(out, reconcile.Request{NamespacedName: client.ObjectKey{Name: targetID, Namespace: namespace}})
			}
			return out
		})).
		Complete(db.peers)
	if err != nil {
		return nil, fmt.Errorf("register controller: %w", err)
	}
	return db, nil
}

// Peers returns the interface for managing nodes in the mesh.
func (db *Database) Peers() storage.Peers {
	return db.peers
}

// PeerGraph returns the interface for querying the peer graph.
func (db *Database) PeerGraph() types.PeerGraph {
	return db.graph
}

// RBAC returns the interface for managing RBAC policies in the mesh.
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
