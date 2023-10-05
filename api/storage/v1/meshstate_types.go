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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// MeshState is the Schema for the MeshState API.
type MeshState struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// IPv6Prefix is the IPv6 prefix for the mesh.
	IPv6Prefix string `json:"ipv6Prefix,omitempty"`
	// IPv4Prefix is the IPv4 prefix for the mesh.
	IPv4Prefix string `json:"ipv4Prefix,omitempty"`
	// MeshDomain is the domain name for the mesh.
	MeshDomain string `json:"meshDomain,omitempty"`
}

//+kubebuilder:object:root=true

// MeshStateList contains a list of meshstates.
type MeshStateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MeshState `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MeshState{}, &MeshStateList{})
}
