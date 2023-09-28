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

package provider

import (
	"context"
	"time"

	"github.com/webmeshproj/webmesh/pkg/storage"
)

// Ensure we satisfy the storage interface.
var _ storage.MeshStorage = &Storage{}

// Storage is the storage interface for the storage provider.
type Storage struct{ *Provider }

// GetValue returns the value of a key.
func (st *Storage) GetValue(ctx context.Context, key []byte) ([]byte, error) {
	return nil, storage.ErrNotImplemented
}

// PutValue sets the value of a key. TTL is optional and can be set to 0.
func (st *Storage) PutValue(ctx context.Context, key, value []byte, ttl time.Duration) error {
	return storage.ErrNotImplemented
}

// Delete removes a key.
func (st *Storage) Delete(ctx context.Context, key []byte) error {
	return storage.ErrNotImplemented
}

// ListKeys returns all keys with a given prefix.
func (st *Storage) ListKeys(ctx context.Context, prefix []byte) ([][]byte, error) {
	return nil, storage.ErrNotImplemented
}

// IterPrefix iterates over all keys with a given prefix. It is important
// that the iterator not attempt any write operations as this will cause
// a deadlock. The iteration will stop if the iterator returns an error.
func (st *Storage) IterPrefix(ctx context.Context, prefix []byte, fn storage.PrefixIterator) error {
	return storage.ErrNotImplemented
}

// Subscribe will call the given function whenever a key with the given prefix is changed.
// The returned function can be called to unsubscribe.
func (st *Storage) Subscribe(ctx context.Context, prefix []byte, fn storage.SubscribeFunc) (context.CancelFunc, error) {
	return nil, storage.ErrNotImplemented
}
