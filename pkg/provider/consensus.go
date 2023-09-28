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

	v1 "github.com/webmeshproj/api/v1"
	"github.com/webmeshproj/webmesh/pkg/storage"
)

// Ensure we satisfy the consensus interface.
var _ storage.Consensus = &Consensus{}

// Consensus is the consensus interface for the storage provider.
type Consensus struct{ *Provider }

// IsLeader returns true if the node is the leader of the storage group.
func (c *Consensus) IsLeader() bool {
	return false
}

// IsMember returns true if the node is a member of the storage group.
func (c *Consensus) IsMember() bool {
	return true
}

// GetPeers returns the peers of the storage group.
func (c *Consensus) GetPeers(context.Context) ([]*v1.StoragePeer, error) {
	return nil, storage.ErrNotImplemented
}

// GetLeader returns the leader of the storage group.
func (c *Consensus) GetLeader(context.Context) (*v1.StoragePeer, error) {
	return nil, storage.ErrNotImplemented
}

// AddVoter adds a voter to the consensus group.
func (c *Consensus) AddVoter(context.Context, *v1.StoragePeer) error {
	return storage.ErrNotImplemented
}

// AddObserver adds an observer to the consensus group.
func (c *Consensus) AddObserver(context.Context, *v1.StoragePeer) error {
	return storage.ErrNotImplemented
}

// DemoteVoter demotes a voter to an observer.
func (c *Consensus) DemoteVoter(context.Context, *v1.StoragePeer) error {
	return storage.ErrNotImplemented
}

// RemovePeer removes a peer from the consensus group. If wait
// is true, the function will wait for the peer to be removed.
func (c *Consensus) RemovePeer(ctx context.Context, peer *v1.StoragePeer, wait bool) error {
	return storage.ErrNotImplemented
}
