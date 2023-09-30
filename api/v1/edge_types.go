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

// MeshEdgeSpec defines the desired state of a MeshEdge.
type MeshEdgeSpec struct {
	MeshEdge types.MeshEdge `json:",inline"`
}

// MeshEdgeStatus defines the observed state of a MeshEdge.
type MeshEdgeStatus struct{}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// MeshEdge is the Schema for the MeshEdges API.
type MeshEdge struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PeerSpec   `json:"spec,omitempty"`
	Status PeerStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// MeshEdgeList contains a list of peers.
type MeshEdgeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MeshEdge `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MeshEdge{}, &MeshEdgeList{})
}