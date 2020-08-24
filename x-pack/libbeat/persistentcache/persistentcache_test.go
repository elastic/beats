// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package persistentcache

import (
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/tests/resources"
)

func TestPutGet(t *testing.T) {
	logp.TestingSetup()
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
	logp.TestingSetup()
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
	logp.TestingSetup()
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
	logp.TestingSetup()
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
	logp.TestingSetup()
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
	logp.TestingSetup()
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

var benchmarkCacheSizes = []int{10, 100, 1000, 10000, 100000}

func BenchmarkPut(b *testing.B) {
	type cache interface {
		Put(key string, value interface{}) error
		Close() error
	}

	registry := newTestRegistry(b)
	newStatestoreCache := func(tb testing.TB, name string) cache {
		cache, err := newCache(registry, name, Options{})
		require.NoError(tb, err)
		return cache
	}

	caches := []struct {
		name    string
		factory func(t testing.TB, name string) cache
	}{
		{name: "statestore", factory: newStatestoreCache},
	}

	b.Run("random strings", func(b *testing.B) {
		for _, c := range caches {
			b.Run(c.name, func(b *testing.B) {
				b.ReportAllocs()

				cache := c.factory(b, b.Name())
				defer cache.Close()

				value := uuid.Must(uuid.NewV4()).String()

				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					err := cache.Put(strconv.Itoa(i), value)
					if err != nil {
						b.Fatal(err)
					}
				}
				b.StopTimer()
			})
		}
	})

	b.Run("objects", func(b *testing.B) {
		for _, c := range caches {
			type entry struct {
				ID   string
				Data [128]byte
			}

			b.Run(c.name, func(b *testing.B) {
				b.ReportAllocs()

				cache := c.factory(b, b.Name())
				defer cache.Close()

				value := entry{ID: uuid.Must(uuid.NewV4()).String()}

				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					err := cache.Put(strconv.Itoa(i), value)
					if err != nil {
						b.Fatal(err)
					}
				}
				b.StopTimer()
			})
		}
	})

	b.Run("maps", func(b *testing.B) {
		for _, c := range caches {
			b.Run(c.name, func(b *testing.B) {
				b.ReportAllocs()

				cache := c.factory(b, b.Name())
				defer cache.Close()

				value := map[string]string{
					"id": uuid.Must(uuid.NewV4()).String(),
				}

				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					err := cache.Put(strconv.Itoa(i), value)
					if err != nil {
						b.Fatal(err)
					}
				}
				b.StopTimer()
			})
		}
	})

	for _, size := range benchmarkCacheSizes {
		b.Run(fmt.Sprintf("%d objects", size), func(b *testing.B) {
			type entry struct {
				ID   string
				Data [128]byte
			}
			objects := make([]entry, size)
			for i := 0; i < size; i++ {
				objects[i] = entry{
					ID: uuid.Must(uuid.NewV4()).String(),
				}
			}

			for _, c := range caches {
				b.Run(c.name, func(b *testing.B) {
					b.ReportAllocs()

					for i := 0; i < b.N; i++ {
						cache := c.factory(b, b.Name())
						for _, object := range objects {
							cache.Put(object.ID, object)
						}
						cache.Close()
					}
				})
			}
		})
	}
}

func BenchmarkOpen(b *testing.B) {
	type cache interface {
		Put(key string, value interface{}) error
		Close() error
	}

	registry := newTestRegistry(b)
	newStatestoreCache := func(tb testing.TB, name string) cache {
		cache, err := newCache(registry, name, Options{})
		require.NoError(tb, err)
		return cache
	}

	caches := map[string]struct {
		factory func(t testing.TB, name string) cache
	}{
		"statestore": {factory: newStatestoreCache},
	}

	for _, size := range benchmarkCacheSizes {
		b.Run(fmt.Sprintf("%d objects", size), func(b *testing.B) {
			type entry struct {
				ID   string
				Data [128]byte
			}

			for name, c := range caches {
				cache := c.factory(b, name)
				for i := 0; i < size; i++ {
					e := entry{
						ID: uuid.Must(uuid.NewV4()).String(),
					}
					err := cache.Put(e.ID, e)
					require.NoError(b, err)
				}
				cache.Close()

				b.Run(name, func(b *testing.B) {
					b.ReportAllocs()

					for i := 0; i < b.N; i++ {
						cache := c.factory(b, name)
						cache.Close()
					}
				})
			}
		})
	}
}

func BenchmarkGet(b *testing.B) {
	type cache interface {
		Put(key string, value interface{}) error
		Get(key string, value interface{}) error
		Close() error
	}

	registry := newTestRegistry(b)
	newStatestoreCache := func(tb testing.TB, name string) cache {
		cache, err := newCache(registry, name, Options{})
		require.NoError(tb, err)
		return cache
	}

	caches := []struct {
		name    string
		factory func(t testing.TB, name string) cache
	}{
		{name: "statestore", factory: newStatestoreCache},
	}

	for _, size := range benchmarkCacheSizes {
		b.Run(fmt.Sprintf("%d objects", size), func(b *testing.B) {
			for _, c := range caches {
				type entry struct {
					ID   string
					Data [128]byte
				}

				cacheName := b.Name()

				objects := make([]entry, size)
				cache := c.factory(b, cacheName)
				for i := 0; i < size; i++ {
					e := entry{
						ID: uuid.Must(uuid.NewV4()).String(),
					}
					objects[i] = e
					err := cache.Put(e.ID, e)
					require.NoError(b, err)
				}
				cache.Close()

				b.Run(c.name, func(b *testing.B) {
					b.ReportAllocs()

					cache := c.factory(b, cacheName)

					var result entry

					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						expected := objects[rand.Intn(size)]
						cache.Get(expected.ID, &result)
						if expected.ID != result.ID {
							b.FailNow()
						}
					}
					b.StopTimer()
					cache.Close()
				})
			}
		})
	}
}

func BenchmarkCleanup(b *testing.B) {
	type cache interface {
		PutWithTimeout(key string, value interface{}, d time.Duration) error
		CleanUp() int
		Close() error
	}

	registry := newTestRegistry(b)
	newStatestoreCache := func(tb testing.TB, name string, clock func() time.Time) cache {
		cache, err := newCache(registry, name, Options{})
		cache.clock = clock
		require.NoError(tb, err)
		return cache
	}

	caches := map[string]struct {
		factory func(t testing.TB, name string, clock func() time.Time) cache
	}{
		"statestore": {factory: newStatestoreCache},
	}

	for _, size := range benchmarkCacheSizes {
		b.Run(fmt.Sprintf("%d objects", size), func(b *testing.B) {

			type entry struct {
				ID   string
				Data [128]byte
			}

			for name, c := range caches {
				b.Run(name, func(b *testing.B) {
					b.ReportAllocs()

					now := time.Now()
					cache := c.factory(b, name, func() time.Time { return now })
					defer cache.Close()

					for i := 0; i < size/2; i++ {
						e := entry{
							ID: uuid.Must(uuid.NewV4()).String(),
						}
						err := cache.PutWithTimeout(e.ID, e, 10*time.Second)
						require.NoError(b, err)
					}
					now = now.Add(10 * time.Second)
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						b.StopTimer()
						for i := 0; i < size/2; i++ {
							e := entry{
								ID: uuid.Must(uuid.NewV4()).String(),
							}
							err := cache.PutWithTimeout(e.ID, e, 10*time.Second)
							require.NoError(b, err)
						}
						b.StartTimer()

						// At this point we should have ~size elements in the
						// cache, and ~size/2 elements expired.
						expired := cache.CleanUp()
						if expired == 0 {
							b.FailNow()
						}
						now = now.Add(10 * time.Second)
					}
				})

			}
		})
	}
}

func newTestRegistry(t testing.TB) *Registry {
	t.Helper()

	tempDir, err := ioutil.TempDir("", "beat-data-dir-")
	require.NoError(t, err)

	t.Cleanup(func() { os.RemoveAll(tempDir) })

	return &Registry{
		path: filepath.Join(tempDir, cacheFile),
	}
}

func dirSize(tb testing.TB, path string) int64 {
	var size int64

	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	require.NoError(tb, err)

	return size
}
