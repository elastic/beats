// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration

package elasticsearchstorage

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension/xextension/storage"
)

// newTestClientNamed returns a client scoped to an explicit component name,
// so a single test can hold several independent stores.
func newTestClientNamed(t *testing.T, ext *elasticStorage, name string) storage.Client {
	t.Helper()
	id := component.MustNewIDWithName("test_storage", name)
	client, err := ext.GetClient(context.Background(), component.KindReceiver, id, "")
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Close(context.Background()) })
	return client
}

// refreshIndex forces a refresh of the client's index so freshly-written docs
// become searchable, making Walk deterministic without waiting for the
// default refresh interval.
func refreshIndex(t *testing.T, ext *elasticStorage, c storage.Client) {
	t.Helper()
	idx := c.(*esStorageClient).index
	ext.clientMu.Lock()
	defer ext.clientMu.Unlock()
	_, _, err := ext.client.Request("POST", "/"+idx+"/_refresh", "", nil, nil)
	require.NoError(t, err)
}

// walker type-asserts the client to storage.Walker, the only enumeration
// method the client exposes.
func walker(t *testing.T, c storage.Client) storage.Walker {
	t.Helper()
	w, ok := c.(storage.Walker)
	require.True(t, ok, "client must implement storage.Walker")
	return w
}

func TestESClient_Walk_Empty(t *testing.T) {
	ext := newTestExtension(t)
	ctx := context.Background()
	c := newTestClientNamed(t, ext, "walk_empty")

	calls := 0
	require.NoError(t, walker(t, c).Walk(ctx, func(string, []byte) ([]*storage.Operation, error) {
		calls++
		return nil, nil
	}))
	assert.Zero(t, calls, "Walk on an empty store must not call the callback")
}

func TestESClient_Walk_PaginatesAllEntries(t *testing.T) {
	ext := newTestExtension(t)
	ctx := context.Background()
	c := newTestClientNamed(t, ext, "walk_full")

	// Shrink the page size so we exercise multi-page pagination without
	// writing 1000+ docs.
	c.(*esStorageClient).pageSize = 10

	n := 15
	want := make(map[string]string, n)
	for i := 0; i < n; i++ {
		key := fmt.Sprintf("k-%d", i)
		val := fmt.Sprintf(`{"i":%d}`, i)
		want[key] = val
		require.NoError(t, c.Set(ctx, key, []byte(val)))
	}
	refreshIndex(t, ext, c)

	seen := make(map[string]string, n)
	require.NoError(t, walker(t, c).Walk(ctx, func(key string, value []byte) ([]*storage.Operation, error) {
		seen[key] = string(value)
		return nil, nil
	}))
	assert.Len(t, seen, n, "Walk must visit every stored key across pages")
	for k, v := range want {
		assert.JSONEq(t, v, seen[k], "value for %q", k)
	}
}

func TestESClient_Walk_AppliesReturnedOps(t *testing.T) {
	ext := newTestExtension(t)
	ctx := context.Background()
	c := newTestClientNamed(t, ext, "walk_ops")

	require.NoError(t, c.Set(ctx, "a", []byte(`{"n":1}`)))
	refreshIndex(t, ext, c)

	require.NoError(t, walker(t, c).Walk(ctx, func(string, []byte) ([]*storage.Operation, error) {
		return []*storage.Operation{storage.SetOperation("b", []byte(`{"n":2}`))}, nil
	}))
	got, err := c.Get(ctx, "b")
	require.NoError(t, err)
	assert.JSONEq(t, `{"n":2}`, string(got), "a Set op returned by the walk must be applied")
}

func TestESClient_Walk_SkipAllStopsEarly(t *testing.T) {
	ext := newTestExtension(t)
	ctx := context.Background()
	c := newTestClientNamed(t, ext, "walk_skip")

	require.NoError(t, c.Set(ctx, "a", []byte(`{"n":1}`)))
	require.NoError(t, c.Set(ctx, "b", []byte(`{"n":2}`)))
	refreshIndex(t, ext, c)

	count := 0
	require.NoError(t, walker(t, c).Walk(ctx, func(string, []byte) ([]*storage.Operation, error) {
		count++
		return nil, storage.SkipAll
	}))
	assert.Equal(t, 1, count, "SkipAll must stop after the first entry")
}

func TestESClient_Walk_ErrorAbortsWithoutApplying(t *testing.T) {
	ext := newTestExtension(t)
	ctx := context.Background()
	c := newTestClientNamed(t, ext, "walk_err")

	require.NoError(t, c.Set(ctx, "a", []byte(`{"n":1}`)))
	refreshIndex(t, ext, c)

	sentinel := fmt.Errorf("boom")
	err := walker(t, c).Walk(ctx, func(string, []byte) ([]*storage.Operation, error) {
		return []*storage.Operation{storage.SetOperation("should_not_exist", []byte(`{}`))}, sentinel
	})
	assert.ErrorIs(t, err, sentinel)

	got, err := c.Get(ctx, "should_not_exist")
	require.NoError(t, err)
	assert.Nil(t, got, "operations must not be applied when the walk aborts with an error")
}
