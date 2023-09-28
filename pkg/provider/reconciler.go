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

// Package provider contains the storage provider implementation for Kubernetes.
package provider

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Ensure we satisfy the resource recorder interface.
var _ resourcelock.EventRecorder = &Provider{}

// Ensure we satisfy the reconcile interface.
var _ reconcile.Reconciler = &Provider{}

// Eventf implements the resource recorder and is used to track changes to the leader
// election lease.
func (p *Provider) Eventf(obj runtime.Object, eventType, reason, message string, args ...interface{}) {
}

// Reconcile reconciles the given request. This is used to notify subscribers of changes to the given object.
func (p *Provider) Reconcile(ctx context.Context, req reconcile.Request) (ctrl.Result, error) {
	return ctrl.Result{}, nil
}

func (p *Provider) enqueueObjectIfOwner(ctx context.Context, o client.Object) []reconcile.Request {
	for _, ref := range o.GetOwnerReferences() {
		if ref.Controller != nil && *ref.Controller && ref.Kind == "Webmesh" {
			p.log.Info("Secret owner is Webmesh, enqueueing reconcile")
			return []reconcile.Request{
				{
					NamespacedName: types.NamespacedName{
						Name:      o.GetName(),
						Namespace: o.GetNamespace(),
					},
				},
			}
		}
	}
	return nil
}
