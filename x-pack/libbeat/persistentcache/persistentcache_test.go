// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package persistentcache

import (
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

	"github.com/elastic/beats/v8/libbeat/logp"
)

func TestPutGet(t *testing.T) {
	logp.TestingSetup()
	t.Parallel()

	cache, err := New("test", testOptions(t))
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

	options := testOptions(t)

	cache, err := New("test", options)
	require.NoError(t, err)

	type valueType struct {
		Something string
	}

	var key = "somekey"
	var value = valueType{Something: "foo"}

	err = cache.Put(key, value)
	assert.NoError(t, err)

	err = cache.Close()
	assert.NoError(t, err)

	cache, err = New("test", options)
	require.NoError(t, err)
	defer cache.Close()

	var result valueType
	err = cache.Get(key, &result)
	assert.NoError(t, err)
	assert.Equal(t, value, result)
}

func TestExpired(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	logp.TestingSetup()
	t.Parallel()

	options := testOptions(t)
	cache, err := New("test", options)
	require.NoError(t, err)
	defer cache.Close()

	type valueType struct {
		Something string
	}

	var key = "somekey"
	var value = valueType{Something: "foo"}

	// Badger TTL is not reliable on sub-second durations.
	err = cache.PutWithTimeout(key, value, 2*time.Second)
	assert.NoError(t, err)

	var result valueType
	err = cache.Get(key, &result)
	assert.NoError(t, err)
	assert.Equal(t, value, result)

	time.Sleep(2 * time.Second)
	err = cache.Get(key, &result)
	assert.Error(t, err)
}

func TestRefreshOnAccess(t *testing.T) {
	t.Skip("flaky test")

	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	logp.TestingSetup()
	t.Parallel()

	// Badger TTL is not reliable on sub-second durations.
	options := testOptions(t)
	options.Timeout = 2 * time.Second
	options.RefreshOnAccess = true

	cache, err := New("test", options)
	require.NoError(t, err)
	defer cache.Close()

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

	time.Sleep(1 * time.Second)

	var result valueType
	err = cache.Get(key1, &result)
	assert.NoError(t, err)
	assert.Equal(t, value1, result)

	time.Sleep(1 * time.Second)

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

	options := testOptions(b)
	newPersistentCache := func(tb testing.TB, name string) cache {
		cache, err := New(name, options)
		require.NoError(tb, err)
		return cache
	}

	caches := []struct {
		name    string
		factory func(t testing.TB, name string) cache
	}{
		{name: "badger", factory: newPersistentCache},
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

	options := testOptions(b)
	newPersistentCache := func(tb testing.TB, name string) cache {
		cache, err := New(name, options)
		require.NoError(tb, err)
		return cache
	}

	caches := []struct {
		name    string
		factory func(t testing.TB, name string) cache
	}{
		{name: "badger", factory: newPersistentCache},
	}

	for _, size := range benchmarkCacheSizes {
		b.Run(fmt.Sprintf("%d objects", size), func(b *testing.B) {
			type entry struct {
				ID   string
				Data [128]byte
			}

			for _, c := range caches {
				cacheName := b.Name()
				cache := c.factory(b, cacheName)
				for i := 0; i < size; i++ {
					e := entry{
						ID: uuid.Must(uuid.NewV4()).String(),
					}
					err := cache.Put(e.ID, e)
					require.NoError(b, err)
				}
				cache.Close()

				b.Run(c.name, func(b *testing.B) {
					b.ReportAllocs()

					for i := 0; i < b.N; i++ {
						cache := c.factory(b, cacheName)
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

	options := testOptions(b)
	newPersistentCache := func(tb testing.TB, name string) cache {
		cache, err := New(name, options)
		require.NoError(tb, err)
		return cache
	}

	caches := []struct {
		name    string
		factory func(t testing.TB, name string) cache
	}{
		{name: "badger", factory: newPersistentCache},
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
							b.Fatalf("%s != %s", expected.ID, result.ID)
						}
					}
					b.StopTimer()
					cache.Close()
				})
			}
		})
	}
}

func testOptions(t testing.TB) Options {
	t.Helper()

	tempDir, err := ioutil.TempDir("", "beat-data-dir-")
	require.NoError(t, err)

	t.Cleanup(func() { os.RemoveAll(tempDir) })

	return Options{
		RootPath: filepath.Join(tempDir, cacheFile),
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
