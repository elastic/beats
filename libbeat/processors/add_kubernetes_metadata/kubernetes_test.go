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

package add_kubernetes_metadata

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
)

// Test metadata updates don't replace existing pod metrics
func TestAnnotatorDeepUpdate(t *testing.T) {
	cfg := common.MustNewConfigFrom(map[string]interface{}{
		"lookup_fields": []string{"kubernetes.pod.name"},
	})
	matcher, err := NewFieldMatcher(*cfg)
	if err != nil {
		t.Fatal(err)
	}

	processor := kubernetesAnnotator{
		cache: newCache(10 * time.Second),
		matchers: &Matchers{
			matchers: []Matcher{matcher},
		},
	}

	processor.cache.set("foo", common.MapStr{
		"pod": common.MapStr{
			"labels": common.MapStr{
				"dont":     "replace",
				"original": "fields",
			},
		},
	})

	event, err := processor.Run(&beat.Event{
		Fields: common.MapStr{
			"kubernetes": common.MapStr{
				"pod": common.MapStr{
					"name": "foo",
					"id":   "pod_id",
					"metrics": common.MapStr{
						"a": 1,
						"b": 2,
					},
				},
			},
		},
	})
	assert.NoError(t, err)

	assert.Equal(t, common.MapStr{
		"kubernetes": common.MapStr{
			"pod": common.MapStr{
				"name": "foo",
				"id":   "pod_id",
				"metrics": common.MapStr{
					"a": 1,
					"b": 2,
				},
				"labels": common.MapStr{
					"dont":     "replace",
					"original": "fields",
				},
			},
		},
	}, event.Fields)
}
