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

package add_remote_metadata

import (
	"errors"
	"testing"
)

var validateTests = []struct {
	name string
	cfg  config
	want error
}{
	{
		name: "invalid_provider",
		cfg:  config{Provider: "invalid_provider"},
		want: errors.New("unknown provider: invalid_provider"),
	},
	{
		name: "missing_fields_and_target",
		cfg:  config{Provider: "map"},
		want: errors.New("no target field and no source fields specified"),
	},
	{
		name: "missing_fields",
		cfg:  config{Provider: "map", Fields: []string{"src"}},
		want: nil,
	},
	{
		name: "missing_target",
		cfg:  config{Provider: "map", Target: "dst"},
		want: nil,
	},
}

func TestValidate(t *testing.T) {
	for _, test := range validateTests {
		t.Run(test.name, func(t *testing.T) {
			got := test.cfg.Validate()
			if !sameError(got, test.want) {
				t.Errorf("unexpected error: got:%v want:%v", got, test.want)
			}
		})
	}
}

var getMappingsTests = []struct {
	name string
	cfg  config
	want error
}{
	{
		name: "valid",
		cfg:  config{Provider: "map", Fields: []string{"src.field"}},
		want: nil,
	},
	{
		name: "type_conflict",
		// Note that the order of fields here matters; it probably
		// should not, but this is current mapstr.M.Put behaviour.
		cfg:  config{Provider: "map", Fields: []string{"src", "src.field"}},
		want: errors.New("failed to set mapping 'src.field' -> 'src.field': expected map but type is string"),
	},
	{
		name: "repeated_field",
		cfg:  config{Provider: "map", Fields: []string{"src.field", "src.field"}},
		want: errors.New("field 'src.field' repeated"),
	},
}

func TestGetMappings(t *testing.T) {
	for _, test := range getMappingsTests {
		t.Run(test.name, func(t *testing.T) {
			_, got := test.cfg.getMappings()
			if !sameError(got, test.want) {
				t.Errorf("unexpected error: got:%v want:%v", got, test.want)
			}
		})
	}
}

func sameError(a, b error) bool {
	switch {
	case a == nil && b == nil:
		return true
	case a == nil, b == nil:
		return false
	default:
		return a.Error() == b.Error()
	}
}
