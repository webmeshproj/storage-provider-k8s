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
	"strconv"

	"github.com/webmeshproj/webmesh/pkg/storage"
	"github.com/webmeshproj/webmesh/pkg/storage/errors"
	"github.com/webmeshproj/webmesh/pkg/storage/types"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	storagev1 "github.com/webmeshproj/storage-provider-k8s/api/storage/v1"
	"github.com/webmeshproj/storage-provider-k8s/pkg/provider/util"
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

// RBACEnabledConfigMap is the name of the ConfigMap that stores the RBAC enabled state.
const RBACEnabledConfigMap = "webmesh-rbac-enabled"

// SetEnabled sets the RBAC enabled state.
func (r *RBAC) SetEnabled(ctx context.Context, enabled bool) error {
	var cm corev1.ConfigMap
	err := r.cli.Get(ctx, client.ObjectKey{
		Namespace: r.namespace,
		Name:      RBACEnabledConfigMap,
	}, &cm)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			return err
		}
		// Create a new ConfigMap.
		cm = corev1.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ConfigMap",
				APIVersion: corev1.SchemeGroupVersion.String(),
			},
		}
	}
	if cm.Data == nil {
		cm.Data = map[string]string{}
	}
	cm.Data["enabled"] = fmt.Sprintf("%v", enabled)
	return util.PatchObject(ctx, r.cli, &cm)
}

// GetEnabled returns the RBAC enabled state.
func (r *RBAC) GetEnabled(ctx context.Context) (bool, error) {
	var cm corev1.ConfigMap
	err := r.cli.Get(ctx, client.ObjectKey{
		Namespace: r.namespace,
		Name:      RBACEnabledConfigMap,
	}, &cm)
	if err != nil {
		return false, client.IgnoreNotFound(err)
	}
	if cm.Data == nil {
		return false, nil
	}
	return strconv.ParseBool(cm.Data["enabled"])
}

// PutRole creates or updates a role.
func (r *RBAC) PutRole(ctx context.Context, role types.Role) error {
	var strole storagev1.Role
	strole.Spec.Role = role
	return util.PatchObject(ctx, r.cli, &strole)
}

// GetRole returns a role by name.
func (r *RBAC) GetRole(ctx context.Context, name string) (types.Role, error) {
	var strole storagev1.Role
	err := r.cli.Get(ctx, client.ObjectKey{
		Namespace: r.namespace,
		Name:      name,
	}, &strole)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			return types.Role{}, fmt.Errorf("get role: %w", err)
		}
		return types.Role{}, errors.ErrRoleNotFound
	}
	return strole.Spec.Role, nil
}

// DeleteRole deletes a role by name.
func (r *RBAC) DeleteRole(ctx context.Context, name string) error {
	err := r.cli.Delete(ctx, &storagev1.Role{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Role",
			APIVersion: storagev1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: r.namespace,
			Name:      name,
		},
	})
	return client.IgnoreNotFound(err)
}

// ListRoles returns a list of all roles.
func (r *RBAC) ListRoles(ctx context.Context) (types.RolesList, error) {
	var roles storagev1.RoleList
	err := r.cli.List(ctx, &roles, &client.ListOptions{
		Namespace: r.namespace,
	})
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			return nil, fmt.Errorf("list roles: %w", err)
		}
		return nil, nil
	}
	out := make(types.RolesList, len(roles.Items))
	for i, role := range roles.Items {
		out[i] = role.Spec.Role
	}
	return out, nil
}

// PutRoleBinding creates or updates a rolebinding.
func (r *RBAC) PutRoleBinding(ctx context.Context, rolebinding types.RoleBinding) error {
	var rb storagev1.RoleBinding
	rb.Spec.RoleBinding = rolebinding
	return util.PatchObject(ctx, r.cli, &rb)
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
