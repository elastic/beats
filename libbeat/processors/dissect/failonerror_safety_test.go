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

package dissect

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// TestDissectOverwriteKeysSafety verifies that the pre-check for existing keys
// prevents partial writes when OverwriteKeys=false (the default). This proves
// the Clone() skip is safe: the processor checks all keys before writing any.
func TestDissectOverwriteKeysSafety(t *testing.T) {
	c, err := conf.NewConfigFrom(map[string]interface{}{
		"tokenizer":     "hello %{key}",
		"target_prefix": "",
	})
	require.NoError(t, err)

	processor, err := NewProcessor(c, logptest.NewTestingLogger(t, ""))
	require.NoError(t, err)

	input := mapstr.M{
		"message": "hello world",
		"key":     "existing-value",
	}
	event := &beat.Event{Fields: input.Clone()}
	original := input.Clone()

	result, err := processor.Run(event)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot override existing key")

	// Remove error.message and dissect flags added by the processor.
	result.Fields.Delete("error")
	result.Fields.Delete(beat.FlagField)
	assert.Equal(t, original, result.Fields,
		"event fields must be unchanged when key conflict is detected (clone skip safety)")
}
