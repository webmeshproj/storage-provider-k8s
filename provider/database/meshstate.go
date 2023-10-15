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
	"sync"

	"github.com/webmeshproj/webmesh/pkg/storage"
	"github.com/webmeshproj/webmesh/pkg/storage/errors"
	"github.com/webmeshproj/webmesh/pkg/storage/types"
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

// SetMeshState sets the mesh state.
func (st *MeshState) SetMeshState(ctx context.Context, state types.NetworkState) error {
	st.mu.Lock()
	defer st.mu.Unlock()
	// Fetch the current state.
	currentState, err := st.fetchState(ctx)
	if err != nil {
		return err
	}
	// Patch the state.
	currentState.NetworkState = state
	return util.PatchObject(ctx, st.cli, currentState)
}

// GetMeshState returns the mesh state.
func (st *MeshState) GetMeshState(ctx context.Context) (types.NetworkState, error) {
	st.mu.Lock()
	defer st.mu.Unlock()
	state, err := st.fetchState(ctx)
	if err != nil {
		return types.NetworkState{}, err
	}
	// If we don't have a mesh domain, return not found
	if state.NetworkState.GetDomain() == "" {
		return types.NetworkState{}, errors.NewKeyNotFoundError([]byte("mesh domain"))
	}
	// Make sure we at least have an IPv4 or IPv6 network.
	if state.NetworkState.GetNetworkV4() == "" && state.NetworkState.GetNetworkV6() == "" {
		return types.NetworkState{}, errors.NewKeyNotFoundError([]byte("mesh network"))
	}
	return state.NetworkState, nil
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
				TypeMeta: storagev1.MeshStateTypeMeta,
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
	state.TypeMeta = storagev1.MeshStateTypeMeta
	return &state, nil
}
