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

package monitoring

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v8/libbeat/common"
)

func TestIsBufferEnabled(t *testing.T) {
	tests := []struct {
		name   string
		input  map[string]interface{}
		expect bool
	}{{
		name: "enabled",
		input: map[string]interface{}{
			"enabled": true,
		},
		expect: true,
	}, {
		name: "disabled",
		input: map[string]interface{}{
			"enabled": false,
		},
		expect: false,
	}, {
		name: "missing",
		input: map[string]interface{}{
			"size": 10,
		},
		expect: false,
	}, {
		name:   "nil",
		input:  nil,
		expect: false,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := common.NewConfigFrom(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expect, IsBufferEnabled(cfg))
		})
	}
}
