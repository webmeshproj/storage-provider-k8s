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
	"github.com/webmeshproj/webmesh/pkg/storage/errors"
	"github.com/webmeshproj/webmesh/pkg/storage/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	storagev1 "github.com/webmeshproj/storage-provider-k8s/api/storage/v1"
	"github.com/webmeshproj/storage-provider-k8s/provider/util"
)

// Ensure we implement the interface.
var _ storage.Networking = &Networking{}

//+kubebuilder:rbac:groups=storage.webmesh.io,resources=networkacls;routes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=storage.webmesh.io,resources=networkacls/status;routes/status,verbs=get;update;patch

// RouteNodeLabel is the label used to store the node ID.
const RouteNodeLabel = "webmesh.io/node-id"

// Networking implements the Networking interface.
type Networking struct {
	cli       client.Client
	namespace string
}

// NewNetworking returns a new Networking instance.
func NewNetworking(cli client.Client, namespace string) *Networking {
	return &Networking{
		cli:       cli,
		namespace: namespace,
	}
}

// PutNetworkACL creates or updates a NetworkACL.
func (nw *Networking) PutNetworkACL(ctx context.Context, acl types.NetworkACL) error {
	var nacl storagev1.NetworkACL
	nacl.TypeMeta = storagev1.NetworkACLTypeMeta
	nacl.ObjectMeta = metav1.ObjectMeta{
		Name:      acl.Name,
		Namespace: nw.namespace,
	}
	nacl.NetworkACL = acl
	return util.PatchObject(ctx, nw.cli, &nacl)
}

// GetNetworkACL returns a NetworkACL by name.
func (nw *Networking) GetNetworkACL(ctx context.Context, name string) (types.NetworkACL, error) {
	var nacl storagev1.NetworkACL
	err := nw.cli.Get(ctx, client.ObjectKey{
		Name:      name,
		Namespace: nw.namespace,
	}, &nacl)
	if err != nil {
		if client.IgnoreNotFound(err) == nil {
			return types.NetworkACL{}, errors.ErrACLNotFound
		}
		return types.NetworkACL{}, err
	}
	return nacl.NetworkACL, nil
}

// DeleteNetworkACL deletes a NetworkACL by name.
func (nw *Networking) DeleteNetworkACL(ctx context.Context, name string) error {
	err := nw.cli.Delete(ctx, &storagev1.NetworkACL{
		TypeMeta: storagev1.NetworkACLTypeMeta,
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: nw.namespace,
		},
	})
	return client.IgnoreNotFound(err)
}

// ListNetworkACLs returns a list of NetworkACLs.
func (nw *Networking) ListNetworkACLs(ctx context.Context) (types.NetworkACLs, error) {
	var nacls storagev1.NetworkACLList
	err := nw.cli.List(ctx, &nacls, client.InNamespace(nw.namespace))
	if err != nil {
		if client.IgnoreNotFound(err) == nil {
			return nil, nil
		}
		return nil, err
	}
	acls := make(types.NetworkACLs, len(nacls.Items))
	for i, nacl := range nacls.Items {
		acls[i] = nacl.NetworkACL
	}
	return acls, nil
}

// PutRoute creates or updates a Route.
func (nw *Networking) PutRoute(ctx context.Context, route types.Route) error {
	var r storagev1.Route
	r.TypeMeta = storagev1.RouteTypeMeta
	r.ObjectMeta = metav1.ObjectMeta{
		Name:      route.Name,
		Namespace: nw.namespace,
		Labels: map[string]string{
			RouteNodeLabel: route.GetNode(),
		},
	}
	r.Route = route
	return util.PatchObject(ctx, nw.cli, &r)
}

// GetRoute returns a Route by name.
func (nw *Networking) GetRoute(ctx context.Context, name string) (types.Route, error) {
	var r storagev1.Route
	err := nw.cli.Get(ctx, client.ObjectKey{
		Name:      name,
		Namespace: nw.namespace,
	}, &r)
	if err != nil {
		if client.IgnoreNotFound(err) == nil {
			return types.Route{}, errors.ErrRouteNotFound
		}
		return types.Route{}, err
	}
	return r.Route, nil
}

// GetRoutesByNode returns a list of Routes for a given Node.
func (nw *Networking) GetRoutesByNode(ctx context.Context, nodeID types.NodeID) (types.Routes, error) {
	var routelist storagev1.RouteList
	err := nw.cli.List(ctx, &routelist, client.MatchingLabels{
		RouteNodeLabel: nodeID.String(),
	}, client.InNamespace(nw.namespace))
	if err != nil {
		if client.IgnoreNotFound(err) == nil {
			return nil, nil
		}
		return nil, err
	}
	rs := make(types.Routes, len(routelist.Items))
	for i, route := range routelist.Items {
		rs[i] = route.Route
	}
	return rs, nil
}

// GetRoutesByCIDR returns a list of Routes for a given CIDR.
func (nw *Networking) GetRoutesByCIDR(ctx context.Context, cidr netip.Prefix) (types.Routes, error) {
	var routelist storagev1.RouteList
	err := nw.cli.List(ctx, &routelist, client.InNamespace(nw.namespace))
	if err != nil {
		if client.IgnoreNotFound(err) == nil {
			return nil, nil
		}
		return nil, err
	}
	rs := make(types.Routes, 0, len(routelist.Items))
	for _, route := range routelist.Items {
		for _, prefix := range route.Route.DestinationPrefixes() {
			if prefix.Bits() == cidr.Bits() && prefix.Addr().Compare(cidr.Addr()) == 0 {
				rs = append(rs, route.Route)
				break
			}
		}
	}
	return rs, nil
}

// DeleteRoute deletes a Route by name.
func (nw *Networking) DeleteRoute(ctx context.Context, name string) error {
	err := nw.cli.Delete(ctx, &storagev1.Route{
		TypeMeta: storagev1.RouteTypeMeta,
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: nw.namespace,
		},
	})
	return client.IgnoreNotFound(err)
}

// ListRoutes returns a list of Routes.
func (nw *Networking) ListRoutes(ctx context.Context) (types.Routes, error) {
	var routes storagev1.RouteList
	err := nw.cli.List(ctx, &routes, client.InNamespace(nw.namespace))
	if err != nil {
		if client.IgnoreNotFound(err) == nil {
			return nil, nil
		}
		return nil, err
	}
	rs := make(types.Routes, len(routes.Items))
	for i, route := range routes.Items {
		rs[i] = route.Route
	}
	return rs, nil
}
