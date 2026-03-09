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

//go:build !requirefips

package translate_ldap_attribute

import "testing"

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name        string
		value       string
		expect      string
		expectError bool
	}{
		{name: "empty defaults to auto", value: "", expect: guidTranslationAuto},
		{name: "explicit auto", value: "auto", expect: guidTranslationAuto},
		{name: "explicit always", value: "always", expect: guidTranslationAlways},
		{name: "case insensitive", value: " NEVER  ", expect: guidTranslationNever},
		{name: "invalid", value: "sometimes", expectError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := defaultConfig()
			cfg.ADGUIDTranslation = tt.value
			err := cfg.validate()
			if tt.expectError {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cfg.ADGUIDTranslation != tt.expect {
				t.Fatalf("expected %q, got %q", tt.expect, cfg.ADGUIDTranslation)
			}
		})
	}
}
