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
	v1 "github.com/webmeshproj/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// StoragePeer defines a peer in the storage consensus group.
type StoragePeerSpec struct {
	Peer *v1.StoragePeer `json:"peer"`
}

// StoragePeerStatus defines the observed state of a Route.
type StoragePeerStatus struct{}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// StoragePeer is the Schema for the StoragePeer API.
type StoragePeer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   StoragePeerSpec   `json:"spec,omitempty"`
	Status StoragePeerStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// StoragePeerList contains a list of storage peers.
type StoragePeerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []StoragePeer `json:"items"`
}

func init() {
	SchemeBuilder.Register(&StoragePeer{}, &StoragePeerList{})
}
