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
	"bytes"
	"context"

	"github.com/webmeshproj/webmesh/pkg/storage"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Ensure we satisfy the resource recorder interface.
var _ resourcelock.EventRecorder = &Provider{}

// Eventf implements the resource recorder and is used to track changes to the leader
// election lease.
func (p *Provider) Eventf(obj runtime.Object, eventType, reason, message string, args ...interface{}) {
}

type Subscription struct {
	prefix []byte
	seen   map[string][]byte
	fn     storage.SubscribeFunc
	ctx    context.Context
	cancel context.CancelFunc
}

// Reconcile reconciles the given request. This is used to notify subscribers of changes to the given object.
func (p *Provider) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	p.subsmu.Lock()
	defer p.subsmu.Unlock()
	// Get the secret related the request.
	var secret corev1.Secret
	err := p.mgr.GetClient().Get(ctx, req.NamespacedName, &secret)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			return ctrl.Result{}, err
		}
		// Notify any subscribers of the deletion.
		for subID, sub := range p.storage.subs {
			select {
			case <-sub.ctx.Done():
				delete(p.storage.subs, subID)
				continue
			default:
			}
			if bytes.HasPrefix(bytes.ToLower([]byte(req.Name)), bytes.ToLower(sub.prefix)) {
				sub.fn([]byte(req.Name), nil)
			}
		}
		return ctrl.Result{}, nil
	}
	// Notify any subscribers of the change.
	for subID, sub := range p.storage.subs {
		select {
		case <-sub.ctx.Done():
			delete(p.storage.subs, subID)
			continue
		default:
		}
		for _, v := range secret.Data {
			var item DataItem
			err := item.Unmarshal(v)
			if err != nil {
				p.log.Error(err, "Failed to unmarshal data item")
				continue
			}
			if !bytes.HasPrefix(item.Key, sub.prefix) {
				continue
			}
			// Check if we have seen this key and its value before.
			if val, ok := sub.seen[string(item.Key)]; ok && bytes.Equal(val, item.Value) {
				continue
			}
			sub.seen[string(item.Key)] = item.Value
			// Notify the subscriber.
			sub.fn(item.Key, item.Value)
		}
		// Iterate on seen to see if any values were deleted from the bucket.
		for key := range sub.seen {
			if !bytes.HasPrefix([]byte(key), sub.prefix) {
				continue
			}
			if _, ok := secret.Data[hashKey([]byte(key))]; !ok {
				// Notify the subscriber.
				sub.fn([]byte(key), nil)
				delete(sub.seen, key)
			}
		}
	}
	return ctrl.Result{}, nil
}

func (p *Provider) enqueueObjectIfOwner(ctx context.Context, o client.Object) []ctrl.Request {
	labels := o.GetLabels()
	if labels == nil {
		return nil
	}
	if labels[MeshStorageLabel] != "true" {
		return nil
	}
	return []ctrl.Request{{NamespacedName: client.ObjectKey{Name: o.GetName(), Namespace: o.GetNamespace()}}}
}
