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

package common

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/elastic-agent-libs/config"
)

func TestConfigJSONBlob(t *testing.T) {
	cases := []struct {
		name        string
		config      map[string]interface{}
		expectedOut []byte
		expectedErr string
	}{
		{
			name: "successfully unpacks string",
			config: map[string]interface{}{
				"jsonBlob": `{"key":"value"}`,
			},
			expectedOut: []byte(`{"key":"value"}`),
		},
		{
			name: "successfully unpacks map[string]interface{}",
			config: map[string]interface{}{
				"jsonBlob": map[string]interface{}{"key": "value"},
			},
			expectedOut: []byte(`{"key":"value"}`),
		},
		{
			name: "successfully unpacks MapStr",
			config: map[string]interface{}{
				"jsonBlob": MapStr{"key": "value"},
			},
			expectedOut: []byte(`{"key":"value"}`),
		},
		{
			name: "fails if can't be converted to json",
			config: map[string]interface{}{
				"jsonBlob": `invalid`,
			},
			expectedErr: "the field can't be converted to valid JSON accessing 'jsonBlob'",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			cfg := config.MustNewConfigFrom(tc.config)
			conf := struct {
				JSONBlob JSONBlob `config:"jsonBlob"`
			}{}
			err := cfg.Unpack(&conf)
			if tc.expectedErr == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tc.expectedErr)
			}
			assert.EqualValues(t, string(tc.expectedOut), string(conf.JSONBlob))
		})
	}
}
