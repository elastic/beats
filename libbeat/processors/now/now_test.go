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

package now

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestNow(t *testing.T) {
	es9ReleaseDate := time.Date(2025, 04, 8, 12, 0, 0, 42, time.UTC)
	currentTime = func() time.Time { return es9ReleaseDate }

	var testCases = []struct {
		description string
		config      nowConfig
		Input       mapstr.M
		Output      mapstr.M
		error       bool
		badConfig   bool
	}{
		{
			description: "Single target field now",
			config: nowConfig{
				Field: "field1",
			},
			Output: mapstr.M{
				"field1": es9ReleaseDate,
			},
			error:     false,
			badConfig: false,
		},
		{
			description: "Target field with now plus existing field",
			config: nowConfig{
				Field: "field1",
			},
			Input: mapstr.M{
				"field2": "some data",
			},
			Output: mapstr.M{
				"field1": es9ReleaseDate,
				"field2": "some data",
			},
			error:     false,
			badConfig: false,
		},
		{
			description: "Target with existing value",
			config: nowConfig{
				Field: "field1",
			},
			Input: mapstr.M{
				"field1": "existing data",
				"field2": "some data",
			},
			Output: mapstr.M{
				"field1": es9ReleaseDate,
				"field2": "some data",
			},
			error:     false,
			badConfig: false,
		},
		{
			description: "Target with dot's (nested field)",
			config: nowConfig{
				Field: "nested.field1",
			},
			Input: mapstr.M{
				"input": "should equal output",
			},
			Output: mapstr.M{
				"nested": mapstr.M{
					"field1": es9ReleaseDate,
				},
				"input": "should equal output",
			},
			error:     false,
			badConfig: false,
		},
		{
			description: "Target with dot's and leaf value along the path, causing error",
			config: nowConfig{
				Field: "nested.field1",
			},
			Input: mapstr.M{
				"nested": "existing 'leaf' data",
				"input":  "should equal output",
			},
			Output: mapstr.M{
				"nested": "existing 'leaf' data",
				"input":  "should equal output",
			},
			error:     true,
			badConfig: false,
		},
		{
			description: "Bad config, no field set",
			config:      nowConfig{},
			error:       true,
			badConfig:   true,
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.description, func(t *testing.T) {
			t.Parallel()

			configInput, err := config.NewConfigFrom(test.config)
			require.NoError(t, err, "Failed to create config from test case")
			testConfig := nowConfig{}
			err = configInput.Unpack(&testConfig)
			if test.badConfig {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			processor := &now{
				log:    logptest.NewTestingLogger(t, "now"),
				config: testConfig,
			}

			inputEvent := &beat.Event{
				Fields: test.Input,
			}

			outputEvent, err := processor.Run(inputEvent)
			if !test.error {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}

			assert.Equal(t, test.Output, outputEvent.Fields, "Output event does not match expected")
		})
	}
}
