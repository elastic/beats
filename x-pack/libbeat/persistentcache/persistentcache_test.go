// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package persistentcache

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/tests/resources"
)

func init() {
	logp.DevelopmentSetup()
}

func TestPutGet(t *testing.T) {
	t.Parallel()

	registry := newTestRegistry(t)

	cache, err := newCache(registry, "test", Options{})
	require.NoError(t, err)
	defer cache.Close()

	type valueType struct {
		Something string
	}

	var key = "somekey"
	var value = valueType{Something: "foo"}

	err = cache.Put(key, value)
	assert.NoError(t, err)

	var result valueType
	err = cache.Get(key, &result)
	assert.NoError(t, err)
	assert.Equal(t, value, result)

	err = cache.Get("notexist", &result)
	assert.Error(t, err)
}

func TestPersist(t *testing.T) {
	t.Parallel()

	registry := newTestRegistry(t)

	cache, err := newCache(registry, "test", Options{})
	require.NoError(t, err)

	type valueType struct {
		Something string
	}

	var key = "somekey"
	var value = valueType{Something: "foo"}

	err = cache.Put(key, value)
	assert.NoError(t, err)

	cache.Close()

	cache, err = newCache(registry, "test", Options{})
	require.NoError(t, err)

	var result valueType
	err = cache.Get(key, &result)
	assert.NoError(t, err)
	assert.Equal(t, value, result)
}

func TestExpired(t *testing.T) {
	t.Parallel()

	registry := newTestRegistry(t)

	cache, err := newCache(registry, "test", Options{})
	require.NoError(t, err)

	now := time.Now()
	cache.clock = func() time.Time { return now }

	type valueType struct {
		Something string
	}

	var key = "somekey"
	var value = valueType{Something: "foo"}

	err = cache.PutWithTimeout(key, value, 5*time.Minute)
	assert.NoError(t, err)

	var result valueType
	err = cache.Get(key, &result)
	assert.NoError(t, err)
	assert.Equal(t, value, result)

	now = now.Add(10 * time.Minute)
	err = cache.Get(key, &result)
	assert.Error(t, err)
}

func TestCleanup(t *testing.T) {
	t.Parallel()

	registry := newTestRegistry(t)

	cache, err := newCache(registry, "test", Options{})
	require.NoError(t, err)

	now := time.Now()
	cache.clock = func() time.Time { return now }

	type valueType struct {
		Something string
	}

	var key = "somekey"
	var value = valueType{Something: "foo"}

	err = cache.PutWithTimeout(key, value, 5*time.Minute)
	assert.NoError(t, err)

	var result valueType
	err = cache.Get(key, &result)
	assert.NoError(t, err)
	assert.Equal(t, value, result)

	now = now.Add(10 * time.Minute)
	removedCount := cache.CleanUp()
	assert.Equal(t, 1, removedCount)

	err = cache.Get(key, &result)
	assert.Error(t, err)
}

func TestJanitor(t *testing.T) {
	defer resources.NewGoroutinesChecker().Check(t)

	registry := newTestRegistry(t)

	cache, err := newCache(registry, "test", Options{})
	require.NoError(t, err)

	cache.StartJanitor(10 * time.Millisecond)
	defer cache.StopJanitor()

	now := time.Now()
	cache.clock = func() time.Time { return now }

	type valueType struct {
		Something string
	}

	var key = "somekey"
	var value = valueType{Something: "foo"}

	err = cache.PutWithTimeout(key, value, 10*time.Second)
	assert.NoError(t, err)

	now = now.Add(20 * time.Second)

	var result valueType
	timeout := time.After(5 * time.Second)
	removed := false
	for !removed {
		select {
		case <-time.After(1 * time.Millisecond):
			err = cache.Get(key, &result)
			require.Error(t, err)
			if !errors.Is(err, expiredError) {
				removed = true
			}
		case <-timeout:
			t.Fatal("timeout waiting for janitor to remove key")
		}
	}
}

func TestRefreshOnAccess(t *testing.T) {
	t.Parallel()

	registry := newTestRegistry(t)

	options := Options{
		Timeout:         60 * time.Second,
		RefreshOnAccess: true,
	}

	cache, err := newCache(registry, "test", options)
	require.NoError(t, err)

	now := time.Now()
	cache.clock = func() time.Time { return now }

	type valueType struct {
		Something string
	}

	var key1 = "somekey"
	var value1 = valueType{Something: "foo"}
	var key2 = "otherkey"
	var value2 = valueType{Something: "bar"}

	err = cache.Put(key1, value1)
	assert.NoError(t, err)
	err = cache.Put(key2, value2)
	assert.NoError(t, err)

	now = now.Add(40 * time.Second)

	var result valueType
	err = cache.Get(key1, &result)
	assert.NoError(t, err)
	assert.Equal(t, value1, result)

	now = now.Add(40 * time.Second)
	removedCount := cache.CleanUp()
	assert.Equal(t, 1, removedCount)

	err = cache.Get(key1, &result)
	assert.NoError(t, err)
	assert.Equal(t, value1, result)
	err = cache.Get(key2, &result)
	assert.Error(t, err)
}

func newTestRegistry(t *testing.T) *Registry {
	t.Helper()

	tempDir, err := ioutil.TempDir("", "beat-data-dir-")
	require.NoError(t, err)

	t.Cleanup(func() { os.RemoveAll(tempDir) })

	return &Registry{
		path: filepath.Join(tempDir, cacheFile),
	}
}
