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

package util

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	storagev1 "github.com/webmeshproj/storage-provider-k8s/api/storage/v1"
)

// StripPatchMeta strips the needed metadata to do a server-side apply with an object.
func StripPatchMeta(object client.Object) {
	object.SetResourceVersion("")
	object.SetUID("")
	object.SetGeneration(0)
	object.SetCreationTimestamp(metav1.Time{})
	object.SetManagedFields(nil)
}

// PatchObject patches an object with server-side apply.
func PatchObject(ctx context.Context, cli client.Client, object client.Object) error {
	StripPatchMeta(object)
	err := cli.Patch(
		ctx,
		object,
		client.Apply,
		client.ForceOwnership,
		client.FieldOwner(storagev1.FieldOwner),
	)
	if err != nil {
		return fmt.Errorf("server-side apply object %s/%s: %w", object.GetNamespace(), object.GetName(), err)
	}
	return nil
}
