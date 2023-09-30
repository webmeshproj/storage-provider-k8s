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

	"github.com/webmeshproj/webmesh/pkg/storage"
	"github.com/webmeshproj/webmesh/pkg/storage/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Ensure we implement the interface.
var _ storage.RBAC = &RBAC{}

// RBAC implements the RBAC interface.
type RBAC struct {
	cli       client.Client
	namespace string
}

// NewRBAC returns a new RBAC instance.
func NewRBAC(cli client.Client, namespace string) *RBAC {
	return &RBAC{
		cli:       cli,
		namespace: namespace,
	}
}

// SetEnabled sets the RBAC enabled state.
func (r *RBAC) SetEnabled(ctx context.Context, enabled bool) error {
	return nil
}

// GetEnabled returns the RBAC enabled state.
func (r *RBAC) GetEnabled(ctx context.Context) (bool, error) {
	return false, nil
}

// PutRole creates or updates a role.
func (r *RBAC) PutRole(ctx context.Context, role types.Role) error {
	return nil
}

// GetRole returns a role by name.
func (r *RBAC) GetRole(ctx context.Context, name string) (types.Role, error) {
	return types.Role{}, nil
}

// DeleteRole deletes a role by name.
func (r *RBAC) DeleteRole(ctx context.Context, name string) error {
	return nil
}

// ListRoles returns a list of all roles.
func (r *RBAC) ListRoles(ctx context.Context) (types.RolesList, error) {
	return nil, nil
}

// PutRoleBinding creates or updates a rolebinding.
func (r *RBAC) PutRoleBinding(ctx context.Context, rolebinding types.RoleBinding) error {
	return nil
}

// GetRoleBinding returns a rolebinding by name.
func (r *RBAC) GetRoleBinding(ctx context.Context, name string) (types.RoleBinding, error) {
	return types.RoleBinding{}, nil
}

// DeleteRoleBinding deletes a rolebinding by name.
func (r *RBAC) DeleteRoleBinding(ctx context.Context, name string) error {
	return nil
}

// ListRoleBindings returns a list of all rolebindings.
func (r *RBAC) ListRoleBindings(ctx context.Context) ([]types.RoleBinding, error) {
	return nil, nil
}

// PutGroup creates or updates a group.
func (r *RBAC) PutGroup(ctx context.Context, group types.Group) error {
	return nil
}

// GetGroup returns a group by name.
func (r *RBAC) GetGroup(ctx context.Context, name string) (types.Group, error) {
	return types.Group{}, nil
}

// DeleteGroup deletes a group by name.
func (r *RBAC) DeleteGroup(ctx context.Context, name string) error {
	return nil
}

// ListGroups returns a list of all groups.
func (r *RBAC) ListGroups(ctx context.Context) ([]types.Group, error) {
	return nil, nil
}

// ListNodeRoles returns a list of all roles for a node.
func (r *RBAC) ListNodeRoles(ctx context.Context, nodeID string) (types.RolesList, error) {
	return nil, nil
}

// ListUserRoles returns a list of all roles for a user.
func (r *RBAC) ListUserRoles(ctx context.Context, user string) (types.RolesList, error) {
	return nil, nil
}
