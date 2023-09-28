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
	"fmt"
	"sync"

	v1 "github.com/webmeshproj/api/v1"
	"github.com/webmeshproj/webmesh/pkg/storage"
	"google.golang.org/protobuf/encoding/protojson"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Ensure we satisfy the consensus interface.
var _ storage.Consensus = &Consensus{}

const (
	StoragePeersSecret = "webmesh-storage-peers"
)

type Peer struct {
	*v1.StoragePeer
}

func (p Peer) MarshalJSON() ([]byte, error) {
	return protojson.Marshal(p.StoragePeer)
}

func (p *Peer) UnmarshalJSON(data []byte) error {
	var sp v1.StoragePeer
	p.StoragePeer = &sp
	return protojson.Unmarshal(data, p.StoragePeer)
}

// Consensus is the consensus interface for the storage provider.
type Consensus struct {
	*Provider
	self Peer
	mu   sync.RWMutex
}

// IsLeader returns true if the node is the leader of the storage group.
func (c *Consensus) IsLeader() bool {
	return c.leaders.IsLeader()
}

// IsMember returns true if the node is a member of the storage group.
func (c *Consensus) IsMember() bool {
	return true
}

// GetPeers returns the peers of the storage group.
func (c *Consensus) GetPeers(ctx context.Context) ([]*v1.StoragePeer, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	peers, err := c.getPeers(ctx)
	if err != nil {
		return nil, fmt.Errorf("get peers: %w", err)
	}
	var sps []*v1.StoragePeer
	for _, p := range peers {
		sps = append(sps, p.StoragePeer)
	}
	return sps, nil
}

// GetLeader returns the leader of the storage group.
func (c *Consensus) GetLeader(ctx context.Context) (*v1.StoragePeer, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.IsLeader() {
		// Fast path return ourself if we have it stored.
		if c.self.StoragePeer != nil {
			return c.self.StoragePeer, nil
		}
	}
	peers, err := c.getPeers(ctx)
	if err != nil {
		return nil, fmt.Errorf("get peers: %w", err)
	}
	for _, p := range peers {
		if c.leaders.GetLeader() == p.GetId() {
			if c.IsLeader() {
				c.self = p
			}
			return p.StoragePeer, nil
		}
	}
	return nil, storage.ErrNoLeader
}

// AddVoter adds a voter to the consensus group.
func (c *Consensus) AddVoter(ctx context.Context, peer *v1.StoragePeer) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.IsLeader() {
		return storage.ErrNotLeader
	}
	peer.ClusterStatus = v1.ClusterStatus_CLUSTER_VOTER
	// Get the current peers.
	secret, err := c.getPeersSecret(ctx)
	if err != nil {
		return err
	}
	// Add the peer to the secret.
	data, err := Peer{peer}.MarshalJSON()
	if err != nil {
		return fmt.Errorf("marshal peer: %w", err)
	}
	secret.Data[peer.GetId()] = data
	return c.patchPeers(ctx, secret)
}

// AddObserver adds an observer to the consensus group.
func (c *Consensus) AddObserver(ctx context.Context, peer *v1.StoragePeer) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.IsLeader() {
		return storage.ErrNotLeader
	}
	peer.ClusterStatus = v1.ClusterStatus_CLUSTER_OBSERVER
	// Get the current peers.
	secret, err := c.getPeersSecret(ctx)
	if err != nil {
		return err
	}
	// Add the peer to the secret.
	data, err := Peer{peer}.MarshalJSON()
	if err != nil {
		return fmt.Errorf("marshal peer: %w", err)
	}
	secret.Data[peer.GetId()] = data
	return c.patchPeers(ctx, secret)
}

// DemoteVoter demotes a voter to an observer.
func (c *Consensus) DemoteVoter(ctx context.Context, peer *v1.StoragePeer) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.IsLeader() {
		return storage.ErrNotLeader
	}
	peer.ClusterStatus = v1.ClusterStatus_CLUSTER_OBSERVER
	// Get the current peers.
	secret, err := c.getPeersSecret(ctx)
	if err != nil {
		return err
	}
	// Add the peer to the secret.
	data, err := Peer{peer}.MarshalJSON()
	if err != nil {
		return fmt.Errorf("marshal peer: %w", err)
	}
	secret.Data[peer.GetId()] = data
	return c.patchPeers(ctx, secret)
}

// RemovePeer removes a peer from the consensus group. If wait
// is true, the function will wait for the peer to be removed.
func (c *Consensus) RemovePeer(ctx context.Context, peer *v1.StoragePeer, wait bool) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.IsLeader() {
		return storage.ErrNotLeader
	}
	// Get the current peers.
	secret, err := c.getPeersSecret(ctx)
	if err != nil {
		return err
	}
	// Remove the peer from the secret.
	delete(secret.Data, peer.GetId())
	return c.patchPeers(ctx, secret)
}

func (c *Consensus) getPeers(ctx context.Context) ([]Peer, error) {
	secret, err := c.getPeersSecret(ctx)
	if err != nil {
		return nil, err
	}
	var peers []Peer
	for _, v := range secret.Data {
		var p Peer
		err := p.UnmarshalJSON(v)
		if err != nil {
			return nil, fmt.Errorf("unmarshal peer: %w", err)
		}
		peers = append(peers, p)
	}
	return peers, nil
}

func (c *Consensus) getPeersSecret(ctx context.Context) (*corev1.Secret, error) {
	var secret corev1.Secret
	err := c.mgr.GetClient().Get(ctx, client.ObjectKey{
		Name:      StoragePeersSecret,
		Namespace: c.Namespace,
	}, &secret)
	if err != nil {
		return nil, fmt.Errorf("get peers secret: %w", err)
	}
	return &secret, nil
}

func (c *Consensus) patchPeers(ctx context.Context, secret *corev1.Secret) error {
	secret.TypeMeta = metav1.TypeMeta{
		Kind:       "Secret",
		APIVersion: "v1",
	}
	err := c.mgr.GetClient().Patch(ctx, secret, client.Apply, client.ForceOwnership, client.FieldOwner(FieldOwner))
	if err != nil {
		return fmt.Errorf("patch peers secret: %w", err)
	}
	return nil
}
