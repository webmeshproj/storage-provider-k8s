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
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/webmeshproj/webmesh/pkg/storage"
	"github.com/webmeshproj/webmesh/pkg/storage/storageutil"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Ensure we satisfy the storage interface.
var _ storage.MeshStorage = &Storage{}

const (
	// MeshStorageLabel is the label used to identify all mesh storage secrets.
	MeshStorageLabel = "webmesh.io/storage"
	// BucketLabel is the label used to identify the bucket for a given key.
	BucketLabel = "webmesh.io/storage-bucket"
	// FieldOwner is the field used to identify the owner of a secret.
	FieldOwner = "storage.webmesh.io"
)

// Storage is the storage interface for the storage provider.
type Storage struct {
	*Provider
	mu sync.RWMutex
}

// DataItem is a single item of data.
type DataItem struct {
	Value  []byte
	Expiry time.Time
}

// Unmarshal unmarshals the data item.
func (d *DataItem) Unmarshal(data []byte) error {
	return gob.NewDecoder(bytes.NewReader(data)).Decode(d)
}

// Marshal marshals the data item.
func (d DataItem) Marshal() ([]byte, error) {
	var buf bytes.Buffer
	err := gob.NewEncoder(&buf).Encode(d)
	return buf.Bytes(), err
}

// GetValue returns the value of a key.
func (st *Storage) GetValue(ctx context.Context, key []byte) ([]byte, error) {
	st.mu.RLock()
	defer st.mu.RUnlock()
	if !storageutil.IsValidKey(string(key)) {
		return nil, storage.ErrInvalidKey
	}
	bucket := st.bucketForKey(key)
	var secret corev1.Secret
	err := st.mgr.GetClient().Get(ctx, client.ObjectKey{
		Name:      bucket,
		Namespace: st.Namespace,
	}, &secret)
	if err != nil {
		if client.IgnoreNotFound(err) == nil {
			return nil, storage.NewKeyNotFoundError(key)
		}
		return nil, err
	}
	data, ok := secret.Data[string(key)]
	if !ok {
		return nil, storage.NewKeyNotFoundError(key)
	}
	var item DataItem
	err = item.Unmarshal(data)
	if err != nil {
		return nil, fmt.Errorf("unmarshal data item: %w", err)
	}
	if !item.Expiry.IsZero() && item.Expiry.Before(time.Now().UTC()) {
		// Defer a delete if we are the leader.
		if st.leaders.IsLeader() {
			defer func() {
				go func() {
					st.mu.Lock()
					defer st.mu.Unlock()
					delete(secret.Data, string(key))
					err := st.patchBucket(ctx, &secret)
					if err != nil {
						st.log.Error(err, "Failed to delete expired key", "key", string(key))
					}
				}()
			}()
		}
		return nil, storage.NewKeyNotFoundError(key)
	}
	return item.Value, nil
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
	bucket := st.bucketForKey(key)
	var secret corev1.Secret
	err := st.mgr.GetClient().Get(ctx, client.ObjectKey{
		Name:      bucket,
		Namespace: st.Namespace,
	}, &secret)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			return err
		}
		// If the bucket is not found, we are creating it for the first time.
		secret.Name = bucket
		secret.Namespace = st.Namespace
		secret.Labels = map[string]string{
			MeshStorageLabel: "true",
			BucketLabel:      bucket,
		}
		secret.Data = map[string][]byte{}
	}
	data, err := (DataItem{
		Value: value,
		Expiry: func() time.Time {
			if ttl == 0 {
				return time.Time{}
			}
			return time.Now().UTC().Add(ttl)
		}(),
	}).Marshal()
	if err != nil {
		return fmt.Errorf("marshal data item: %w", err)
	}
	secret.Data[string(key)] = data
	err = st.patchBucket(ctx, &secret)
	if err != nil {
		return err
	}
	return nil
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
	buckets, err := st.bucketsForPrefix(ctx, prefix)
	if err != nil {
		return nil, err
	}
	var keys [][]byte
	for _, bucket := range buckets {
		for k := range bucket.Data {
			// Extra sanity check on the key
			if !strings.HasPrefix(k, string(prefix)) {
				continue
			}
			// Check if the key is expired.
			item := DataItem{}
			err := item.Unmarshal(bucket.Data[k])
			if err != nil {
				return nil, fmt.Errorf("unmarshal data item: %w", err)
			}
			if !item.Expiry.IsZero() && item.Expiry.Before(time.Now().UTC()) {
				// Leave deletions to other function calls.
				continue
			}
			keys = append(keys, []byte(k))
		}
	}
	return keys, nil
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
	buckets, err := st.bucketsForPrefix(ctx, prefix)
	if err != nil {
		return err
	}
	// Map of index to keys to delete
	toDelete := map[int][]string{}
	for idx, bucket := range buckets {
		for k, v := range bucket.Data {
			if !strings.HasPrefix(k, string(prefix)) {
				continue
			}
			var item DataItem
			err := item.Unmarshal(v)
			if err != nil {
				return fmt.Errorf("unmarshal data item: %w", err)
			}
			if !item.Expiry.IsZero() && item.Expiry.Before(time.Now().UTC()) {
				// Defer a delete if we are the leader
				if st.leaders.IsLeader() {
					toDelete[idx] = append(toDelete[idx], k)
				}
				continue
			}
			if err := fn([]byte(k), item.Value); err != nil {
				return err
			}
		}
	}
	if st.leaders.IsLeader() && len(toDelete) > 0 {
		// Deferring the delete here is safe as we are the leader.
		defer func() {
			go func() {
				st.mu.Lock()
				defer st.mu.Unlock()
				for idx, keys := range toDelete {
					for _, key := range keys {
						delete(buckets[idx].Data, key)
					}
					err := st.patchBucket(ctx, buckets[idx])
					if err != nil {
						st.log.Error(err, "Failed to delete expired keys", "keys", keys)
					}
				}
			}()
		}()
	}
	return nil
}

// Subscribe will call the given function whenever a key with the given prefix is changed.
// The returned function can be called to unsubscribe.
func (st *Storage) Subscribe(ctx context.Context, prefix []byte, fn storage.SubscribeFunc) (context.CancelFunc, error) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if !storageutil.IsValidKey(string(prefix)) {
		return nil, storage.ErrInvalidPrefix
	}
	ctx, cancel := context.WithCancel(ctx)
	st.subs[uuid.NewString()] = Subscription{
		prefix: string(prefix),
		seen:   make(map[string][]byte),
		fn:     fn,
		ctx:    ctx,
		cancel: cancel,
	}
	return cancel, nil
}

func (st *Storage) patchBucket(ctx context.Context, bucket *corev1.Secret) error {
	err := st.mgr.GetClient().Patch(ctx, bucket, client.Apply, client.ForceOwnership, client.FieldOwner(FieldOwner))
	if err != nil {
		return fmt.Errorf("patch bucket secret: %w", err)
	}
	return nil
}

func (st *Storage) bucketsForPrefix(ctx context.Context, prefix []byte) ([]*corev1.Secret, error) {
	// We list all secrets that have a bucket label that matches the prefix.
	var bucketList corev1.SecretList
	err := st.mgr.GetClient().List(ctx,
		&bucketList,
		client.MatchingLabels{MeshStorageLabel: "true"},
	)
	if err != nil {
		return nil, err
	}
	var buckets []*corev1.Secret
	for _, bucket := range bucketList.Items {
		if len(prefix) == 0 {
			buckets = append(buckets, &bucket)
			continue
		}
		if strings.HasPrefix(bucket.Labels[BucketLabel], string(prefix)) {
			buckets = append(buckets, &bucket)
		}
	}
	return buckets, nil
}

func (st *Storage) bucketForKey(key []byte) string {
	// We create buckets according to the largest part of the prefix.
	k := string(key)
	spl := strings.Split(k, "/")
	if len(spl) == 0 {
		return ""
	}
	return strings.Join(spl[:len(spl)-1], "/")
}
