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
	"github.com/webmeshproj/webmesh/pkg/storage"
	"github.com/webmeshproj/webmesh/pkg/storage/types"

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
func New(mgr manager.Manager) *Database {
	return &Database{
		peers:   NewPeers(mgr.GetClient()),
		graph:   types.NewGraphWithStore(NewGraphStore(mgr.GetClient())),
		rbac:    NewRBAC(mgr.GetClient()),
		state:   NewMeshState(mgr.GetClient()),
		network: NewNetworking(mgr.GetClient()),
	}
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