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
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	storagev1 "github.com/webmeshproj/storage-provider-k8s/api/storage/v1"
	"github.com/webmeshproj/storage-provider-k8s/provider/util"
)

// Ensure we implement the interface.
var _ storage.MeshState = &MeshState{}

//+kubebuilder:rbac:groups=storage.webmesh.io,resources=meshstates,verbs=get;list;watch;create;update;patch;delete

// MeshStateConfigName is the name of the mesh state object for a given cluster.
const MeshStateConfigName = "webmesh-mesh-state"

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

// GetIPv6Prefix returns the IPv6 prefix.
func (st *MeshState) GetIPv6Prefix(ctx context.Context) (netip.Prefix, error) {
	st.mu.Lock()
	defer st.mu.Unlock()
	state, err := st.fetchState(ctx)
	if err != nil {
		return netip.Prefix{}, err
	}
	if state.IPv6Prefix == "" {
		return netip.Prefix{}, errors.ErrKeyNotFound
	}
	return netip.ParsePrefix(state.IPv6Prefix)
}

// SetIPv6Prefix sets the IPv6 prefix.
func (st *MeshState) SetIPv6Prefix(ctx context.Context, prefix netip.Prefix) error {
	st.mu.Lock()
	defer st.mu.Unlock()
	state, err := st.fetchState(ctx)
	if err != nil {
		return err
	}
	state.IPv6Prefix = prefix.String()
	ctrl.Log.WithName("meshstate").V(1).Info("Set IPv6 prefix", "prefix", prefix.String())
	return util.PatchObject(ctx, st.cli, state.DeepCopy())
}

// GetIPv4Prefix returns the IPv4 prefix.
func (st *MeshState) GetIPv4Prefix(ctx context.Context) (netip.Prefix, error) {
	st.mu.Lock()
	defer st.mu.Unlock()
	state, err := st.fetchState(ctx)
	if err != nil {
		return netip.Prefix{}, err
	}
	if state.IPv4Prefix == "" {
		return netip.Prefix{}, errors.ErrKeyNotFound
	}
	return netip.ParsePrefix(state.IPv4Prefix)
}

// SetIPv4Prefix sets the IPv4 prefix.
func (st *MeshState) SetIPv4Prefix(ctx context.Context, prefix netip.Prefix) error {
	st.mu.Lock()
	defer st.mu.Unlock()
	state, err := st.fetchState(ctx)
	if err != nil {
		return err
	}
	state.IPv4Prefix = prefix.String()
	ctrl.Log.WithName("meshstate").V(1).Info("Set IPv4 prefix", "prefix", prefix.String())
	return util.PatchObject(ctx, st.cli, state.DeepCopy())
}

// GetMeshDomain returns the mesh domain.
func (st *MeshState) GetMeshDomain(ctx context.Context) (string, error) {
	st.mu.Lock()
	defer st.mu.Unlock()
	state, err := st.fetchState(ctx)
	if err != nil {
		return "", err
	}
	if state.MeshDomain == "" {
		return "", errors.ErrKeyNotFound
	}
	return state.MeshDomain, nil
}

// SetMeshDomain sets the mesh domain.
func (st *MeshState) SetMeshDomain(ctx context.Context, domain string) error {
	st.mu.Lock()
	defer st.mu.Unlock()
	state, err := st.fetchState(ctx)
	if err != nil {
		return err
	}
	state.MeshDomain = domain
	ctrl.Log.WithName("meshstate").V(1).Info("Set Mesh Domain", "domain", domain)
	return util.PatchObject(ctx, st.cli, state.DeepCopy())
}

// fetchState fetches the current state.
func (st *MeshState) fetchState(ctx context.Context) (*storagev1.MeshState, error) {
	var state storagev1.MeshState
	err := st.cli.Get(ctx, client.ObjectKey{
		Namespace: st.namespace,
		Name:      MeshStateConfigName,
	}, &state)
	if err != nil {
		if client.IgnoreNotFound(err) == nil {
			// Return an empty state if the config doesn't exist.
			ctrl.Log.WithName("meshstate").V(2).Info("State does not exist yet, returning empty state")
			return &storagev1.MeshState{
				TypeMeta: metav1.TypeMeta{
					Kind:       "MeshState",
					APIVersion: storagev1.GroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Namespace: st.namespace,
					Name:      MeshStateConfigName,
				},
			}, nil
		}
		return nil, fmt.Errorf("fetch mesh state: %w", err)
	}
	ctrl.Log.WithName("meshstate").V(2).Info("Current state", "state", state)
	// Ensure type meta is present for a call to patch.
	state.TypeMeta = metav1.TypeMeta{
		Kind:       "MeshState",
		APIVersion: storagev1.GroupVersion.String(),
	}
	return &state, nil
}
