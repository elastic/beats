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

package testing

import (
	"testing"

	"github.com/menderesk/beats/v7/libbeat/common"

	"github.com/stretchr/testify/assert"
)

func TestOmitDocumentedField(t *testing.T) {
	tts := []struct {
		a, b   string
		result bool
	}{
		{a: "hello", b: "world", result: false},
		{a: "hello", b: "hello", result: true},
		{a: "elasticsearch.stats", b: "elasticsearch.stats", result: true},
		{a: "elasticsearch.stats.hello.world", b: "elasticsearch.*", result: true},
		{a: "elasticsearch.stats.hello.world", b: "*", result: true},
	}

	for _, tt := range tts {
		result := omitDocumentedField(tt.a, tt.b)
		assert.Equal(t, tt.result, result)
	}
}

func TestDocumentedFieldCheck(t *testing.T) {
	foundKeys := common.MapStr{
		"hello":               "hello",
		"elasticsearch.stats": "stats1",
	}
	omitfields := []string{
		"hello",
	}
	knownKeys := map[string]interface{}{
		"elasticsearch.stats": "key1",
	}
	err := documentedFieldCheck(foundKeys, knownKeys, omitfields)
	//error should be nil, as `hello` field is ignored and `elasticsearch.stats` field is defined
	assert.NoError(t, err)

	foundKeys = common.MapStr{
		"elasticsearch.stats.cpu":              "stats2",
		"elasticsearch.metrics.requests.count": "requests2",
	}

	knownKeys = map[string]interface{}{
		"elasticsearch.stats.*":     "key1",
		"elasticsearch.metrics.*.*": "hello1",
	}
	// error should be nil as the foundKeys are covered by the `prefix` cases
	err = documentedFieldCheck(foundKeys, knownKeys, omitfields)
	assert.NoError(t, err)

	foundKeys = common.MapStr{
		"elasticsearch.stats.cpu":              "stats2",
		"elasticsearch.metrics.requests.count": "requests2",
	}

	knownKeys = map[string]interface{}{
		"elasticsearch.*":         "key1",
		"elasticsearch.metrics.*": "hello1",
	}
	// error should not be nil as the foundKeys are not covered by the `prefix` cases
	err = documentedFieldCheck(foundKeys, knownKeys, omitfields)
	assert.Error(t, err, "field missing 'elasticsearch.stats.cpu'")

}
