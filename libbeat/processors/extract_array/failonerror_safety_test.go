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

package extract_array

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// TestExtractArrayFailOnErrorSafety verifies that when FailOnError=true and
// the mapping index is out of bounds, the event fields are unchanged
// (proving the clone skip is safe).
func TestExtractArrayFailOnErrorSafety(t *testing.T) {
	processor, err := New(conf.MustNewConfigFrom(mapstr.M{
		"field": "array",
		"mappings": mapstr.M{
			"dest": 999,
		},
	}), logptest.NewTestingLogger(t, ""))
	require.NoError(t, err)

	input := mapstr.M{
		"array": []interface{}{"only-one-element"},
	}
	event := &beat.Event{Fields: input.Clone()}
	original := input.Clone()

	result, err := processor.Run(event)
	require.Error(t, err)

	result.Fields.Delete("error")
	assert.Equal(t, original, result.Fields,
		"event fields must be unchanged after out-of-bounds error (clone skip safety)")
}
