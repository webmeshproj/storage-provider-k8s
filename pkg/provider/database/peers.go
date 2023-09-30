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
	"fmt"

	"github.com/webmeshproj/webmesh/pkg/crypto"
	"github.com/webmeshproj/webmesh/pkg/storage"
	"github.com/webmeshproj/webmesh/pkg/storage/errors"
	"github.com/webmeshproj/webmesh/pkg/storage/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	storagev1 "github.com/webmeshproj/storage-provider-k8s/api/storage/v1"
	"github.com/webmeshproj/storage-provider-k8s/pkg/provider/util"
)

// Ensure we implement the interface.
var _ storage.Peers = &Peers{}

//+kubebuilder:rbac:groups=storage.webmesh.io,resources=peers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=storage.webmesh.io,resources=peers/status,verbs=get;update;patch

// Peers implements the Peers interface.
type Peers struct {
	cli       client.Client
	graph     types.PeerGraph
	namespace string
}

// NewPeers returns a new Peers instance.
func NewPeers(cli client.Client, namespace string) *Peers {
	return &Peers{
		cli:       cli,
		graph:     types.NewGraphWithStore(NewGraphStore(cli, namespace)),
		namespace: namespace,
	}
}

// PublicKeyLabel is the label used to store the public key.
const PublicKeyLabel = "webmesh.io/public-key"

// Put creates or updates a node.
func (p *Peers) Put(ctx context.Context, n types.MeshNode) error {
	var peer storagev1.Peer
	peer.TypeMeta = metav1.TypeMeta{
		APIVersion: storagev1.GroupVersion.String(),
		Kind:       "Peer",
	}
	peer.ObjectMeta = metav1.ObjectMeta{
		Namespace: p.namespace,
		Name:      n.GetId(),
		Labels: map[string]string{
			PublicKeyLabel: n.GetPublicKey(),
		},
	}
	peer.Spec.Node = n
	return util.PatchObject(ctx, p.cli, &peer)
}

// Get gets a node by ID.
func (p *Peers) Get(ctx context.Context, id types.NodeID) (types.MeshNode, error) {
	var peer storagev1.Peer
	err := p.cli.Get(ctx, client.ObjectKey{
		Namespace: p.namespace,
		Name:      id.String(),
	}, &peer)
	if err != nil {
		if client.IgnoreNotFound(err) == nil {
			return types.MeshNode{}, errors.ErrNodeNotFound
		}
		return types.MeshNode{}, err
	}
	return peer.Spec.Node, nil
}

// GetByPubKey gets a node by their public key.
func (p *Peers) GetByPubKey(ctx context.Context, key crypto.PublicKey) (types.MeshNode, error) {
	encoded, err := key.Encode()
	if err != nil {
		return types.MeshNode{}, err
	}
	var peerlist storagev1.PeerList
	err = p.cli.List(ctx, &peerlist, client.MatchingLabels{
		PublicKeyLabel: encoded,
	}, client.InNamespace(p.namespace))
	if err != nil {
		if client.IgnoreNotFound(err) == nil {
			return types.MeshNode{}, errors.ErrNodeNotFound
		}
		return types.MeshNode{}, err
	}
	if len(peerlist.Items) == 0 {
		return types.MeshNode{}, errors.ErrNodeNotFound
	}
	return peerlist.Items[0].Spec.Node, nil
}

// Delete deletes a node.
func (p *Peers) Delete(ctx context.Context, id types.NodeID) error {
	// First check for and remove any edges
	edges, err := p.graph.Edges()
	if err != nil {
		return fmt.Errorf("get edges: %w", err)
	}
	for _, edge := range edges {
		if edge.Source.String() == id.String() || edge.Target.String() == id.String() {
			err = p.graph.RemoveEdge(edge.Source, edge.Target)
			if err != nil {
				return fmt.Errorf("remove edge: %w", err)
			}
		}
	}
	err = p.cli.Delete(ctx, &storagev1.Peer{
		TypeMeta: metav1.TypeMeta{
			APIVersion: storagev1.GroupVersion.String(),
			Kind:       "Peer",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: p.namespace,
			Name:      id.String(),
		},
	})
	if err != nil {
		if client.IgnoreNotFound(err) == nil {
			return nil
		}
		return err
	}
	return nil
}

// List lists all nodes.
func (p *Peers) List(ctx context.Context, filters ...storage.PeerFilter) ([]types.MeshNode, error) {
	var peerlist storagev1.PeerList
	err := p.cli.List(ctx, &peerlist, client.InNamespace(p.namespace))
	if err != nil {
		if client.IgnoreNotFound(err) == nil {
			return nil, nil
		}
	}
	out := make([]types.MeshNode, len(peerlist.Items))
	for i, peer := range peerlist.Items {
		out[i] = peer.Spec.Node
	}
	return storage.PeerFilters(filters).Filter(out), nil
}

// ListIDs lists all node IDs.
func (p *Peers) ListIDs(ctx context.Context) ([]types.NodeID, error) {
	var peerlist storagev1.PeerList
	err := p.cli.List(ctx, &peerlist, client.InNamespace(p.namespace))
	if err != nil {
		if client.IgnoreNotFound(err) == nil {
			return nil, nil
		}
	}
	out := make([]types.NodeID, len(peerlist.Items))
	for i, peer := range peerlist.Items {
		out[i] = peer.Spec.Node.NodeID()
	}
	return out, nil
}

// Subscribe subscribes to node changes.
func (p *Peers) Subscribe(ctx context.Context, fn storage.PeerSubscribeFunc) (context.CancelFunc, error) {
	return func() {}, fmt.Errorf("TODO")
}

// AddEdge adds an edge between two nodes.
func (p *Peers) PutEdge(ctx context.Context, edge types.MeshEdge) error {
	var edg storagev1.MeshEdge
	edg.TypeMeta = metav1.TypeMeta{
		APIVersion: storagev1.GroupVersion.String(),
		Kind:       "MeshEdge",
	}
	edg.ObjectMeta = metav1.ObjectMeta{
		Namespace: p.namespace,
		Name:      edge.SourceID().String() + "_" + edge.TargetID().String(),
		Labels: map[string]string{
			EdgeSourceLabel: edge.SourceID().String(),
			EdgeTargetLabel: edge.TargetID().String(),
		},
	}
	edg.Spec.MeshEdge = edge
	return util.PatchObject(ctx, p.cli, &edg)
}

// GetEdge gets an edge between two nodes.
func (p *Peers) GetEdge(ctx context.Context, from, to types.NodeID) (types.MeshEdge, error) {
	var edgeList storagev1.MeshEdgeList
	err := p.cli.List(ctx, &edgeList, client.MatchingLabels{
		EdgeSourceLabel: from.String(),
		EdgeTargetLabel: to.String(),
	}, client.InNamespace(p.namespace))
	if err != nil {
		if client.IgnoreNotFound(err) == nil {
			return types.MeshEdge{}, errors.ErrEdgeNotFound
		}
		return types.MeshEdge{}, err
	}
	if len(edgeList.Items) == 0 {
		return types.MeshEdge{}, errors.ErrEdgeNotFound
	}
	return edgeList.Items[0].Spec.MeshEdge, nil
}

// RemoveEdge removes an edge between two nodes.
func (p *Peers) RemoveEdge(ctx context.Context, from, to types.NodeID) error {
	err := p.cli.Delete(ctx, &storagev1.MeshEdge{
		TypeMeta: metav1.TypeMeta{
			APIVersion: storagev1.GroupVersion.String(),
			Kind:       "MeshEdge",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: p.namespace,
			Name:      from.String() + "_" + to.String(),
		},
	})
	return client.IgnoreNotFound(err)
}

func (p *Peers) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return ctrl.Result{}, nil
}
