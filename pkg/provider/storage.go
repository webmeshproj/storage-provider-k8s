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
	"sync"
	"time"

	"github.com/webmeshproj/webmesh/pkg/storage"
	"github.com/webmeshproj/webmesh/pkg/storage/storageutil"
)

// Ensure we satisfy the storage interface.
var _ storage.MeshStorage = &Storage{}

// Storage is the storage interface for the storage provider.
type Storage struct {
	*Provider
	mu sync.RWMutex
}

// GetValue returns the value of a key.
func (st *Storage) GetValue(ctx context.Context, key []byte) ([]byte, error) {
	st.mu.RLock()
	defer st.mu.RUnlock()
	if !storageutil.IsValidKey(string(key)) {
		return nil, storage.ErrInvalidKey
	}
	return nil, storage.ErrNotImplemented
}

// PutValue sets the value of a key. TTL is optional and can be set to 0.
func (st *Storage) PutValue(ctx context.Context, key, value []byte, ttl time.Duration) error {
	st.mu.Lock()
	defer st.mu.Unlock()
	if !storageutil.IsValidKey(string(key)) {
		return storage.ErrInvalidKey
	}
	if !st.leaders.IsLeader() {
		return storage.ErrNotLeader
	}
	return storage.ErrNotImplemented
}

// Delete removes a key.
func (st *Storage) Delete(ctx context.Context, key []byte) error {
	st.mu.Lock()
	defer st.mu.Unlock()
	if !storageutil.IsValidKey(string(key)) {
		return storage.ErrInvalidKey
	}
	if !st.leaders.IsLeader() {
		return storage.ErrNotLeader
	}
	return storage.ErrNotImplemented
}

// ListKeys returns all keys with a given prefix.
func (st *Storage) ListKeys(ctx context.Context, prefix []byte) ([][]byte, error) {
	st.mu.RLock()
	defer st.mu.RUnlock()
	if !storageutil.IsValidKey(string(prefix)) {
		return nil, storage.ErrInvalidPrefix
	}
	return nil, storage.ErrNotImplemented
}

// IterPrefix iterates over all keys with a given prefix. It is important
// that the iterator not attempt any write operations as this will cause
// a deadlock. The iteration will stop if the iterator returns an error.
func (st *Storage) IterPrefix(ctx context.Context, prefix []byte, fn storage.PrefixIterator) error {
	st.mu.RLock()
	defer st.mu.RUnlock()
	if !storageutil.IsValidKey(string(prefix)) {
		return storage.ErrInvalidPrefix
	}
	return storage.ErrNotImplemented
}

// Subscribe will call the given function whenever a key with the given prefix is changed.
// The returned function can be called to unsubscribe.
func (st *Storage) Subscribe(ctx context.Context, prefix []byte, fn storage.SubscribeFunc) (context.CancelFunc, error) {
	st.mu.RLock()
	defer st.mu.RUnlock()
	if !storageutil.IsValidKey(string(prefix)) {
		return nil, storage.ErrInvalidPrefix
	}
	return nil, storage.ErrNotImplemented
}
