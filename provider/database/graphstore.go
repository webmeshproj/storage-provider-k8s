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
	"crypto/sha1"
	"fmt"
	"sync"

	"github.com/dominikbraun/graph"
	"github.com/google/uuid"
	"github.com/webmeshproj/webmesh/pkg/crypto"
	"github.com/webmeshproj/webmesh/pkg/storage"
	"github.com/webmeshproj/webmesh/pkg/storage/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	storagev1 "github.com/webmeshproj/storage-provider-k8s/api/storage/v1"
	"github.com/webmeshproj/storage-provider-k8s/provider/util"
)

// Ensure we implement the interface.
var _ types.PeerGraphStore = &GraphStore{}

//+kubebuilder:rbac:groups=storage.webmesh.io,resources=meshedges;peers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=storage.webmesh.io,resources=meshedges/status;peers/status,verbs=get;update;patch

// GraphStore implements the PeerGraphStore interface.
type GraphStore struct {
	cli       client.Client
	namespace string
	subs      map[string]*subscription
	submu     sync.RWMutex
}

type subscription struct {
	ctx    context.Context
	fn     storage.PeerSubscribeFunc
	cancel context.CancelFunc
}

// NewGraphStore returns a new GraphStore instance.
func NewGraphStore(cli client.Client, namespace string) *GraphStore {
	return &GraphStore{
		cli:       cli,
		namespace: namespace,
	}
}

// TruncateNodeID truncates a node ID to 63 characters. This is necessary because
// Kubernetes labels are limited to 63 characters.
func TruncateNodeID(id types.NodeID) string {
	return types.TruncateIDTo(id.String(), 63)
}

// SumKey sums the key into a compatible label value.
func SumKey(key crypto.PublicKey) (string, error) {
	encoded, err := key.Encode()
	if err != nil {
		return "", err
	}
	return HashEncodedKey(encoded)
}

// HashEncodedKey hashes the encoded key into a compatible label value.
func HashEncodedKey(encoded string) (string, error) {
	h := sha1.New()
	_, err := h.Write([]byte(encoded))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// HashNodeID hashed a node ID into a compatible kubernetes object name.
func HashNodeID(id types.NodeID) (string, error) {
	h := sha1.New()
	_, err := h.Write([]byte(id.String()))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// HashEdge hashes the edge into a compatible kubernetes object name.
func HashEdge(source, target types.NodeID) (string, error) {
	h := sha1.New()
	_, err := h.Write([]byte(source.String() + "-" + target.String()))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// AddVertex should add the given vertex with the given hash value and vertex properties to the
// graph. If the vertex already exists, it is up to you whether ErrVertexAlreadyExists or no
// error should be returned.
func (g *GraphStore) AddVertex(nodeID types.NodeID, node types.MeshNode, props graph.VertexProperties) error {
	hashedKey, err := HashEncodedKey(node.GetPublicKey())
	if err != nil {
		return err
	}
	name, err := HashNodeID(nodeID)
	if err != nil {
		return err
	}
	var peer storagev1.Peer
	peer.TypeMeta = metav1.TypeMeta{
		APIVersion: storagev1.GroupVersion.String(),
		Kind:       "Peer",
	}
	peer.ObjectMeta = metav1.ObjectMeta{
		Namespace: g.namespace,
		Name:      name,
		Labels: map[string]string{
			storagev1.PublicKeyLabel: hashedKey,
			storagev1.NodeIDLabel:    TruncateNodeID(nodeID),
		},
	}
	peer.Spec.Node = node
	return util.PatchObject(context.Background(), g.cli, &peer)
}

// Vertex should return the vertex and vertex properties with the given hash value. If the
// vertex doesn't exist, ErrVertexNotFound should be returned.
func (g *GraphStore) Vertex(nodeID types.NodeID) (node types.MeshNode, props graph.VertexProperties, err error) {
	var peers storagev1.PeerList
	err = g.cli.List(context.Background(), &peers, client.MatchingLabels{
		storagev1.NodeIDLabel: TruncateNodeID(nodeID),
	})
	if err != nil {
		if client.IgnoreNotFound(err) == nil {
			return types.MeshNode{}, graph.VertexProperties{}, graph.ErrVertexNotFound
		}
		return types.MeshNode{}, graph.VertexProperties{}, err
	}
	if len(peers.Items) == 0 {
		return types.MeshNode{}, graph.VertexProperties{}, graph.ErrVertexNotFound
	}
	return peers.Items[0].Spec.Node, graph.VertexProperties{}, nil
}

// RemoveVertex should remove the vertex with the given hash value.
func (g *GraphStore) RemoveVertex(nodeID types.NodeID) error {
	ctx := context.Background()
	name, err := HashNodeID(nodeID)
	if err != nil {
		return err
	}
	err = g.cli.Delete(ctx, &storagev1.Peer{
		TypeMeta: metav1.TypeMeta{
			APIVersion: storagev1.GroupVersion.String(),
			Kind:       "Peer",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: g.namespace,
			Name:      name,
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

// ListVertices should return all vertices in the graph in a slice.
func (g *GraphStore) ListVertices() ([]types.NodeID, error) {
	var peerlist storagev1.PeerList
	err := g.cli.List(context.Background(), &peerlist, client.InNamespace(g.namespace))
	if err != nil {
		if client.IgnoreNotFound(err) == nil {
			return nil, nil
		}
		return nil, err
	}
	var vertices []types.NodeID
	for _, peer := range peerlist.Items {
		vertices = append(vertices, peer.Spec.Node.NodeID())
	}
	return vertices, nil
}

// VertexCount should return the number of vertices in the graph. This should be equal to the
// length of the slice returned by ListVertices.
func (g *GraphStore) VertexCount() (int, error) {
	verts, err := g.ListVertices()
	if err != nil {
		return 0, err
	}
	return len(verts), nil
}

// AddEdge should add an edge between the vertices with the given source and target hashes.
//
// If either vertex doesn't exit, ErrVertexNotFound should be returned for the respective
// vertex. If the edge already exists, ErrEdgeAlreadyExists should be returned.
func (g *GraphStore) AddEdge(sourceNode, targetNode types.NodeID, edge graph.Edge[types.NodeID]) error {
	name, err := HashEdge(sourceNode, targetNode)
	if err != nil {
		return err
	}
	var edg storagev1.MeshEdge
	edg.TypeMeta = metav1.TypeMeta{
		APIVersion: storagev1.GroupVersion.String(),
		Kind:       "MeshEdge",
	}
	edg.ObjectMeta = metav1.ObjectMeta{
		Namespace: g.namespace,
		Name:      name,
		Labels: map[string]string{
			storagev1.EdgeSourceLabel: TruncateNodeID(sourceNode),
			storagev1.EdgeTargetLabel: TruncateNodeID(targetNode),
		},
	}
	edg.Spec.MeshEdge = types.Edge(edge).ToMeshEdge(sourceNode, targetNode)
	return util.PatchObject(context.Background(), g.cli, &edg)
}

// UpdateEdge should update the edge between the given vertices with the data of the given
// Edge instance. If the edge doesn't exist, ErrEdgeNotFound should be returned.
func (g *GraphStore) UpdateEdge(sourceNode, targetNode types.NodeID, edge graph.Edge[types.NodeID]) error {
	name, err := HashEdge(sourceNode, targetNode)
	if err != nil {
		return err
	}
	var edg storagev1.MeshEdge
	edg.TypeMeta = metav1.TypeMeta{
		APIVersion: storagev1.GroupVersion.String(),
		Kind:       "MeshEdge",
	}
	edg.ObjectMeta = metav1.ObjectMeta{
		Namespace: g.namespace,
		Name:      name,
		Labels: map[string]string{
			storagev1.EdgeSourceLabel: TruncateNodeID(sourceNode),
			storagev1.EdgeTargetLabel: TruncateNodeID(targetNode),
		},
	}
	edg.Spec.MeshEdge = types.Edge(edge).ToMeshEdge(sourceNode, targetNode)
	return util.PatchObject(context.Background(), g.cli, &edg)
}

// RemoveEdge should remove the edge between the vertices with the given source and target
// hashes.
//
// If either vertex doesn't exist, it is up to you whether ErrVertexNotFound or no error should
// be returned. If the edge doesn't exist, it is up to you whether ErrEdgeNotFound or no error
// should be returned.
func (g *GraphStore) RemoveEdge(sourceNode, targetNode types.NodeID) error {
	name, err := HashEdge(sourceNode, targetNode)
	if err != nil {
		return err
	}
	err = g.cli.Delete(context.Background(), &storagev1.MeshEdge{
		TypeMeta: metav1.TypeMeta{
			APIVersion: storagev1.GroupVersion.String(),
			Kind:       "MeshEdge",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: g.namespace,
			Name:      name,
		},
	})
	return client.IgnoreNotFound(err)
}

// Edge should return the edge joining the vertices with the given hash values. It should
// exclusively look for an edge between the source and the target vertex, not vice versa. The
// graph implementation does this for undirected graphs itself.
//
// Note that unlike Graph.Edge, this function is supposed to return an Edge[K], i.e. an edge
// that only contains the vertex hashes instead of the vertices themselves.
//
// If the edge doesn't exist, ErrEdgeNotFound should be returned.
func (g *GraphStore) Edge(sourceNode, targetNode types.NodeID) (graph.Edge[types.NodeID], error) {
	var edgeList storagev1.MeshEdgeList
	err := g.cli.List(context.Background(), &edgeList, client.MatchingLabels{
		storagev1.EdgeSourceLabel: TruncateNodeID(sourceNode),
		storagev1.EdgeTargetLabel: TruncateNodeID(targetNode),
	}, client.InNamespace(g.namespace))
	if err != nil {
		if client.IgnoreNotFound(err) == nil {
			return graph.Edge[types.NodeID]{}, graph.ErrEdgeNotFound
		}
		return graph.Edge[types.NodeID]{}, err
	}
	if len(edgeList.Items) == 0 {
		return graph.Edge[types.NodeID]{}, graph.ErrEdgeNotFound
	}
	return edgeList.Items[0].Spec.MeshEdge.AsGraphEdge(), nil
}

// ListEdges should return all edges in the graph in a slice.
func (g *GraphStore) ListEdges() ([]graph.Edge[types.NodeID], error) {
	var edgeList storagev1.MeshEdgeList
	err := g.cli.List(context.Background(), &edgeList, client.InNamespace(g.namespace))
	if err != nil {
		if client.IgnoreNotFound(err) == nil {
			return nil, nil
		}
		return nil, err
	}
	var edges []graph.Edge[types.NodeID]
	for _, edge := range edgeList.Items {
		edges = append(edges, edge.Spec.MeshEdge.AsGraphEdge())
	}
	return edges, nil
}

// Subscribe subscribes to node changes.
func (g *GraphStore) Subscribe(ctx context.Context, fn storage.PeerSubscribeFunc) (context.CancelFunc, error) {
	g.submu.Lock()
	defer g.submu.Unlock()
	id, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(ctx)
	sub := &subscription{
		ctx:    ctx,
		fn:     fn,
		cancel: cancel,
	}
	if g.subs == nil {
		g.subs = make(map[string]*subscription)
	}
	g.subs[id.String()] = sub
	return sub.cancel, nil
}
