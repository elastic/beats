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

package multiline

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	c "github.com/elastic/elastic-agent-libs/config"
)

func TestInvalidConfiguration(t *testing.T) {
	testcases := map[string]struct {
		config        map[string]interface{}
		expectedError error
	}{
		"missing multiline pattern": {
			config: map[string]interface{}{
				"match": "before",
			},
			expectedError: ErrMissingPattern,
		},
		"unknown multiline mode": {
			config: map[string]interface{}{
				"type": "no_such_mode",
			},
			expectedError: fmt.Errorf("unknown multiline type: no_such_mode"),
		},
		"missing multiline count": {
			config: map[string]interface{}{
				"type": "count",
			},
			expectedError: ErrMissingCount,
		},
		"missing multiline pattern when while_pattern type is selected": {
			config: map[string]interface{}{
				"type": "while_pattern",
			},
			expectedError: ErrMissingPattern,
		},
	}

	for name, test := range testcases {
		test := test
		t.Run(name, func(t *testing.T) {
			var config Config
			c := c.MustNewConfigFrom(test.config)
			err := c.Unpack(&config)
			require.NotNil(t, err)
			require.Contains(t, err.Error(), test.expectedError.Error())
		})
	}
}

func TestValidConfiguration(t *testing.T) {
	testcases := map[string]struct {
		config map[string]interface{}
	}{
		"correct pattern based multiline": {
			config: map[string]interface{}{
				"type":    "pattern",
				"match":   "before",
				"pattern": "^\n",
			},
		},
		"correct while_pattern based multiline": {
			config: map[string]interface{}{
				"type":    "while_pattern",
				"pattern": "^\n",
			},
		},
		"correct count based multiline": {
			config: map[string]interface{}{
				"type":        "count",
				"count_lines": 5,
			},
		},
	}

	for name, test := range testcases {
		test := test
		t.Run(name, func(t *testing.T) {
			var config Config
			c := c.MustNewConfigFrom(test.config)
			err := c.Unpack(&config)
			require.Nil(t, err)
		})
	}
}
