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

package v2

import (
	"testing"

	"github.com/elastic/beats/v8/libbeat/feature"
)

func TestPlugin_Validate(t *testing.T) {
	cases := map[string]struct {
		valid  bool
		plugin Plugin
	}{
		"valid": {
			valid: true,
			plugin: Plugin{
				Name:       "test",
				Stability:  feature.Stable,
				Deprecated: false,
				Info:       "test",
				Doc:        "doc string",
				Manager:    ConfigureWith(nil),
			},
		},
		"missing name": {
			valid: false,
			plugin: Plugin{
				Stability:  feature.Stable,
				Deprecated: false,
				Info:       "test",
				Doc:        "doc string",
				Manager:    ConfigureWith(nil),
			},
		},
		"invalid stability": {
			valid: false,
			plugin: Plugin{
				Name:       "test",
				Deprecated: false,
				Info:       "test",
				Doc:        "doc string",
				Manager:    ConfigureWith(nil),
			},
		},
		"missing manager": {
			valid: false,
			plugin: Plugin{
				Name:       "test",
				Stability:  feature.Stable,
				Deprecated: false,
				Info:       "test",
				Doc:        "doc string",
			},
		},
	}

	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			err := test.plugin.validate()
			if test.valid {
				expectNoError(t, err)
			} else {
				expectError(t, err)
			}
		})
	}
}
