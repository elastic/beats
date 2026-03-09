// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration

package elasticsearchstorage

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.uber.org/zap/zaptest"

	"github.com/elastic/beats/v7/libbeat/esleg/eslegclient"
	"github.com/elastic/beats/v7/libbeat/statestore/backend"
	"github.com/elastic/beats/v7/libbeat/tests/integration"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

// uniqueIndex returns a unique index name scoped to the test to avoid
// collisions between parallel test runs against the same Elasticsearch cluster.
func uniqueIndex(t *testing.T) string {
	safe := strings.NewReplacer("/", "-", " ", "-")
	return "test-elasticstorage-" + safe.Replace(strings.ToLower(t.Name()))
}

func newTestStore(t *testing.T, storeName string) *store {
	t.Helper()

	integration.EnsureESIsRunning(t)
	esURL := integration.GetESURL(t, "http")
	user := esURL.User.Username()
	pass, _ := esURL.User.Password()

	conn, err := eslegclient.NewConnection(eslegclient.ConnectionSettings{
		URL:      fmt.Sprintf("%s://%s", esURL.Scheme, esURL.Host),
		Username: user,
		Password: pass,
	}, logptest.NewTestingLogger(t, ""))
	require.NoError(t, err)

	require.NoError(t, conn.Connect(t.Context()))
	t.Cleanup(func() { _ = conn.Close() })

	s, err := openStore(conn, storeName)
	require.NoError(t, err)
	t.Cleanup(func() { _ = s.Close() })

	return s
}

func TestStore_Close(t *testing.T) {
	s := newTestStore(t, uniqueIndex(t))
	require.NoError(t, s.Close())
}

func TestStore_SetGet(t *testing.T) {
	s := newTestStore(t, uniqueIndex(t))

	type cursor struct {
		Position int    `json:"position"`
		Name     string `json:"name"`
	}

	want := cursor{Position: 42, Name: "hello"}
	require.NoError(t, s.Set("mykey", want))

	var got cursor
	require.NoError(t, s.Get("mykey", &got))
	assert.Equal(t, want, got)
}

func TestStore_Get_UnknownKey(t *testing.T) {
	s := newTestStore(t, uniqueIndex(t))

	var v interface{}
	err := s.Get("nonexistent", &v)
	assert.ErrorIs(t, err, ErrKeyUnknown)
}

func TestStore_Has_ExistingKey(t *testing.T) {
	s := newTestStore(t, uniqueIndex(t))

	require.NoError(t, s.Set("existingkey", map[string]string{"foo": "bar"}))

	ok, err := s.Has("existingkey")
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestStore_Has_MissingKey(t *testing.T) {
	s := newTestStore(t, uniqueIndex(t))

	ok, err := s.Has("doesnotexist")
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestStore_Remove(t *testing.T) {
	s := newTestStore(t, uniqueIndex(t))

	require.NoError(t, s.Set("toremove", "value"))

	ok, err := s.Has("toremove")
	require.NoError(t, err)
	require.True(t, ok)

	require.NoError(t, s.Remove("toremove"))

	ok, err = s.Has("toremove")
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestStore_Each(t *testing.T) {
	s := newTestStore(t, uniqueIndex(t))

	keys := []string{"key1", "key2", "key3"}
	for _, k := range keys {
		require.NoError(t, s.Set(k, k+"_value"))
	}

	seen := make(map[string]bool)
	require.NoError(t, s.Each(func(key string, _ backend.ValueDecoder) (bool, error) {
		seen[key] = true
		return true, nil
	}))

	for _, k := range keys {
		assert.True(t, seen[k], "expected key %q to be yielded by Each", k)
	}
}

func TestStore_Each_Empty(t *testing.T) {
	s := newTestStore(t, uniqueIndex(t))

	called := false
	require.NoError(t, s.Each(func(_ string, _ backend.ValueDecoder) (bool, error) {
		called = true
		return true, nil
	}))
	assert.False(t, called, "callback must not be called on an empty store")
}

func TestStore_Each_EarlyStop(t *testing.T) {
	s := newTestStore(t, uniqueIndex(t))

	for i := 0; i < 5; i++ {
		require.NoError(t, s.Set(fmt.Sprintf("k%d", i), i))
	}

	count := 0
	require.NoError(t, s.Each(func(_ string, _ backend.ValueDecoder) (bool, error) {
		count++
		return false, nil // stop after the first entry
	}))
	assert.Equal(t, 1, count, "Each must stop when callback returns false")
}

func TestStore_Each_PropagatesError(t *testing.T) {
	s := newTestStore(t, uniqueIndex(t))
	require.NoError(t, s.Set("somekey", "somevalue"))

	wantErr := errors.New("injected error")
	err := s.Each(func(_ string, _ backend.ValueDecoder) (bool, error) {
		return false, wantErr
	})
	assert.ErrorIs(t, err, wantErr)
}

func TestStore_SetID(t *testing.T) {
	integration.EnsureESIsRunning(t)
	esURL := integration.GetESURL(t, "http")
	user := esURL.User.Username()
	pass, _ := esURL.User.Password()

	conn, err := eslegclient.NewConnection(eslegclient.ConnectionSettings{
		URL:      fmt.Sprintf("%s://%s", esURL.Scheme, esURL.Host),
		Username: user,
		Password: pass,
	}, logptest.NewTestingLogger(t, ""))
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	require.NoError(t, conn.Connect(ctx))
	t.Cleanup(func() { _ = conn.Close() })

	s, err := openStore(conn, "original")
	require.NoError(t, err)

	assert.Equal(t, renderIndexName("original"), s.index)

	s.SetID("new-id")
	assert.Equal(t, renderIndexName("new-id"), s.index)

	// Empty ID must be ignored.
	s.SetID("")
	assert.Equal(t, renderIndexName("new-id"), s.index)
}

func TestRenderIndexName(t *testing.T) {
	assert.Equal(t, "agentless-state-mystore", renderIndexName("mystore"))
	assert.Equal(t, "agentless-state-", renderIndexName(""))
}

func TestElasticStorage_Lifecycle(t *testing.T) {
	integration.EnsureESIsRunning(t)
	esURL := integration.GetESURL(t, "http")
	user := esURL.User.Username()
	pass, _ := esURL.User.Password()

	cfg := &Config{
		ElasticsearchConfig: map[string]interface{}{
			"hosts":    []string{fmt.Sprintf("%s://%s", esURL.Scheme, esURL.Host)},
			"username": user,
			"password": pass,
		},
	}

	ext := &elasticStorage{cfg: cfg, logger: zaptest.NewLogger(t)}

	require.NoError(t, ext.Start(context.Background(), componenttest.NewNopHost()))
	assert.NotNil(t, ext.client)

	// Access returns a usable store.
	s, err := ext.Access(uniqueIndex(t))
	require.NoError(t, err)
	require.NotNil(t, s)

	// Verify a basic round-trip via the returned store.
	require.NoError(t, s.Set("lifecycle-key", map[string]int{"n": 7}))
	var got map[string]int
	require.NoError(t, s.Get("lifecycle-key", &got))
	assert.Equal(t, map[string]int{"n": 7}, got)

	require.NoError(t, ext.Shutdown(context.Background()))
}

func TestElasticStorage_Start_BadCredentials(t *testing.T) {
	integration.EnsureESIsRunning(t)
	esURL := integration.GetESURL(t, "http")

	cfg := &Config{
		ElasticsearchConfig: map[string]interface{}{
			"hosts":    []string{fmt.Sprintf("%s://%s", esURL.Scheme, esURL.Host)},
			"username": "invaliduser",
			"password": "wrongpassword",
		},
	}
	ext := &elasticStorage{cfg: cfg, logger: zaptest.NewLogger(t)}
	err := ext.Start(context.Background(), componenttest.NewNopHost())
	// NewConnectedClient performs a ping that will receive a 401; it should err.
	require.Error(t, err)
	_ = ext.Shutdown(context.Background())
}
