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

	"github.com/dominikbraun/graph"
	"github.com/webmeshproj/webmesh/pkg/crypto"
	"github.com/webmeshproj/webmesh/pkg/storage/errors"
	"github.com/webmeshproj/webmesh/pkg/storage/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	storagev1 "github.com/webmeshproj/storage-provider-k8s/api/storage/v1"
	"github.com/webmeshproj/storage-provider-k8s/provider/util"
)

// Ensure we implement the interface.
var _ types.PeerGraphStore = &GraphStore{}

//+kubebuilder:rbac:groups=storage.webmesh.io,resources=meshedges,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=storage.webmesh.io,resources=meshedges/status,verbs=get;update;patch

// GraphStore implements the PeerGraphStore interface.
type GraphStore struct {
	cli       client.Client
	namespace string
}

// NewGraphStore returns a new GraphStore instance.
func NewGraphStore(cli client.Client, namespace string) *GraphStore {
	return &GraphStore{
		cli:       cli,
		namespace: namespace,
	}
}

// AddVertex should add the given vertex with the given hash value and vertex properties to the
// graph. If the vertex already exists, it is up to you whether ErrVertexAlreadyExists or no
// error should be returned.
func (g *GraphStore) AddVertex(nodeID types.NodeID, node types.MeshNode, props graph.VertexProperties) error {
	// Check that the node is valid, this should be pushed up to the API server.
	if nodeID.IsEmpty() {
		return errors.ErrEmptyNodeID
	}
	if !nodeID.IsValid() {
		return fmt.Errorf("%w: %s", errors.ErrInvalidNodeID, nodeID)
	}
	if node.GetPublicKey() != "" {
		_, err := crypto.DecodePublicKey(node.GetPublicKey())
		if err != nil {
			return errors.ErrInvalidKey
		}
	}
	// Check if the vertex already exists.
	_, _, err := g.Vertex(nodeID)
	if err == nil {
		return graph.ErrVertexAlreadyExists
	}
	hashedKey, err := HashEncodedKey(node.GetPublicKey())
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
		Name:      nodeID.String(),
		Labels: map[string]string{
			storagev1.PublicKeyLabel: hashedKey,
		},
	}
	peer.Spec.Node = node
	return util.PatchObject(context.Background(), g.cli, &peer)
}

// Vertex should return the vertex and vertex properties with the given hash value. If the
// vertex doesn't exist, ErrVertexNotFound should be returned.
func (g *GraphStore) Vertex(nodeID types.NodeID) (node types.MeshNode, props graph.VertexProperties, err error) {
	if nodeID.IsEmpty() {
		err = errors.ErrEmptyNodeID
		return
	}
	if !nodeID.IsValid() {
		err = fmt.Errorf("%w: %s", errors.ErrInvalidNodeID, nodeID)
		return
	}
	var peer storagev1.Peer
	err = g.cli.Get(context.Background(), client.ObjectKey{
		Namespace: g.namespace,
		Name:      nodeID.String(),
	}, &peer)
	if err != nil {
		if client.IgnoreNotFound(err) == nil {
			return types.MeshNode{}, graph.VertexProperties{}, graph.ErrVertexNotFound
		}
		return types.MeshNode{}, graph.VertexProperties{}, err
	}
	return peer.Spec.Node, graph.VertexProperties{}, nil
}

// RemoveVertex should remove the vertex with the given hash value. If the vertex doesn't
// exist, ErrVertexNotFound should be returned. If the vertex has edges to other vertices,
// ErrVertexHasEdges should be returned.
func (g *GraphStore) RemoveVertex(nodeID types.NodeID) error {
	if nodeID.IsEmpty() {
		return errors.ErrEmptyNodeID
	}
	if !nodeID.IsValid() {
		return fmt.Errorf("%w: %s", errors.ErrInvalidNodeID, nodeID)
	}
	ctx := context.Background()
	// First check if the vertex exists.
	_, _, err := g.Vertex(nodeID)
	if err != nil {
		return err
	}
	// Check if it has edges
	var found bool
	var edgelist storagev1.MeshEdgeList
	err = g.cli.List(ctx, &edgelist, client.MatchingLabels{
		storagev1.EdgeSourceLabel: nodeID.String(),
	}, client.InNamespace(g.namespace))
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			return err
		}
	} else {
		if len(edgelist.Items) > 0 {
			found = true
		}
	}
	err = g.cli.List(ctx, &edgelist, client.MatchingLabels{
		storagev1.EdgeTargetLabel: nodeID.String(),
	}, client.InNamespace(g.namespace))
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			return err
		}
	} else {
		if len(edgelist.Items) > 0 {
			found = true
		}
	}
	if found {
		return graph.ErrVertexHasEdges
	}
	err = g.cli.Delete(ctx, &storagev1.Peer{
		TypeMeta: metav1.TypeMeta{
			APIVersion: storagev1.GroupVersion.String(),
			Kind:       "Peer",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: g.namespace,
			Name:      nodeID.String(),
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
	// We diverge from the suggested implementation and only check that one of the nodes
	// exists. This is so joiners can add edges to nodes that are not yet in the graph.
	// If this ends up causing problems, we can change it.
	if sourceNode.IsEmpty() || targetNode.IsEmpty() {
		return errors.ErrEmptyNodeID
	}
	if !sourceNode.IsValid() || !targetNode.IsValid() {
		return fmt.Errorf("%w: %s and %s", errors.ErrInvalidNodeID, sourceNode, targetNode)
	}
	var eitherVertexExists bool
	_, _, err := g.Vertex(sourceNode)
	if err != nil && err != graph.ErrVertexNotFound {
		return err
	} else if err == nil {
		eitherVertexExists = true
	}
	_, _, err = g.Vertex(targetNode)
	if err != nil && err != graph.ErrVertexNotFound {
		return err
	} else if err == nil {
		eitherVertexExists = true
	}
	if !eitherVertexExists {
		return graph.ErrVertexNotFound
	}
	// Check if the edge already exists.
	_, err = g.Edge(sourceNode, targetNode)
	if err == nil {
		return graph.ErrEdgeAlreadyExists
	}
	var edg storagev1.MeshEdge
	edg.TypeMeta = metav1.TypeMeta{
		APIVersion: storagev1.GroupVersion.String(),
		Kind:       "MeshEdge",
	}
	edg.ObjectMeta = metav1.ObjectMeta{
		Namespace: g.namespace,
		Name:      sourceNode.String() + "-" + targetNode.String(),
		Labels: map[string]string{
			storagev1.EdgeSourceLabel: sourceNode.String(),
			storagev1.EdgeTargetLabel: targetNode.String(),
		},
	}
	edg.Spec.MeshEdge = types.Edge(edge).ToMeshEdge(sourceNode, targetNode)
	return util.PatchObject(context.Background(), g.cli, &edg)
}

// UpdateEdge should update the edge between the given vertices with the data of the given
// Edge instance. If the edge doesn't exist, ErrEdgeNotFound should be returned.
func (g *GraphStore) UpdateEdge(sourceNode, targetNode types.NodeID, edge graph.Edge[types.NodeID]) error {
	// Check that the edge exists
	if sourceNode.IsEmpty() || targetNode.IsEmpty() {
		return fmt.Errorf("node ID must not be empty")
	}
	if !sourceNode.IsValid() || !targetNode.IsValid() {
		return fmt.Errorf("%w: %s and %s", errors.ErrInvalidNodeID, sourceNode, targetNode)
	}
	_, err := g.Edge(sourceNode, targetNode)
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
		Name:      sourceNode.String() + "-" + targetNode.String(),
		Labels: map[string]string{
			storagev1.EdgeSourceLabel: sourceNode.String(),
			storagev1.EdgeTargetLabel: targetNode.String(),
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
	if sourceNode.IsEmpty() || targetNode.IsEmpty() {
		return fmt.Errorf("node ID must not be empty")
	}
	if !sourceNode.IsValid() || !targetNode.IsValid() {
		return fmt.Errorf("%w: %s and %s", errors.ErrInvalidNodeID, sourceNode, targetNode)
	}
	err := g.cli.Delete(context.Background(), &storagev1.MeshEdge{
		TypeMeta: metav1.TypeMeta{
			APIVersion: storagev1.GroupVersion.String(),
			Kind:       "MeshEdge",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: g.namespace,
			Name:      sourceNode.String() + "-" + targetNode.String(),
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
	if sourceNode.IsEmpty() || targetNode.IsEmpty() {
		return graph.Edge[types.NodeID]{}, fmt.Errorf("node ID must not be empty")
	}
	if !sourceNode.IsValid() || !targetNode.IsValid() {
		return graph.Edge[types.NodeID]{}, fmt.Errorf("%w: %s and %s", errors.ErrInvalidNodeID, sourceNode, targetNode)
	}
	var edgeList storagev1.MeshEdgeList
	err := g.cli.List(context.Background(), &edgeList, client.MatchingLabels{
		storagev1.EdgeSourceLabel: sourceNode.String(),
		storagev1.EdgeTargetLabel: targetNode.String(),
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
