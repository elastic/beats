// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package otelstorage

import (
	"context"
	"errors"
	"sort"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/extension/xextension/storage"

	"github.com/elastic/beats/v7/libbeat/statestore/backend"
)

func TestRoundTrip_StoreFromClient_Mock(t *testing.T) {
	f := &fakeStorageClient{m: map[string][]byte{}}
	st := NewStoreFromClient(f)

	require.NoError(t, st.Set("k", map[string]any{"x": "hello"}))

	var got map[string]any
	require.NoError(t, st.Get("k", &got))
	require.Equal(t, "hello", got["x"])

	ok, err := st.Has("k")
	require.NoError(t, err)
	require.True(t, ok)

	require.ErrorIs(t, st.Get("missing", &got), errKeyUnknown)

	require.NoError(t, st.Remove("k"))
	ok, err = st.Has("k")
	require.NoError(t, err)
	require.False(t, ok)
}

func TestStoreFromClient_Each(t *testing.T) {
	f := &fakeStorageClient{m: map[string][]byte{
		"a": []byte(`{"z":"one"}`),
		"b": []byte(`{"z":"two"}`),
	}}
	st := NewStoreFromClient(f)

	var keys []string
	err := st.Each(func(k string, dec backend.ValueDecoder) (bool, error) {
		keys = append(keys, k)
		var m map[string]any
		require.NoError(t, dec.Decode(&m))
		return true, nil
	})
	require.NoError(t, err)
	require.Equal(t, []string{"a", "b"}, keys)
}

// fakeStorageClient is a minimal [storage.Client] plus Each for adapter tests.
type fakeStorageClient struct {
	mu sync.RWMutex
	m  map[string][]byte
}

var _ storage.Client = (*fakeStorageClient)(nil)

func (f *fakeStorageClient) Get(ctx context.Context, key string) ([]byte, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.m[key], nil
}

func (f *fakeStorageClient) Set(ctx context.Context, key string, value []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.m[key] = append([]byte(nil), value...)
	return nil
}

func (f *fakeStorageClient) Delete(ctx context.Context, key string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.m, key)
	return nil
}

func (f *fakeStorageClient) Batch(ctx context.Context, ops ...*storage.Operation) error {
	for _, op := range ops {
		switch op.Type {
		case storage.Get:
			b, err := f.Get(ctx, op.Key)
			if err != nil {
				return err
			}
			if b != nil {
				op.Value = append([]byte(nil), b...)
			}
		case storage.Set:
			if err := f.Set(ctx, op.Key, op.Value); err != nil {
				return err
			}
		case storage.Delete:
			if err := f.Delete(ctx, op.Key); err != nil {
				return err
			}
		default:
			return errors.New("wrong operation type")
		}
	}
	return nil
}

func (f *fakeStorageClient) Close(ctx context.Context) error { return nil }

func (f *fakeStorageClient) Walk(ctx context.Context, fn storage.WalkFunc) error {
	f.mu.RLock()
	defer f.mu.RUnlock()
	var keys []string
	for k := range f.m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var batch []*storage.Operation
	for _, k := range keys {
		v := append([]byte(nil), f.m[k]...)
		ops, err := fn(k, v)
		if err != nil {
			if errors.Is(err, storage.SkipAll) {
				batch = append(batch, ops...)
				return f.Batch(ctx, batch...)
			}
			return err
		}
		batch = append(batch, ops...)
	}
	if len(batch) > 0 {
		return f.Batch(ctx, batch...)
	}
	return nil
}
