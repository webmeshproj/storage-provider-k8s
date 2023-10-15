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

package provider

import (
	"context"
	"crypto/sha1"
	"fmt"
	"sync"

	v1 "github.com/webmeshproj/api/v1"
	"github.com/webmeshproj/webmesh/pkg/storage"
	"github.com/webmeshproj/webmesh/pkg/storage/errors"
	"github.com/webmeshproj/webmesh/pkg/storage/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	storagev1 "github.com/webmeshproj/storage-provider-k8s/api/storage/v1"
	"github.com/webmeshproj/storage-provider-k8s/provider/util"
)

// Ensure we satisfy the consensus interface.
var _ storage.Consensus = &Consensus{}

//+kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=storage.webmesh.io,resources=storagepeers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=storage.webmesh.io,resources=storagepeers/status,verbs=get;update;patch

const (
	// ConsensusTraceVLevel is the trace level for the consensus package.
	ConsensusTraceVLevel = 2
)

// HashNodeID hashed a node ID into a compatible kubernetes object name.
func HashID(id string) string {
	h := sha1.New()
	h.Write([]byte(id))
	return fmt.Sprintf("%x", h.Sum(nil))
}

// Consensus is the consensus interface for the storage provider.
type Consensus struct {
	*Provider
	isObserver bool
	self       types.StoragePeer
	mu         sync.Mutex
}

func (c *Consensus) trace(ctx context.Context, msg string, args ...interface{}) {
	c.log.V(ConsensusTraceVLevel).WithName("storage-consensus").Info(
		msg, append(args, "namespace", c.Namespace, "node-id", c.NodeID)...,
	)
}

// IsLeader returns true if the node is the leader of the storage group.
func (c *Consensus) IsLeader() bool {
	if c.isObserver {
		return false
	}
	return c.leaders.IsLeader()
}

// IsMember returns true if the node is a member of the storage group.
func (c *Consensus) IsMember() bool {
	return true
}

// StepDown is a no-op for now, but hooks can potentially be given to the
// caller to perform some action when the node steps down.
func (c *Consensus) StepDown(ctx context.Context) error {
	return nil
}

// GetPeer returns the peers of the storage group.
func (c *Consensus) GetPeer(ctx context.Context, id string) (types.StoragePeer, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.trace(ctx, "Getting peer", "id", id)
	peers, err := c.getPeers(ctx)
	if err != nil {
		return types.StoragePeer{}, fmt.Errorf("get peers: %w", err)
	}
	c.trace(ctx, "Listed peers", "peers", peers)
	for _, p := range peers {
		if p.GetId() == id {
			return p.StoragePeer, nil
		}
	}
	return types.StoragePeer{}, errors.ErrNodeNotFound
}

// GetPeers returns the peers of the storage group.
func (c *Consensus) GetPeers(ctx context.Context) ([]types.StoragePeer, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.trace(ctx, "Listing peers")
	peers, err := c.getPeers(ctx)
	if err != nil {
		return nil, fmt.Errorf("get peers: %w", err)
	}
	c.trace(ctx, "Listed peers", "peers", peers)
	var sps []types.StoragePeer
	for _, p := range peers {
		sps = append(sps, p.StoragePeer)
	}
	return sps, nil
}

// GetLeader returns the leader of the storage group.
func (c *Consensus) GetLeader(ctx context.Context) (types.StoragePeer, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.IsLeader() {
		// Fast path return ourself if we have it stored.
		c.trace(ctx, "Returning self as leader")
		if c.self.StoragePeer != nil {
			return c.self, nil
		}
	}
	c.trace(ctx, "Getting leader from peer list")
	peers, err := c.getPeers(ctx)
	if err != nil {
		return types.StoragePeer{}, fmt.Errorf("get peers: %w", err)
	}
	c.trace(ctx, "Got peers list", "peers", peers)
	for _, p := range peers {
		peer := p.StoragePeer
		if !c.isObserver && c.leaders.GetLeader() == peer.GetId() && c.IsLeader() {
			// Store ourself as the leader.
			c.trace(ctx, "Storing and returning self as leader")
			c.self = peer
			return peer, nil
		} else if !c.isObserver && peer.GetClusterStatus() == v1.ClusterStatus_CLUSTER_VOTER {
			// We consider any voting node a leader when we are an observer.
			return peer, nil
		}
		if peer.ClusterStatus == v1.ClusterStatus_CLUSTER_LEADER {
			return peer, nil
		}
	}
	c.trace(ctx, "No leader found")
	return types.StoragePeer{}, errors.ErrNoLeader
}

// AddVoter adds a voter to the consensus group.
func (c *Consensus) AddVoter(ctx context.Context, peer types.StoragePeer) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.isObserver {
		return errors.ErrNotStorageNode
	}
	peer.ClusterStatus = v1.ClusterStatus_CLUSTER_VOTER
	c.trace(ctx, "Adding voter", "peer", peer)
	stpeer := storagev1.StoragePeer{
		TypeMeta: storagev1.StoragePeerTypeMeta,
		ObjectMeta: metav1.ObjectMeta{
			Name:      HashID(peer.GetId()),
			Namespace: c.Namespace,
			Labels: map[string]string{
				storagev1.NodeIDLabel: peer.GetId(),
			},
		},
		StoragePeer: peer,
	}
	return util.PatchObject(ctx, c.mgr.GetClient(), &stpeer)
}

// AddObserver adds an observer to the consensus group.
func (c *Consensus) AddObserver(ctx context.Context, peer types.StoragePeer) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.isObserver {
		return errors.ErrNotStorageNode
	}
	peer.ClusterStatus = v1.ClusterStatus_CLUSTER_OBSERVER
	c.trace(ctx, "Adding observer", "peer", peer)
	stpeer := storagev1.StoragePeer{
		TypeMeta: storagev1.StoragePeerTypeMeta,
		ObjectMeta: metav1.ObjectMeta{
			Name:      HashID(peer.GetId()),
			Namespace: c.Namespace,
			Labels: map[string]string{
				storagev1.NodeIDLabel: peer.GetId(),
			},
		},
		StoragePeer: peer,
	}
	return util.PatchObject(ctx, c.mgr.GetClient(), &stpeer)
}

// DemoteVoter demotes a voter to an observer.
func (c *Consensus) DemoteVoter(ctx context.Context, peer types.StoragePeer) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.isObserver {
		return errors.ErrNotStorageNode
	}
	var stpeer storagev1.StoragePeer
	err := c.mgr.GetClient().Get(ctx, client.ObjectKey{
		Name:      HashID(peer.GetId()),
		Namespace: c.Namespace,
	}, &stpeer)
	if err != nil {
		return err
	}
	c.trace(ctx, "Demoting voter", "peer", peer)
	stpeer.StoragePeer.ClusterStatus = v1.ClusterStatus_CLUSTER_OBSERVER
	return util.PatchObject(ctx, c.mgr.GetClient(), &stpeer)
}

// RemovePeer removes a peer from the consensus group.
func (c *Consensus) RemovePeer(ctx context.Context, peer types.StoragePeer, wait bool) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.isObserver {
		return errors.ErrNotStorageNode
	}
	c.trace(ctx, "Removing peer", "peer", peer)
	err := c.mgr.GetClient().Delete(ctx, &storagev1.StoragePeer{
		TypeMeta: storagev1.StoragePeerTypeMeta,
		ObjectMeta: metav1.ObjectMeta{
			Name:      HashID(peer.GetId()),
			Namespace: c.Namespace,
		},
	})
	return client.IgnoreNotFound(err)
}

func (c *Consensus) getPeers(ctx context.Context) ([]storagev1.StoragePeer, error) {
	var peers storagev1.StoragePeerList
	err := c.mgr.GetClient().List(ctx, &peers, client.InNamespace(c.Namespace))
	return peers.Items, client.IgnoreNotFound(err)
}

func (c *Consensus) containsPeer(peers []types.StoragePeer, peer string) bool {
	for _, p := range peers {
		if p.GetId() == peer {
			return true
		}
	}
	return false
}
