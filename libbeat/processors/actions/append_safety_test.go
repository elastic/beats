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

package actions

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// TestAppendSafety verifies that the append processor never leaves the event
// in a partial state when an error occurs. Safety is guaranteed by code
// structure: all reads (GetValue) happen before any writes (Delete + PutValue),
// so an error during the read phase returns before the event is ever modified.
func TestAppendSafety(t *testing.T) {
	tests := []struct {
		description string
		config      appendProcessorConfig
		fields      mapstr.M
		wantErr     bool
		wantTarget  []interface{} // if non-nil, assert target field equals this
	}{
		{
			description: "multiple source fields, middle missing, FailOnError=true",
			config: appendProcessorConfig{
				Fields:      []string{"field_a", "does_not_exist", "field_b"},
				TargetField: "target",
				FailOnError: true,
			},
			fields:  mapstr.M{"field_a": "value_a", "field_b": "value_b"},
			wantErr: true,
		},
		{
			description: "multiple source fields, first missing, FailOnError=true",
			config: appendProcessorConfig{
				Fields:      []string{"missing", "field_b", "field_c"},
				TargetField: "target",
				FailOnError: true,
			},
			fields:  mapstr.M{"field_b": "b", "field_c": "c"},
			wantErr: true,
		},
		{
			description: "multiple source fields, last missing, FailOnError=true",
			config: appendProcessorConfig{
				Fields:      []string{"field_a", "field_b", "missing"},
				TargetField: "target",
				FailOnError: true,
			},
			fields:  mapstr.M{"field_a": "a", "field_b": "b"},
			wantErr: true,
		},
		{
			description: "existing target preserved when source field missing",
			config: appendProcessorConfig{
				Fields:      []string{"field_a", "missing"},
				TargetField: "target",
				FailOnError: true,
			},
			fields:  mapstr.M{"target": []interface{}{"original"}, "field_a": "a"},
			wantErr: true,
		},
		{
			description: "FailOnError=false with missing field returns event unchanged",
			config: appendProcessorConfig{
				Fields:      []string{"field_a", "missing"},
				TargetField: "target",
				FailOnError: false,
			},
			fields:  mapstr.M{"field_a": "a"},
			wantErr: false,
		},
		{
			description: "single source field, successful append",
			config: appendProcessorConfig{
				Fields:      []string{"source"},
				TargetField: "target",
				FailOnError: true,
			},
			fields:     mapstr.M{"source": "hello"},
			wantTarget: []interface{}{"hello"},
		},
		{
			description: "target field exists, Delete+PutValue on same path succeeds",
			config: appendProcessorConfig{
				Fields:      []string{"source"},
				TargetField: "target",
				FailOnError: true,
			},
			fields:     mapstr.M{"target": "existing-scalar", "source": "new-value"},
			wantTarget: []interface{}{"existing-scalar", "new-value"},
		},
		{
			description: "multiple source fields, all present, all values appended",
			config: appendProcessorConfig{
				Fields:      []string{"field_a", "field_b", "field_c"},
				TargetField: "target",
				Values:      []interface{}{"static"},
				FailOnError: true,
			},
			fields:     mapstr.M{"field_a": "a", "field_b": []interface{}{"b1", "b2"}, "field_c": "c"},
			wantTarget: []interface{}{"a", "b1", "b2", "c", "static"},
		},
		{
			description: "IgnoreMissing=true skips missing fields, appends the rest",
			config: appendProcessorConfig{
				Fields:        []string{"field_a", "missing_b", "field_c"},
				TargetField:   "target",
				FailOnError:   true,
				IgnoreMissing: true,
			},
			fields:     mapstr.M{"field_a": "a", "field_c": "c"},
			wantTarget: []interface{}{"a", "c"},
		},
	}

	log := logptest.NewTestingLogger(t, "append_safety_test")

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			event := &beat.Event{Meta: mapstr.M{}, Fields: tt.fields.Clone()}
			original := tt.fields.Clone()

			p := &appendProcessor{
				config: tt.config,
				logger: log.Named("append"),
			}

			got, err := p.Run(event)

			if tt.wantErr {
				require.Error(t, err)

				// Strip error.message (added by FailOnError) before comparing.
				got.Fields.Delete("error")
				assert.Equal(t, original, got.Fields,
					"event fields must be unchanged after error")
				return
			}

			require.NoError(t, err)

			if tt.wantTarget != nil {
				v, getErr := got.GetValue(tt.config.TargetField)
				require.NoError(t, getErr)
				assert.Equal(t, tt.wantTarget, v)
			} else {
				assert.Equal(t, original, got.Fields,
					"event fields must be unchanged")
			}
		})
	}
}
