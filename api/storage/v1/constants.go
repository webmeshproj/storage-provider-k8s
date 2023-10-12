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

const (
	// FieldOwner is the field used to identify the owner of mesh data.
	FieldOwner = "storage.webmesh.io"
	// NodeIDLabel is the label used to store the node ID.
	NodeIDLabel = "webmesh.io/node-id"
	// PublicKeyLabel is the label used to store the hashed public key of a node.
	PublicKeyLabel = "webmesh.io/public-key"
	// NodeIPv4Label is the label used to store the IPv4 address of a node.
	NodeIPv4Label = "webmesh.io/node-ipv4"
	// NodeIPv6Label is the label used to store the IPv6 address of a node.
	NodeIPv6Label = "webmesh.io/node-ipv6"
	// EdgeSourceLabel is the label used to store the source node ID.
	EdgeSourceLabel = "webmesh.io/edge-source"
	// EdgeTargetLabel is the label used to store the target node ID.
	EdgeTargetLabel = "webmesh.io/edge-target"
)
