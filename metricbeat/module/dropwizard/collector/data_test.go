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

package collector

import (
	"testing"

	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/stretchr/testify/assert"
)

func TestSplitTagsFromMetricName(t *testing.T) {
	for _, testCase := range []struct {
		title string
		name  string
		key   string
		tags  mapstr.M
	}{
		{
			title: "no tags",
			name:  "my_metric1",
		}, {
			title: "parameter",
			name:  "metric/{something}/other",
		}, {
			title: "trailing parameter",
			name:  "metric/{notakey}",
		}, {
			title: "standard tags",
			name:  "metric{key1=var1, key2=var2}",
			key:   "metric",
			tags:  mapstr.M{"key1": "var1", "key2": "var2"},
		}, {
			title: "standard tags (no space)",
			name:  "metric{key1=var1,key2=var2}",
			key:   "metric",
			tags:  mapstr.M{"key1": "var1", "key2": "var2"},
		}, {
			title: "empty parameter",
			name:  "metric/{}",
		}, {
			title: "empty key or value",
			name:  "metric{=var1, key2=}",
			key:   "metric",
			tags:  mapstr.M{"": "var1", "key2": ""},
		}, {
			title: "empty key and value",
			name:  "metric{=}",
			key:   "metric",
			tags:  mapstr.M{"": ""},
		}, {
			title: "extra comma",
			name:  "metric{a=b,}",
			key:   "metric",
			tags:  mapstr.M{"a": "b"},
		}, {
			title: "extra comma and space",
			name:  "metric{a=b, }",
			key:   "metric",
			tags:  mapstr.M{"a": "b"},
		}, {
			title: "extra comma and space",
			name:  "metric{,a=b}",
		},
	} {
		t.Run(testCase.title, func(t *testing.T) {
			key, tags := splitTagsFromMetricName(testCase.name)
			if testCase.key == "" && tags == nil {
				assert.Equal(t, testCase.name, key)
			} else {
				assert.Equal(t, testCase.key, key)
				assert.Equal(t, testCase.tags, tags)
			}
		})
	}
}
