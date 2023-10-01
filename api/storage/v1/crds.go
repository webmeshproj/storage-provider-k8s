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

package v1

import (
	"context"
	"embed"
	"fmt"
	"sync"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//go:embed crds
var crdFS embed.FS

// CustomObjects is a list of all custom objects used for storage.
var CustomObjects = []client.Object{
	&MeshState{},
	&Peer{},
	&MeshEdge{},
	&NetworkACL{},
	&Route{},
	&Role{},
	&RoleBinding{},
	&Group{},
}

var (
	customResourceDefinitions = []client.Object{}
	once                      sync.Once
)

// GetCustomResourceDefintions returns a list of all CRDs used for storage.
func GetCustomResourceDefintions() []client.Object {
	once.Do(func() {
		// Read all CRDs into memory.
		files, err := crdFS.ReadDir("crds")
		if err != nil {
			panic(err)
		}
		for _, file := range files {
			var crd unstructured.Unstructured
			crd.Object = make(map[string]interface{})
			data, err := crdFS.ReadFile("crds/" + file.Name())
			if err != nil {
				panic(err)
			}
			err = crd.UnmarshalJSON(data)
			if err != nil {
				panic(err)
			}
			crd.SetGeneration(0)
			crd.SetUID("")
			crd.SetResourceVersion("")
			crd.SetSelfLink("")
			crd.SetCreationTimestamp(metav1.Time{})
			crd.SetManagedFields(nil)
			customResourceDefinitions = append(customResourceDefinitions, crd.DeepCopy())
		}
	})
	return customResourceDefinitions
}

// InstallCustomResourceDefinitions installs all CRDs used for storage.
func InstallCustomResourceDefinitions(c client.Client) error {
	for _, crd := range GetCustomResourceDefintions() {
		err := c.Patch(context.Background(), crd, client.Apply, client.ForceOwnership, client.FieldOwner(FieldOwner))
		if err != nil {
			return fmt.Errorf("failed to install CRD %s: %w", crd.GetName(), err)
		}
	}
	return nil
}
