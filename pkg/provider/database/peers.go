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

package database

import (
	"context"

	"github.com/webmeshproj/webmesh/pkg/crypto"
	"github.com/webmeshproj/webmesh/pkg/storage"
	"github.com/webmeshproj/webmesh/pkg/storage/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Ensure we implement the interface.
var _ storage.Peers = &Peers{}

// Peers implements the Peers interface.
type Peers struct {
	cli       client.Client
	namespace string
}

// NewPeers returns a new Peers instance.
func NewPeers(cli client.Client, namespace string) *Peers {
	return &Peers{
		cli:       cli,
		namespace: namespace,
	}
}

// PublicKeyLabel is the label used to store the public key.
const PublicKeyLabel = "webmesh.io/public-key"

// Put creates or updates a node.
func (p *Peers) Put(ctx context.Context, n types.MeshNode) error {
	return nil
}

// Get gets a node by ID.
func (p *Peers) Get(ctx context.Context, id types.NodeID) (types.MeshNode, error) {
	return types.MeshNode{}, nil
}

// GetByPubKey gets a node by their public key.
func (p *Peers) GetByPubKey(ctx context.Context, key crypto.PublicKey) (types.MeshNode, error) {
	return types.MeshNode{}, nil
}

// Delete deletes a node.
func (p *Peers) Delete(ctx context.Context, id types.NodeID) error {
	return nil
}

// List lists all nodes.
func (p *Peers) List(ctx context.Context, filters ...storage.PeerFilter) ([]types.MeshNode, error) {
	return nil, nil
}

// ListIDs lists all node IDs.
func (p *Peers) ListIDs(ctx context.Context) ([]types.NodeID, error) {
	return nil, nil
}

// Subscribe subscribes to node changes.
func (p *Peers) Subscribe(ctx context.Context, fn storage.PeerSubscribeFunc) (context.CancelFunc, error) {
	return func() {}, nil
}

// AddEdge adds an edge between two nodes.
func (p *Peers) PutEdge(ctx context.Context, edge types.MeshEdge) error {
	return nil
}

// GetEdge gets an edge between two nodes.
func (p *Peers) GetEdge(ctx context.Context, from, to types.NodeID) (types.MeshEdge, error) {
	return types.MeshEdge{}, nil
}

// RemoveEdge removes an edge between two nodes.
func (p *Peers) RemoveEdge(ctx context.Context, from, to types.NodeID) error {
	return nil
}
