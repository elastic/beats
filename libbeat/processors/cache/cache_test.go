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
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/elastic/beats/v7/libbeat/beat"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

type cacheTestStep struct {
	event        mapstr.M
	want         mapstr.M
	wantCacheVal map[string]*CacheEntry
	wantErr      error
}

var cacheTests = []struct {
	name        string
	configs     []testConfig
	wantInitErr error
	steps       []cacheTestStep
}{
	{
		name: "invalid_no_backend",
		configs: []testConfig{
			{
				cfg: mapstr.M{
					"put": mapstr.M{
						"key_field":   "crowdstrike.aid",
						"ttl":         "168h",
						"value_field": "crowdstrike.metadata",
					},
				},
			},
		},
		wantInitErr: errors.New("failed to unpack the cache configuration: missing required field accessing 'backend'"),
	},
	{
		name: "invalid_no_key_field",
		configs: []testConfig{
			{
				cfg: mapstr.M{
					"backend": mapstr.M{
						"memory": mapstr.M{
							"id": "aidmaster",
						},
					},
					"put": mapstr.M{
						"value_field": "crowdstrike.metadata",
						"ttl":         "168h",
					},
				},
			},
		},
		wantInitErr: errors.New("failed to unpack the cache configuration: string value is not set accessing 'put.key_field'"),
	},
	{
		name: "invalid_no_value_field",
		configs: []testConfig{
			{
				cfg: mapstr.M{
					"backend": mapstr.M{
						"memory": mapstr.M{
							"id": "aidmaster",
						},
					},
					"put": mapstr.M{
						"key_field": "crowdstrike.aid",
						"ttl":       "168h",
					},
				},
			},
		},
		wantInitErr: errors.New("failed to unpack the cache configuration: string value is not set accessing 'put.value_field'"),
	},
	{
		name: "put_value",
		configs: []testConfig{
			{
				cfg: mapstr.M{
					"backend": mapstr.M{
						"memory": mapstr.M{
							"id": "aidmaster",
						},
					},
					"put": mapstr.M{
						"key_field":   "crowdstrike.aid",
						"value_field": "crowdstrike.metadata",
						"ttl":         "168h",
					},
				},
			},
		},
		wantInitErr: nil,
		steps: []cacheTestStep{
			{
				event: mapstr.M{
					"crowdstrike": mapstr.M{
						"aid":      "one",
						"metadata": "metadata_value",
					},
				},
				want: mapstr.M{
					"crowdstrike": mapstr.M{
						"aid":      "one",
						"metadata": "metadata_value",
					},
				},
				wantCacheVal: map[string]*CacheEntry{
					"one": {Key: "one", Value: "metadata_value"},
				},
				wantErr: nil,
			},
		},
	},
	{
		name: "put_and_get_value",
		configs: []testConfig{
			{
				when: func(e mapstr.M) bool {
					return e["put"] == true
				},
				cfg: mapstr.M{
					"backend": mapstr.M{
						"memory": mapstr.M{
							"id": "aidmaster",
						},
					},
					"put": mapstr.M{
						"key_field":   "crowdstrike.aid",
						"value_field": "crowdstrike.metadata",
						"ttl":         "168h",
					},
				},
			},
			{
				when: func(e mapstr.M) bool {
					return e["get"] == true
				},
				cfg: mapstr.M{
					"backend": mapstr.M{
						"memory": mapstr.M{
							"id": "aidmaster",
						},
					},
					"get": mapstr.M{
						"key_field":    "crowdstrike.aid",
						"target_field": "crowdstrike.metadata_new",
					},
				},
			},
		},
		wantInitErr: nil,
		steps: []cacheTestStep{
			{
				event: mapstr.M{
					"put": true,
					"crowdstrike": mapstr.M{
						"aid":      "one",
						"metadata": "metadata_value",
					},
				},
				want: mapstr.M{
					"put": true,
					"crowdstrike": mapstr.M{
						"aid":      "one",
						"metadata": "metadata_value",
					},
				},
				wantCacheVal: map[string]*CacheEntry{
					"one": {Key: "one", Value: "metadata_value"},
				},
				wantErr: nil,
			},
			{
				event: mapstr.M{
					"get": true,
					"crowdstrike": mapstr.M{
						"aid": "one",
					},
				},
				want: mapstr.M{
					"get": true,
					"crowdstrike": mapstr.M{
						"aid":          "one",
						"metadata_new": "metadata_value",
					},
				},
				wantCacheVal: map[string]*CacheEntry{
					"one": {Key: "one", Value: "metadata_value"},
				},
				wantErr: nil,
			},
		},
	},
	{
		name: "put_and_get_value_reverse_config",
		configs: []testConfig{
			{
				when: func(e mapstr.M) bool {
					return e["get"] == true
				},
				cfg: mapstr.M{
					"backend": mapstr.M{
						"memory": mapstr.M{
							"id": "aidmaster",
						},
					},
					"get": mapstr.M{
						"key_field":    "crowdstrike.aid",
						"target_field": "crowdstrike.metadata_new",
					},
				},
			},
			{
				when: func(e mapstr.M) bool {
					return e["put"] == true
				},
				cfg: mapstr.M{
					"backend": mapstr.M{
						"memory": mapstr.M{
							"id": "aidmaster",
						},
					},
					"put": mapstr.M{
						"key_field":   "crowdstrike.aid",
						"value_field": "crowdstrike.metadata",
						"ttl":         "168h",
					},
				},
			},
		},
		wantInitErr: nil,
		steps: []cacheTestStep{
			{
				event: mapstr.M{
					"put": true,
					"crowdstrike": mapstr.M{
						"aid":      "one",
						"metadata": "metadata_value",
					},
				},
				want: mapstr.M{
					"put": true,
					"crowdstrike": mapstr.M{
						"aid":      "one",
						"metadata": "metadata_value",
					},
				},
				wantCacheVal: map[string]*CacheEntry{
					"one": {Key: "one", Value: "metadata_value"},
				},
				wantErr: nil,
			},
			{
				event: mapstr.M{
					"get": true,
					"crowdstrike": mapstr.M{
						"aid": "one",
					},
				},
				want: mapstr.M{
					"get": true,
					"crowdstrike": mapstr.M{
						"aid":          "one",
						"metadata_new": "metadata_value",
					},
				},
				wantCacheVal: map[string]*CacheEntry{
					"one": {Key: "one", Value: "metadata_value"},
				},
				wantErr: nil,
			},
		},
	},
	{
		name: "put_and_get_value_with_get_error_no_overwrite",
		configs: []testConfig{
			{
				when: func(e mapstr.M) bool {
					return e["put"] == true
				},
				cfg: mapstr.M{
					"backend": mapstr.M{
						"memory": mapstr.M{
							"id": "aidmaster",
						},
					},
					"put": mapstr.M{
						"key_field":   "crowdstrike.aid",
						"value_field": "crowdstrike.metadata",
						"ttl":         "168h",
					},
				},
			},
			{
				when: func(e mapstr.M) bool {
					return e["get"] == true
				},
				cfg: mapstr.M{
					"backend": mapstr.M{
						"memory": mapstr.M{
							"id": "aidmaster",
						},
					},
					"get": mapstr.M{
						"key_field":    "crowdstrike.aid",
						"target_field": "crowdstrike.metadata_new",
					},
				},
			},
		},
		wantInitErr: nil,
		steps: []cacheTestStep{
			{
				event: mapstr.M{
					"put": true,
					"crowdstrike": mapstr.M{
						"aid":      "one",
						"metadata": "metadata_value",
					},
				},
				want: mapstr.M{
					"put": true,
					"crowdstrike": mapstr.M{
						"aid":      "one",
						"metadata": "metadata_value",
					},
				},
				wantCacheVal: map[string]*CacheEntry{
					"one": {Key: "one", Value: "metadata_value"},
				},
				wantErr: nil,
			},
			{
				event: mapstr.M{
					"get": true,
					"crowdstrike": mapstr.M{
						"aid": "one",
						"metadata_new": mapstr.M{
							"someone_is_already_here": mapstr.M{
								"another_key": "value",
							},
						},
					},
				},
				want: mapstr.M{
					"get": true,
					"crowdstrike": mapstr.M{
						"aid": "one",
						"metadata_new": mapstr.M{
							"someone_is_already_here": mapstr.M{
								"another_key": "value",
							},
						},
					},
				},
				wantCacheVal: map[string]*CacheEntry{
					"one": {Key: "one", Value: "metadata_value"},
				},
				wantErr: errors.New("error applying cache get processor: target field 'crowdstrike.metadata_new' already exists and overwrite_keys is false"),
			},
		},
	},
	{
		name: "put_and_get_value_allow_overwrite",
		configs: []testConfig{
			{
				when: func(e mapstr.M) bool {
					return e["put"] == true
				},
				cfg: mapstr.M{
					"backend": mapstr.M{
						"memory": mapstr.M{
							"id": "aidmaster",
						},
					},
					"put": mapstr.M{
						"key_field":   "crowdstrike.aid",
						"value_field": "crowdstrike.metadata",
						"ttl":         "168h",
					},
				},
			},
			{
				when: func(e mapstr.M) bool {
					return e["get"] == true
				},
				cfg: mapstr.M{
					"backend": mapstr.M{
						"memory": mapstr.M{
							"id": "aidmaster",
						},
					},
					"overwrite_keys": true,
					"get": mapstr.M{
						"key_field":    "crowdstrike.aid",
						"target_field": "crowdstrike.metadata_new",
					},
				},
			},
		},
		wantInitErr: nil,
		steps: []cacheTestStep{
			{
				event: mapstr.M{
					"put": true,
					"crowdstrike": mapstr.M{
						"aid":      "one",
						"metadata": "metadata_value",
					},
				},
				want: mapstr.M{
					"put": true,
					"crowdstrike": mapstr.M{
						"aid":      "one",
						"metadata": "metadata_value",
					},
				},
				wantCacheVal: map[string]*CacheEntry{
					"one": {Key: "one", Value: "metadata_value"},
				},
				wantErr: nil,
			},
			{
				event: mapstr.M{
					"get": true,
					"crowdstrike": mapstr.M{
						"aid": "one",
						"metadata_new": mapstr.M{
							"someone_is_already_here": mapstr.M{
								"another_key": "value",
							},
						},
					},
				},
				want: mapstr.M{
					"get": true,
					"crowdstrike": mapstr.M{
						"aid":          "one",
						"metadata_new": "metadata_value",
					},
				},
				wantCacheVal: map[string]*CacheEntry{
					"one": {Key: "one", Value: "metadata_value"},
				},
				wantErr: nil,
			},
		},
	},
	{
		name: "put_and_get_value_allow_overwrite_but_get_error",
		configs: []testConfig{
			{
				when: func(e mapstr.M) bool {
					return e["put"] == true
				},
				cfg: mapstr.M{
					"backend": mapstr.M{
						"memory": mapstr.M{
							"id": "aidmaster",
						},
					},
					"put": mapstr.M{
						"key_field":   "crowdstrike.aid",
						"value_field": "crowdstrike.metadata",
						"ttl":         "168h",
					},
				},
			},
			{
				when: func(e mapstr.M) bool {
					return e["get"] == true
				},
				cfg: mapstr.M{
					"backend": mapstr.M{
						"memory": mapstr.M{
							"id": "aidmaster",
						},
					},
					"overwrite_keys": true,
					"get": mapstr.M{
						"key_field":    "crowdstrike.aid",
						"target_field": "crowdstrike.metadata_new.child",
					},
				},
			},
		},
		wantInitErr: nil,
		steps: []cacheTestStep{
			{
				event: mapstr.M{
					"put": true,
					"crowdstrike": mapstr.M{
						"aid":      "one",
						"metadata": "metadata_value",
					},
				},
				want: mapstr.M{
					"put": true,
					"crowdstrike": mapstr.M{
						"aid":      "one",
						"metadata": "metadata_value",
					},
				},
				wantCacheVal: map[string]*CacheEntry{
					"one": {Key: "one", Value: "metadata_value"},
				},
				wantErr: nil,
			},
			{
				event: mapstr.M{
					"get": true,
					"crowdstrike": mapstr.M{
						"aid":          "one",
						"metadata_new": "someone_is_already_here",
					},
				},
				want: mapstr.M{
					"get": true,
					"crowdstrike": mapstr.M{
						"aid":          "one",
						"metadata_new": "someone_is_already_here",
					},
				},
				wantCacheVal: map[string]*CacheEntry{
					"one": {Key: "one", Value: "metadata_value"},
				},
				wantErr: errors.New("error applying cache get processor: expected map but type is string"),
			},
		},
	},
}

type testConfig struct {
	when func(e mapstr.M) bool
	cfg  mapstr.M
}

func TestCache(t *testing.T) {
	logp.TestingSetup(logp.WithSelectors(name))
	for _, test := range cacheTests {
		t.Run(test.name, func(t *testing.T) {
			var processors []beat.Processor
			for i, cfg := range test.configs {
				config, err := conf.NewConfigFrom(cfg.cfg)
				if err != nil {
					t.Fatal(err)
				}

				p, err := New(config)
				if !sameError(err, test.wantInitErr) {
					t.Errorf("unexpected error from New: got:%v want:%v", err, test.wantInitErr)
				}
				if err != nil {
					return
				}

				t.Log(p)
				c, ok := p.(*cache)
				if !ok {
					t.Fatalf("processor %d is not an *cache", i)
				}

				defer func() {
					err := c.Close()
					if err != nil {
						t.Errorf("unexpected error from c.Close(): %v", err)
					}
				}()

				processors = append(processors, p)
			}

			for i, step := range test.steps {
				for j, p := range processors {
					if test.configs[j].when != nil && !test.configs[j].when(step.event) {
						continue
					}
					got, err := p.Run(&beat.Event{
						Fields: step.event,
					})
					if !sameError(err, step.wantErr) {
						t.Errorf("unexpected error from Run: got:%v want:%v", err, step.wantErr)
						return
					}
					if !cmp.Equal(step.want, got.Fields) {
						t.Errorf("unexpected result %d\n--- want\n+++ got\n%s", i, cmp.Diff(step.want, got.Fields))
					}
					switch got := p.(*cache).store.(type) {
					case *memStore:
						allow := cmp.AllowUnexported(CacheEntry{})
						ignore := cmpopts.IgnoreFields(CacheEntry{}, "Expires", "index")
						if !cmp.Equal(step.wantCacheVal, got.cache, allow, ignore) {
							t.Errorf("unexpected cache state result %d:\n--- want\n+++ got\n%s", i, cmp.Diff(step.wantCacheVal, got.cache, allow, ignore))
						}
					}
				}
			}
		})
	}
}
