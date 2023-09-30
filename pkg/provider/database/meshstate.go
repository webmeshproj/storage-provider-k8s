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
	"net/netip"

	"github.com/webmeshproj/webmesh/pkg/storage"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Ensure we implement the interface.
var _ storage.MeshState = &MeshState{}

// MeshState implements the MeshState interface.
type MeshState struct {
	cli client.Client
}

// NewMeshState returns a new MeshState instance.
func NewMeshState(cli client.Client) *MeshState {
	return &MeshState{
		cli: cli,
	}
}

// GetIPv6Prefix returns the IPv6 prefix.
func (st *MeshState) GetIPv6Prefix(ctx context.Context) (netip.Prefix, error) {
	return netip.Prefix{}, nil
}

// SetIPv6Prefix sets the IPv6 prefix.
func (st *MeshState) SetIPv6Prefix(ctx context.Context, prefix netip.Prefix) error {
	return nil
}

// GetIPv4Prefix returns the IPv4 prefix.
func (st *MeshState) GetIPv4Prefix(ctx context.Context) (netip.Prefix, error) {
	return netip.Prefix{}, nil
}

// SetIPv4Prefix sets the IPv4 prefix.
func (st *MeshState) SetIPv4Prefix(ctx context.Context, prefix netip.Prefix) error {
	return nil
}

// GetMeshDomain returns the mesh domain.
func (st *MeshState) GetMeshDomain(ctx context.Context) (string, error) {
	return "", nil
}

// SetMeshDomain sets the mesh domain.
func (st *MeshState) SetMeshDomain(ctx context.Context, domain string) error {
	return nil
}
