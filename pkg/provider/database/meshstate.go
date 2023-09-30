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
	"net/netip"
	"sync"

	"github.com/webmeshproj/webmesh/pkg/storage"
	"github.com/webmeshproj/webmesh/pkg/storage/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	storagev1 "github.com/webmeshproj/storage-provider-k8s/api/storage/v1"
	"github.com/webmeshproj/storage-provider-k8s/pkg/provider/util"
)

// Ensure we implement the interface.
var _ storage.MeshState = &MeshState{}

// MeshState implements the MeshState interface.
type MeshState struct {
	cli       client.Client
	namespace string
	mu        sync.RWMutex
}

// NewMeshState returns a new MeshState instance.
func NewMeshState(cli client.Client, namespace string) *MeshState {
	return &MeshState{
		cli:       cli,
		namespace: namespace,
	}
}

// MeshStateConfigName is the name of the mesh state ConfigMap.
const MeshStateConfigName = "webmesh-mesh-state"

// GetIPv6Prefix returns the IPv6 prefix.
func (st *MeshState) GetIPv6Prefix(ctx context.Context) (netip.Prefix, error) {
	st.mu.Lock()
	defer st.mu.Unlock()
	state, err := st.fetchState(ctx)
	if err != nil {
		return netip.Prefix{}, err
	}
	if state.Spec.IPv6Prefix == "" {
		return netip.Prefix{}, errors.ErrKeyNotFound
	}
	return netip.ParsePrefix(state.Spec.IPv6Prefix)
}

// SetIPv6Prefix sets the IPv6 prefix.
func (st *MeshState) SetIPv6Prefix(ctx context.Context, prefix netip.Prefix) error {
	st.mu.Lock()
	defer st.mu.Unlock()
	state, err := st.fetchState(ctx)
	if err != nil {
		return err
	}
	state.Spec.IPv6Prefix = prefix.String()
	return patchState(ctx, st.cli, state)
}

// GetIPv4Prefix returns the IPv4 prefix.
func (st *MeshState) GetIPv4Prefix(ctx context.Context) (netip.Prefix, error) {
	st.mu.Lock()
	defer st.mu.Unlock()
	state, err := st.fetchState(ctx)
	if err != nil {
		return netip.Prefix{}, err
	}
	if state.Spec.IPv4Prefix == "" {
		return netip.Prefix{}, errors.ErrKeyNotFound
	}
	return netip.ParsePrefix(state.Spec.IPv4Prefix)
}

// SetIPv4Prefix sets the IPv4 prefix.
func (st *MeshState) SetIPv4Prefix(ctx context.Context, prefix netip.Prefix) error {
	st.mu.Lock()
	defer st.mu.Unlock()
	state, err := st.fetchState(ctx)
	if err != nil {
		return err
	}
	state.Spec.IPv4Prefix = prefix.String()
	return patchState(ctx, st.cli, state)
}

// GetMeshDomain returns the mesh domain.
func (st *MeshState) GetMeshDomain(ctx context.Context) (string, error) {
	st.mu.Lock()
	defer st.mu.Unlock()
	state, err := st.fetchState(ctx)
	if err != nil {
		return "", err
	}
	if state.Spec.MeshDomain == "" {
		return "", errors.ErrKeyNotFound
	}
	return state.Spec.MeshDomain, nil
}

// SetMeshDomain sets the mesh domain.
func (st *MeshState) SetMeshDomain(ctx context.Context, domain string) error {
	st.mu.Lock()
	defer st.mu.Unlock()
	state, err := st.fetchState(ctx)
	if err != nil {
		return err
	}
	state.Spec.MeshDomain = domain
	return patchState(ctx, st.cli, state)
}

// fetchState fetches the current state.
func (st *MeshState) fetchState(ctx context.Context) (*storagev1.MeshState, error) {
	var state storagev1.MeshState
	err := st.cli.Get(ctx, client.ObjectKey{
		Namespace: "webmesh",
		Name:      MeshStateConfigName,
	}, &state)
	if err != nil {
		if client.IgnoreNotFound(err) == nil {
			// Return an empty state if the config doesn't exist.
			return &storagev1.MeshState{
				TypeMeta: metav1.TypeMeta{
					Kind:       "MeshState",
					APIVersion: "storage.webmesh.io/v1",
				},
			}, nil
		}
		return nil, fmt.Errorf("fetch mesh state: %w", err)
	}
	return &state, nil
}

func patchState(ctx context.Context, cli client.Client, state *storagev1.MeshState) error {
	util.StripPatchMeta(&state.ObjectMeta)
	err := cli.Patch(
		ctx,
		state,
		client.Apply,
		client.ForceOwnership,
		client.FieldOwner(storagev1.FieldOwner),
	)
	if err != nil {
		return fmt.Errorf("patch mesh state: %w", err)
	}
	return nil
}
