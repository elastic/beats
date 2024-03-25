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

package cache

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/elastic/elastic-agent-libs/logp"
)

var keep = flag.Bool("keep", false, "keep testdata after test complete")

type fileStoreTestSteps struct {
	doTo func(*fileStore) error
	want *fileStore
}

//nolint:errcheck // Paul Hogan was right.
var fileStoreTests = []struct {
	name          string
	cfg           config
	want          *fileStore
	steps         []fileStoreTestSteps
	wantPersisted []*CacheEntry
}{
	{
		name: "new_put",
		cfg: config{
			Store: &storeConfig{
				File:     &fileConfig{ID: "test"},
				Capacity: 1000,
				Effort:   10,
			},
			Put: &putConfig{
				TTL: ptrTo(time.Second),
			},
		},
		want: &fileStore{path: "testdata/new_put", memStore: memStore{
			id:     "test",
			cache:  map[string]*CacheEntry{},
			refs:   1,
			ttl:    time.Second,
			cap:    1000,
			effort: 10,
		}},
	},
	{
		name: "new_get",
		cfg: config{
			Store: &storeConfig{
				File:     &fileConfig{ID: "test"},
				Capacity: 1000,
				Effort:   10,
			},
			Get: &getConfig{},
		},
		want: &fileStore{path: "testdata/new_get", memStore: memStore{
			id:    "test",
			cache: map[string]*CacheEntry{},
			refs:  1,
			// TTL, capacity and effort are set only by put.
			ttl:    -1,
			cap:    -1,
			effort: -1,
		}},
	},
	{
		name: "new_delete",
		cfg: config{
			Store: &storeConfig{
				File:     &fileConfig{ID: "test"},
				Capacity: 1000,
				Effort:   10,
			},
			Delete: &delConfig{},
		},
		want: &fileStore{path: "testdata/new_delete", memStore: memStore{
			id:    "test",
			cache: map[string]*CacheEntry{},
			refs:  1,
			// TTL, capacity and effort are set only by put.
			ttl:    -1,
			cap:    -1,
			effort: -1,
		}},
	},
	{
		name: "new_get_add_put",
		cfg: config{
			Store: &storeConfig{
				File:     &fileConfig{ID: "test"},
				Capacity: 1000,
				Effort:   10,
			},
			Get: &getConfig{},
		},
		want: &fileStore{path: "testdata/new_get_add_put", memStore: memStore{
			id:    "test",
			cache: map[string]*CacheEntry{},
			// TTL, capacity and effort are set only by put.
			refs:   1,
			ttl:    -1,
			cap:    -1,
			effort: -1,
		}},
		steps: []fileStoreTestSteps{
			0: {
				doTo: func(s *fileStore) error {
					putCfg := config{
						Store: &storeConfig{
							File:     &fileConfig{ID: "test"},
							Capacity: 1000,
							Effort:   10,
						},
						Put: &putConfig{
							TTL: ptrTo(time.Second),
						},
					}
					s.add(putCfg)
					return nil
				},
				want: &fileStore{path: "testdata/new_get_add_put", memStore: memStore{
					id:     "test",
					cache:  map[string]*CacheEntry{},
					refs:   2,
					ttl:    time.Second,
					cap:    1000,
					effort: 10,
				}},
			},
		},
	},
	{
		name: "ensemble",
		cfg: config{
			Store: &storeConfig{
				File:     &fileConfig{ID: "test"},
				Capacity: 1000,
				Effort:   10,
			},
			Get: &getConfig{},
		},
		want: &fileStore{path: "testdata/ensemble", memStore: memStore{
			id:    "test",
			cache: map[string]*CacheEntry{},
			refs:  1,
			// TTL, capacity and effort are set only by put.
			ttl:    -1,
			cap:    -1,
			effort: -1,
		}},
		steps: []fileStoreTestSteps{
			0: {
				doTo: func(s *fileStore) error {
					putCfg := config{
						Store: &storeConfig{
							File:     &fileConfig{ID: "test"},
							Capacity: 1000,
							Effort:   10,
						},
						Put: &putConfig{
							TTL: ptrTo(time.Second),
						},
					}
					s.add(putCfg)
					return nil
				},
				want: &fileStore{path: "testdata/ensemble", memStore: memStore{
					id:     "test",
					cache:  map[string]*CacheEntry{},
					refs:   2,
					dirty:  false,
					ttl:    time.Second,
					cap:    1000,
					effort: 10,
				}},
			},
			1: {
				doTo: func(s *fileStore) error {
					s.Put("one", 1)
					s.Put("two", 2)
					s.Put("three", 3)
					return nil
				},
				want: &fileStore{path: "testdata/ensemble", memStore: memStore{
					id: "test",
					cache: map[string]*CacheEntry{
						"one":   {Key: "one", Value: int(1), index: 0},
						"two":   {Key: "two", Value: int(2), index: 1},
						"three": {Key: "three", Value: int(3), index: 2},
					},
					expiries: expiryHeap{
						{Key: "one", Value: int(1), index: 0},
						{Key: "two", Value: int(2), index: 1},
						{Key: "three", Value: int(3), index: 2},
					},
					refs:   2,
					dirty:  true,
					ttl:    time.Second,
					cap:    1000,
					effort: 10,
				}},
			},
			2: {
				doTo: func(s *fileStore) error {
					got, err := s.Get("two")
					if got != 2 {
						return fmt.Errorf(`unexpected result from Get("two"): got:%v want:2`, got)
					}
					return err
				},
				want: &fileStore{path: "testdata/ensemble", memStore: memStore{
					id: "test",
					cache: map[string]*CacheEntry{
						"one":   {Key: "one", Value: int(1), index: 0},
						"two":   {Key: "two", Value: int(2), index: 1},
						"three": {Key: "three", Value: int(3), index: 2},
					},
					expiries: expiryHeap{
						{Key: "one", Value: int(1), index: 0},
						{Key: "two", Value: int(2), index: 1},
						{Key: "three", Value: int(3), index: 2},
					},
					refs:   2,
					dirty:  true,
					ttl:    time.Second,
					cap:    1000,
					effort: 10,
				}},
			},
			3: {
				doTo: func(s *fileStore) error {
					return s.Delete("two")
				},
				want: &fileStore{path: "testdata/ensemble", memStore: memStore{
					id: "test",
					cache: map[string]*CacheEntry{
						"one":   {Key: "one", Value: int(1), index: 0},
						"three": {Key: "three", Value: int(3), index: 1},
					},
					expiries: expiryHeap{
						{Key: "one", Value: int(1), index: 0},
						{Key: "three", Value: int(3), index: 1},
					},
					refs:   2,
					dirty:  true,
					ttl:    time.Second,
					cap:    1000,
					effort: 10,
				}},
			},
			4: {
				doTo: func(s *fileStore) error {
					got, _ := s.Get("two")
					if got != nil {
						return fmt.Errorf(`unexpected result from Get("two") after deletion: got:%v want:nil`, got)
					}
					return nil
				},
				want: &fileStore{path: "testdata/ensemble", memStore: memStore{
					id: "test",
					cache: map[string]*CacheEntry{
						"one":   {Key: "one", Value: int(1), index: 0},
						"three": {Key: "three", Value: int(3), index: 1},
					},
					expiries: expiryHeap{
						{Key: "one", Value: int(1), index: 0},
						{Key: "three", Value: int(3), index: 1},
					},
					refs:   2,
					dirty:  true,
					ttl:    time.Second,
					cap:    1000,
					effort: 10,
				}},
			},
			5: {
				doTo: func(s *fileStore) error {
					s.dropFrom(&fileStores)
					if !fileStores.has(s.id) {
						return fmt.Errorf("%q fileStore not found after single close", s.id)
					}
					return nil
				},
				want: &fileStore{path: "testdata/ensemble", memStore: memStore{
					id: "test",
					cache: map[string]*CacheEntry{
						"one":   {Key: "one", Value: int(1), index: 0},
						"three": {Key: "three", Value: int(3), index: 1},
					},
					expiries: expiryHeap{
						{Key: "one", Value: int(1), index: 0},
						{Key: "three", Value: int(3), index: 1},
					},
					refs:   1,
					dirty:  true,
					ttl:    time.Second,
					cap:    1000,
					effort: 10,
				}},
			},
			6: {
				doTo: func(s *fileStore) error {
					s.dropFrom(&fileStores)
					if fileStores.has(s.id) {
						return fmt.Errorf("%q fileStore still found after double close", s.id)
					}
					return nil
				},
				want: &fileStore{path: "testdata/ensemble", memStore: memStore{
					id:       "test",
					cache:    nil, // assistively nil-ed.
					expiries: nil, // assistively nil-ed.
					refs:     0,
					dirty:    false,
					ttl:      time.Second,
					cap:      1000,
					effort:   10,
				}},
			},
		},
		wantPersisted: []*CacheEntry{
			// Numeric values are float due to JSON round-trip.
			{Key: "one", Value: 1.0},
			{Key: "three", Value: 3.0},
		},
	},
}

func TestFileStore(t *testing.T) {
	err := os.RemoveAll("testdata")
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("failed to clear testdata directory: %v", err)
	}
	err = os.Mkdir("testdata", 0o755)
	if err != nil && !errors.Is(err, fs.ErrExist) {
		t.Fatalf("failed to create testdata directory: %v", err)
	}
	if !*keep {
		t.Cleanup(func() { os.RemoveAll("testdata") })
	}

	allow := cmp.AllowUnexported(fileStore{}, memStore{}, CacheEntry{})
	ignoreInFileStore := cmpopts.IgnoreFields(fileStore{}, "cancel", "log")
	ignoreInMemStore := cmpopts.IgnoreFields(memStore{}, "mu")
	ignoreInCacheEntry := cmpopts.IgnoreFields(CacheEntry{}, "Expires")

	for _, test := range fileStoreTests {
		t.Run(test.name, func(t *testing.T) {
			// Construct the store and put in into the stores map as
			// we would if we were calling Run.
			path := filepath.Join("testdata", test.name)
			store := newFileStore(test.cfg, test.cfg.Store.File.ID, path, logp.L())
			store.add(test.cfg)
			fileStores.add(store)

			if !cmp.Equal(test.want, store, allow, ignoreInFileStore, ignoreInMemStore) {
				t.Errorf("unexpected new fileStore result:\n--- want\n+++ got\n%s",
					cmp.Diff(test.want, store, allow, ignoreInFileStore, ignoreInMemStore))
			}
			for i, step := range test.steps {
				err := step.doTo(store)
				if err != nil {
					t.Errorf("unexpected error at step %d: %v", i, err)
				}
				if !cmp.Equal(step.want, store, allow, ignoreInFileStore, ignoreInMemStore, ignoreInCacheEntry) {
					t.Errorf("unexpected fileStore step %d result:\n--- want\n+++ got\n%s",
						i, cmp.Diff(step.want, store, allow, ignoreInFileStore, ignoreInMemStore, ignoreInCacheEntry))
				}
			}
			if test.wantPersisted == nil {
				return
			}

			f, err := os.Open(path)
			if err != nil {
				t.Fatalf("failed to open persisted data: %v", err)
			}
			defer f.Close()
			dec := json.NewDecoder(f)
			var got []*CacheEntry
			for {
				var e CacheEntry
				err = dec.Decode(&e)
				if err != nil {
					if !errors.Is(err, io.EOF) {
						t.Fatalf("unexpected error reading persisted cache data: %v", err)
					}
					break
				}
				got = append(got, &e)
			}
			if !cmp.Equal(test.wantPersisted, got, allow, ignoreInCacheEntry) {
				t.Errorf("unexpected persisted state:\n--- want\n+++ got\n%s",
					cmp.Diff(test.wantPersisted, got, allow, ignoreInCacheEntry))
			}
			wantCache := make(map[string]*CacheEntry)
			for _, e := range got {
				wantCache[e.Key] = e
			}
			store = newFileStore(test.cfg, test.cfg.Store.File.ID, path, logp.L())
			// Specialise the in cache entry ignore list to include index.
			ignoreMoreInCacheEntry := cmpopts.IgnoreFields(CacheEntry{}, "Expires", "index")
			if !cmp.Equal(wantCache, store.cache, allow, ignoreMoreInCacheEntry) {
				t.Errorf("unexpected restored state:\n--- want\n+++ got\n%s",
					cmp.Diff(wantCache, store.cache, allow, ignoreMoreInCacheEntry))
			}
			for k, e := range store.cache {
				if e.index < 0 || len(store.expiries) <= e.index {
					t.Errorf("cache entry %s index out of bounds: got:%d [0,%d)", k, e.index, len(store.expiries))
					continue
				}
				if !cmp.Equal(e, store.expiries[e.index], allow, ignoreInCacheEntry) {
					t.Errorf("unexpected mismatched cache/expiry state %s:\n--- want\n+++ got\n%s",
						k, cmp.Diff(e, store.expiries[e.index], allow, ignoreInCacheEntry))
				}
			}
		})
	}
}

// add adds the store to the set. It is used only for testing.
func (s *fileStoreSet) add(store *fileStore) {
	s.mu.Lock()
	s.stores[store.id] = store
	s.mu.Unlock()
}

// has returns whether the store exists in the set. It is used only for testing.
func (s *fileStoreSet) has(id string) bool {
	s.mu.Lock()
	_, ok := s.stores[id]
	s.mu.Unlock()
	return ok
}
