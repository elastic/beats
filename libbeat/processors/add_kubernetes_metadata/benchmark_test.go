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

//go:build linux || darwin || windows

package add_kubernetes_metadata

import (
	"testing"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func BenchmarkKubernetesAnnotatorRun(b *testing.B) {
	cfg := config.MustNewConfigFrom(map[string]interface{}{
		"lookup_fields": []string{"container.id"},
	})
	matcher, err := NewFieldMatcher(*cfg, logptest.NewTestingLogger(b, ""))
	if err != nil {
		b.Fatal(err)
	}

	processor := &kubernetesAnnotator{
		log:   logptest.NewTestingLogger(b, selector),
		cache: newCache(10 * time.Second),
		matchers: &Matchers{
			matchers: []Matcher{matcher},
		},
		kubernetesAvailable: true,
	}

	const cacheKey = "abc123container"

	processor.cache.set(cacheKey, mapstr.M{
		"kubernetes": mapstr.M{
			"pod": mapstr.M{
				"name":      "test-pod",
				"uid":       "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
				"namespace": "default",
				"labels": mapstr.M{
					"app":     "myapp",
					"version": "v1.2.3",
					"env":     "production",
				},
				"annotations": mapstr.M{
					"deployment.kubernetes.io/revision": "3",
				},
			},
			"node": mapstr.M{
				"name": "node-1",
			},
			"namespace": "default",
			"container": mapstr.M{
				"name":    "mycontainer",
				"image":   "myrepo/myimage:latest",
				"id":      cacheKey,
				"runtime": "containerd",
			},
		},
	})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Construct a minimal event with only the lookup field — no Clone() overhead
		// counted against the benchmark. Run() will add kubernetes.* and container.*
		// fields to this fresh event.
		event := &beat.Event{
			Fields: mapstr.M{
				"container": mapstr.M{
					"id": cacheKey,
				},
				"message": "some log line",
			},
		}
		_, err := processor.Run(event)
		if err != nil {
			b.Fatal(err)
		}
	}
}
