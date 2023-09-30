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
	"github.com/webmeshproj/webmesh/pkg/storage/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NetworkACLSpec defines the desired state of a NetworkACL.
type NetworkACLSpec struct {
	NetworkACL types.NetworkACL `json:"networkACL"`
}

// NetworkACLStatus defines the observed state of a NetworkACL.
type NetworkACLStatus struct{}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// NetworkACL is the Schema for the NetworkACLs API.
type NetworkACL struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NetworkACLSpec   `json:"spec,omitempty"`
	Status NetworkACLStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// NetworkACLList contains a list of peers.
type NetworkACLList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NetworkACL `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NetworkACL{}, &NetworkACLList{})
}
