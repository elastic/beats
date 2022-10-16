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
	"reflect"
	"testing"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

var log = logp.NewLogger("append_test")

func Test_cleanEmptyValues(t *testing.T) {
	type args struct {
		dirtyArr []interface{}
	}
	tests := []struct {
		description  string
		args         args
		wantCleanArr []interface{}
	}{
		{
			description: "array with empty values",
			args: args{
				dirtyArr: []interface{}{"asdf", "", 12, "", nil},
			},
			wantCleanArr: []interface{}{"asdf", 12},
		},
		{
			description: "array with no empty values",
			args: args{
				dirtyArr: []interface{}{"asdf", "asd", 12, 123},
			},
			wantCleanArr: []interface{}{"asdf", "asd", 12, 123},
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			if gotCleanArr := cleanEmptyValues(tt.args.dirtyArr); !reflect.DeepEqual(gotCleanArr, tt.wantCleanArr) {
				t.Errorf("cleanEmptyValues() = %v, want %v", gotCleanArr, tt.wantCleanArr)
			}
		})
	}
}

func Test_appendProcessor_appendValues(t *testing.T) {
	type fields struct {
		config appendProcessorConfig
		logger *logp.Logger
	}
	type args struct {
		target string
		fields []string
		values []interface{}
		event  *beat.Event
	}
	tests := []struct {
		description string
		fields      fields
		args        args
		wantErr     bool
	}{
		{
			description: "append value in the arrays from a field when target_field is present",
			args: args{
				target: "target",
				fields: []string{"field"},
				event: &beat.Event{
					Meta: mapstr.M{},
					Fields: mapstr.M{
						"target": []interface{}{"value1", "value2"},
						"field":  "value",
					},
				},
			},
			fields: fields{
				logger: log,
				config: appendProcessorConfig{
					Fields:      []string{"field"},
					TargetField: "target",
				},
			},
			wantErr: false,
		},
		{
			description: "append value in the arrays from a field when target_field is not present",
			args: args{
				target: "target",
				fields: []string{"field"},
				event: &beat.Event{
					Meta: mapstr.M{},
					Fields: mapstr.M{
						"field": "value",
					},
				},
			},
			fields: fields{
				logger: log,
				config: appendProcessorConfig{
					Fields:      []string{"field"},
					TargetField: "target",
				},
			},
			wantErr: false,
		},
		{
			description: "append value in the arrays from an unknown field",
			args: args{
				target: "target",
				fields: []string{"some-field"},
				event: &beat.Event{
					Meta: mapstr.M{},
					Fields: mapstr.M{
						"field": "value",
					},
				},
			},
			fields: fields{
				logger: log,
				config: appendProcessorConfig{
					IgnoreEmptyValues: false,
					IgnoreMissing:     false,
					AllowDuplicate:    true,
					FailOnError:       true,
				},
			},
			wantErr: true,
		},
		{
			description: "append value in the arrays from an unknown field with 'ignore_missing: true'",
			args: args{
				target: "target",
				fields: []string{"some-field"},
				event: &beat.Event{
					Meta: mapstr.M{},
					Fields: mapstr.M{
						"field": "value",
					},
				},
			},
			fields: fields{
				logger: log,
				config: appendProcessorConfig{
					IgnoreEmptyValues: false,
					IgnoreMissing:     true,
					AllowDuplicate:    true,
					FailOnError:       true,
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			f := &appendProcessor{
				config: tt.fields.config,
				logger: tt.fields.logger,
			}
			if err := f.appendValues(tt.args.target, tt.args.fields, tt.args.values, tt.args.event); (err != nil) != tt.wantErr {
				t.Errorf("appendProcessor.appendValues() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_appendProcessor_Run(t *testing.T) {
	type fields struct {
		config appendProcessorConfig
		logger *logp.Logger
	}
	type args struct {
		event *beat.Event
	}
	tests := []struct {
		description string
		fields      fields
		args        args
		want        *beat.Event
		wantErr     bool
	}{
		{
			description: "positive flow",
			fields: fields{
				logger: log,
				config: appendProcessorConfig{
					Fields:            []string{"array-one", "array-two", "concrete-field"},
					TargetField:       "target",
					Values:            []interface{}{"value1", "value2"},
					IgnoreMissing:     false,
					IgnoreEmptyValues: false,
					FailOnError:       true,
					AllowDuplicate:    true,
				},
			},
			args: args{
				event: &beat.Event{
					Meta: mapstr.M{},
					Fields: mapstr.M{
						"concrete-field": "some-value",
						"array-one":      []interface{}{"one", "", "two", "three"},
						"array-two":      []interface{}{"four", "five", ""},
					},
				},
			},
			wantErr: false,
			want: &beat.Event{
				Meta: mapstr.M{},
				Fields: mapstr.M{
					"concrete-field": "some-value",
					"array-one":      []interface{}{"one", "", "two", "three"},
					"array-two":      []interface{}{"four", "five", ""},
					"target":         []interface{}{"one", "", "two", "three", "four", "five", "", "some-value", "value1", "value2"},
				},
			},
		},
		{
			description: "test for nested field",
			fields: fields{
				logger: log,
				config: appendProcessorConfig{
					Fields:            []string{"array.one", "array.two", "concrete-field"},
					TargetField:       "target",
					Values:            []interface{}{"value1", "value2"},
					IgnoreMissing:     false,
					IgnoreEmptyValues: false,
					FailOnError:       true,
					AllowDuplicate:    true,
				},
			},
			args: args{
				event: &beat.Event{
					Meta: mapstr.M{},
					Fields: mapstr.M{
						"concrete-field": "some-value",
						"array": mapstr.M{
							"one": []interface{}{"one", "", "two", "three"},
							"two": []interface{}{"four", "five", ""},
						},
					},
				},
			},
			wantErr: false,
			want: &beat.Event{
				Meta: mapstr.M{},
				Fields: mapstr.M{
					"concrete-field": "some-value",
					"array": mapstr.M{
						"one": []interface{}{"one", "", "two", "three"},
						"two": []interface{}{"four", "five", ""},
					},
					"target": []interface{}{"one", "", "two", "three", "four", "five", "", "some-value", "value1", "value2"},
				},
			},
		},
		{
			description: "remove empty values form output - 'ignore_empty_values: true'",
			fields: fields{
				logger: log,
				config: appendProcessorConfig{
					Fields:            []string{"array-one", "array-two", "concrete-field"},
					TargetField:       "target",
					Values:            []interface{}{"value1", nil, "value2", "", nil},
					IgnoreMissing:     false,
					IgnoreEmptyValues: true,
					FailOnError:       true,
					AllowDuplicate:    true,
				},
			},
			args: args{
				event: &beat.Event{
					Meta: mapstr.M{},
					Fields: mapstr.M{
						"concrete-field": "",
						"array-one":      []interface{}{"one", "", "two", "three"},
						"array-two":      []interface{}{"four", "five", ""},
					},
				},
			},
			wantErr: false,
			want: &beat.Event{
				Meta: mapstr.M{},
				Fields: mapstr.M{
					"concrete-field": "",
					"array-one":      []interface{}{"one", "", "two", "three"},
					"array-two":      []interface{}{"four", "five", ""},
					"target":         []interface{}{"one", "two", "three", "four", "five", "value1", "value2"},
				},
			},
		},
		{
			description: "append value of a missing field with 'ignore_missing: false'",
			fields: fields{
				logger: log,
				config: appendProcessorConfig{
					Fields:            []string{"missing-field"},
					TargetField:       "target",
					IgnoreMissing:     false,
					IgnoreEmptyValues: false,
					FailOnError:       true,
					AllowDuplicate:    true,
				},
			},
			args: args{
				event: &beat.Event{
					Meta:   mapstr.M{},
					Fields: mapstr.M{},
				},
			},
			wantErr: true,
			want: &beat.Event{
				Meta: mapstr.M{},
				Fields: mapstr.M{
					"error": mapstr.M{
						"message": "failed to append fields in append_processor processor: could not fetch value for key: missing-field, Error: key not found",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			f := &appendProcessor{
				config: tt.fields.config,
				logger: tt.fields.logger,
			}
			got, err := f.Run(tt.args.event)
			if (err != nil) != tt.wantErr {
				t.Errorf("appendProcessor.Run() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("appendProcessor.Run() = %v, want %v", got, tt.want)
			}
		})
	}
}
