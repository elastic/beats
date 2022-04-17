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

package checks

import (
	"testing"

	"github.com/menderesk/beats/v7/libbeat/beat"
	"github.com/menderesk/beats/v7/libbeat/common"
	"github.com/menderesk/beats/v7/libbeat/processors"
)

type mockProcessor struct{}

func newMock(c *common.Config) (processors.Processor, error) {
	return &mockProcessor{}, nil
}

func (m *mockProcessor) Run(event *beat.Event) (*beat.Event, error) {
	return event, nil
}

func (m *mockProcessor) String() string {
	return "mockProcessor"
}

func TestRequiredFields(t *testing.T) {
	tests := map[string]struct {
		Config   map[string]interface{}
		Required []string
		Valid    bool
	}{
		"one required field present in the configuration": {
			Config: map[string]interface{}{
				"required_field": nil,
				"not_required":   nil,
			},
			Required: []string{
				"required_field",
			},
			Valid: true,
		},
		"two required field present in the configuration": {
			Config: map[string]interface{}{
				"required_field":         nil,
				"another_required_field": nil,
				"not_required":           nil,
			},
			Required: []string{
				"required_field",
				"another_required_field",
			},
			Valid: true,
		},
		"one required field present and one missing in the configuration": {
			Config: map[string]interface{}{
				"required_field": nil,
				"not_required":   nil,
			},
			Required: []string{
				"required_field",
				"one_more_required_field",
			},
			Valid: false,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			runTest(t, RequireFields, test.Config, test.Required, test.Valid)
		})
	}
}

func TestAllowedFields(t *testing.T) {
	tests := map[string]struct {
		Config  map[string]interface{}
		Allowed []string
		Valid   bool
	}{
		"one allowed field present in the configuration": {
			Config: map[string]interface{}{
				"allowed_field": nil,
			},
			Allowed: []string{
				"allowed_field",
			},
			Valid: true,
		},
		"two allowed field present in the configuration": {
			Config: map[string]interface{}{
				"allowed_field":         nil,
				"another_allowed_field": nil,
			},
			Allowed: []string{
				"allowed_field",
				"another_allowed_field",
			},
			Valid: true,
		},
		"one allowed field present and one not allowed is present in the configuration": {
			Config: map[string]interface{}{
				"allowed_field": nil,
				"not_allowed":   nil,
			},
			Allowed: []string{
				"allowed_field",
			},
			Valid: false,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			runTest(t, AllowedFields, test.Config, test.Allowed, test.Valid)
		})
	}
}

func TestMutuallyExclusiveRequiredFields(t *testing.T) {
	tests := map[string]struct {
		Config            map[string]interface{}
		MutuallyExclusive []string
		Valid             bool
	}{
		"one mutually exclusive field is present in the configuration": {
			Config: map[string]interface{}{
				"first_option": nil,
			},
			MutuallyExclusive: []string{
				"first_option",
				"second_option",
			},
			Valid: true,
		},
		"two mutually exclusive field is present in the configuration": {
			Config: map[string]interface{}{
				"first_option":  nil,
				"second_option": nil,
			},
			MutuallyExclusive: []string{
				"first_option",
				"second_option",
			},
			Valid: false,
		},
		"no mutually exclusive field is present in the configuration": {
			Config: map[string]interface{}{
				"third_option": nil,
			},
			MutuallyExclusive: []string{
				"first_option",
				"second_option",
			},
			Valid: false,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			runTest(t, MutuallyExclusiveRequiredFields, test.Config, test.MutuallyExclusive, test.Valid)
		})
	}
}

func runTest(
	t *testing.T,
	check func(fields ...string) func(*common.Config) error,
	config map[string]interface{},
	fields []string,
	valid bool,
) {
	cfg, err := common.NewConfigFrom(config)
	if err != nil {
		t.Fatalf("Unexpected error while creating configuration: %+v\n", err)
	}
	factory := ConfigChecked(newMock, check(fields...))
	_, err = factory(cfg)

	if err != nil && valid {
		t.Errorf("Unexpected error when validating configuration of processor: %+v\n", err)
	}

	if err == nil && !valid {
		t.Errorf("Expected error but nothing was reported\n")
	}
}
