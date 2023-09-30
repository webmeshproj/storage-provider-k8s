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
	"github.com/webmeshproj/webmesh/pkg/storage/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Ensure we implement the interface.
var _ storage.Networking = &Networking{}

// Networking implements the Networking interface.
type Networking struct {
	cli client.Client
}

// NewNetworking returns a new Networking instance.
func NewNetworking(cli client.Client) *Networking {
	return &Networking{
		cli: cli,
	}
}

// PutNetworkACL creates or updates a NetworkACL.
func (nw *Networking) PutNetworkACL(ctx context.Context, acl types.NetworkACL) error {
	return nil
}

// GetNetworkACL returns a NetworkACL by name.
func (nw *Networking) GetNetworkACL(ctx context.Context, name string) (types.NetworkACL, error) {
	return types.NetworkACL{}, nil
}

// DeleteNetworkACL deletes a NetworkACL by name.
func (nw *Networking) DeleteNetworkACL(ctx context.Context, name string) error {
	return nil
}

// ListNetworkACLs returns a list of NetworkACLs.
func (nw *Networking) ListNetworkACLs(ctx context.Context) (types.NetworkACLs, error) {
	return nil, nil
}

// PutRoute creates or updates a Route.
func (nw *Networking) PutRoute(ctx context.Context, route types.Route) error {
	return nil
}

// GetRoute returns a Route by name.
func (nw *Networking) GetRoute(ctx context.Context, name string) (types.Route, error) {
	return types.Route{}, nil
}

// GetRoutesByNode returns a list of Routes for a given Node.
func (nw *Networking) GetRoutesByNode(ctx context.Context, nodeID types.NodeID) (types.Routes, error) {
	return nil, nil
}

// GetRoutesByCIDR returns a list of Routes for a given CIDR.
func (nw *Networking) GetRoutesByCIDR(ctx context.Context, cidr netip.Prefix) (types.Routes, error) {
	return nil, nil
}

// DeleteRoute deletes a Route by name.
func (nw *Networking) DeleteRoute(ctx context.Context, name string) error {
	return nil
}

// ListRoutes returns a list of Routes.
func (nw *Networking) ListRoutes(ctx context.Context) (types.Routes, error) {
	return nil, nil
}
