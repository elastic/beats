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
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

type memStoreTestSteps struct {
	doTo func(*memStore) error
	want *memStore
}

var memStoreTests = []struct {
	name  string
	cfg   config
	want  *memStore
	steps []memStoreTestSteps
}{
	{
		name: "new_put",
		cfg: config{
			Store: &storeConfig{
				Memory:   &id{"test"},
				Capacity: 1000,
				Effort:   10,
			},
			Put: &putConfig{
				TTL: ptrTo(time.Second),
			},
		},
		want: &memStore{
			cache:  map[string]*CacheEntry{},
			ttl:    time.Second,
			cap:    1000,
			effort: 10,
		},
	},
	{
		name: "new_get",
		cfg: config{
			Store: &storeConfig{
				Memory:   &id{"test"},
				Capacity: 1000,
				Effort:   10,
			},
			Get: &getConfig{},
		},
		want: &memStore{
			cache: map[string]*CacheEntry{},
			// TTL, capacity and effort are set only by put.
			ttl:    -1,
			cap:    -1,
			effort: -1,
		},
	},
	{
		name: "new_delete",
		cfg: config{
			Store: &storeConfig{
				Memory:   &id{"test"},
				Capacity: 1000,
				Effort:   10,
			},
			Delete: &delConfig{},
		},
		want: &memStore{
			cache: map[string]*CacheEntry{},
			// TTL, capacity and effort are set only by put.
			ttl:    -1,
			cap:    -1,
			effort: -1,
		},
	},
	{
		name: "new_get_add_put",
		cfg: config{
			Store: &storeConfig{
				Memory:   &id{"test"},
				Capacity: 1000,
				Effort:   10,
			},
			Get: &getConfig{},
		},
		want: &memStore{
			cache: map[string]*CacheEntry{},
			// TTL, capacity and effort are set only by put.
			ttl:    -1,
			cap:    -1,
			effort: -1,
		},
		steps: []memStoreTestSteps{
			0: {
				doTo: func(s *memStore) error {
					putCfg := config{
						Store: &storeConfig{
							Memory:   &id{"test"},
							Capacity: 1000,
							Effort:   10,
						},
						Put: &putConfig{
							TTL: ptrTo(time.Second),
						},
					}
					s.setPutOptions(putCfg)
					return nil
				},
				want: &memStore{
					cache:  map[string]*CacheEntry{},
					ttl:    time.Second,
					cap:    1000,
					effort: 10,
				},
			},
		},
	},
	{
		name: "ensemble",
		cfg: config{
			Store: &storeConfig{
				Memory:   &id{"test"},
				Capacity: 1000,
				Effort:   10,
			},
			Get: &getConfig{},
		},
		want: &memStore{
			cache: map[string]*CacheEntry{},
			// TTL, capacity and effort are set only by put.
			ttl:    -1,
			cap:    -1,
			effort: -1,
		},
		steps: []memStoreTestSteps{
			0: {
				doTo: func(s *memStore) error {
					putCfg := config{
						Store: &storeConfig{
							Memory:   &id{"test"},
							Capacity: 1000,
							Effort:   10,
						},
						Put: &putConfig{
							TTL: ptrTo(time.Second),
						},
					}
					s.setPutOptions(putCfg)
					return nil
				},
				want: &memStore{
					cache:  map[string]*CacheEntry{},
					ttl:    time.Second,
					cap:    1000,
					effort: 10,
				},
			},
			1: {
				doTo: func(s *memStore) error {
					s.Put("one", 1)
					s.Put("two", 2)
					s.Put("three", 3)
					return nil
				},
				want: &memStore{
					cache: map[string]*CacheEntry{
						"one":   {key: "one", value: int(1), index: 0},
						"two":   {key: "two", value: int(2), index: 1},
						"three": {key: "three", value: int(3), index: 2},
					},
					expiries: expiryHeap{
						{key: "one", value: int(1), index: 0},
						{key: "two", value: int(2), index: 1},
						{key: "three", value: int(3), index: 2},
					},
					ttl:    time.Second,
					cap:    1000,
					effort: 10,
				},
			},
			2: {
				doTo: func(s *memStore) error {
					got, err := s.Get("two")
					if got != 2 {
						return fmt.Errorf(`unexpected result from Get("two"): got:%v want:2`, got)
					}
					return err
				},
				want: &memStore{
					cache: map[string]*CacheEntry{
						"one":   {key: "one", value: int(1), index: 0},
						"two":   {key: "two", value: int(2), index: 1},
						"three": {key: "three", value: int(3), index: 2},
					},
					expiries: expiryHeap{
						{key: "one", value: int(1), index: 0},
						{key: "two", value: int(2), index: 1},
						{key: "three", value: int(3), index: 2},
					},
					ttl:    time.Second,
					cap:    1000,
					effort: 10,
				},
			},
			3: {
				doTo: func(s *memStore) error {
					return s.Delete("two")
				},
				want: &memStore{
					cache: map[string]*CacheEntry{
						"one":   {key: "one", value: int(1), index: 0},
						"three": {key: "three", value: int(3), index: 1},
					},
					expiries: expiryHeap{
						{key: "one", value: int(1), index: 0},
						{key: "three", value: int(3), index: 1},
					},
					ttl:    time.Second,
					cap:    1000,
					effort: 10,
				},
			},
			4: {
				doTo: func(s *memStore) error {
					got, _ := s.Get("two")
					if got != nil {
						return fmt.Errorf(`unexpected result from Get("two") after deletion: got:%v want:nil`, got)
					}
					return nil
				},
				want: &memStore{
					cache: map[string]*CacheEntry{
						"one":   {key: "one", value: int(1), index: 0},
						"three": {key: "three", value: int(3), index: 1},
					},
					expiries: expiryHeap{
						{key: "one", value: int(1), index: 0},
						{key: "three", value: int(3), index: 1},
					},
					ttl:    time.Second,
					cap:    1000,
					effort: 10,
				},
			},
		},
	},
}

func TestMemStore(t *testing.T) {
	allow := cmp.AllowUnexported(memStore{}, CacheEntry{})
	ignoreInMemStore := cmpopts.IgnoreFields(memStore{}, "mu")
	ignoreInCacheEntry := cmpopts.IgnoreFields(CacheEntry{}, "expires")

	for _, test := range memStoreTests {
		t.Run(test.name, func(t *testing.T) {
			store := newMemStore(test.cfg)
			if !cmp.Equal(test.want, store, allow, ignoreInMemStore) {
				t.Errorf("unexpected new memStore result:\n--- want\n+++ got\n%s",
					cmp.Diff(test.want, store, allow, ignoreInMemStore))
			}
			for i, step := range test.steps {
				err := step.doTo(store)
				if err != nil {
					t.Errorf("unexpected error at step %d: %v", i, err)
				}
				if !cmp.Equal(step.want, store, allow, ignoreInMemStore, ignoreInCacheEntry) {
					t.Errorf("unexpected memStore step %d result:\n--- want\n+++ got\n%s",
						i, cmp.Diff(step.want, store, allow, ignoreInMemStore, ignoreInCacheEntry))
				}
			}
		})
	}
}

func ptrTo[T any](v T) *T { return &v }
