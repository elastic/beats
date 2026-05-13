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

import (
	"testing"

	"github.com/elastic/elastic-agent-libs/logp"
)

func TestAlternateSchemeAddress(t *testing.T) {
	tests := []struct {
		name string
		addr string
		want string
	}{
		{"ldap default port", "ldap://dc.example.com:389", "ldaps://dc.example.com:636"},
		{"ldaps default port", "ldaps://dc.example.com:636", "ldap://dc.example.com:389"},
		{"ldap with host only", "ldap://dc.example.com", "ldaps://dc.example.com:636"},
		{"non-standard port ldap", "ldap://dc.example.com:1389", "ldaps://dc.example.com:1389"},
		{"non-standard port ldaps", "ldaps://dc.example.com:1636", "ldap://dc.example.com:1636"},
		{"invalid empty", "", ""},
		{"invalid no host", "ldap://", ""},
		{"wrong scheme", "http://dc.example.com:389", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := alternateSchemeAddress(tt.addr)
			if got != tt.want {
				t.Errorf("alternateSchemeAddress(%q) = %q, want %q", tt.addr, got, tt.want)
			}
		})
	}
}

func TestExpandCandidatesWithAlternateSchemes(t *testing.T) {
	log := logp.NewLogger("discovery_test")
	tests := []struct {
		name string
		in   []string
		want []string
	}{
		{
			name: "single ldap adds ldaps with ldaps first",
			in:   []string{"ldap://dc.example.com:389"},
			want: []string{"ldaps://dc.example.com:636", "ldap://dc.example.com:389"},
		},
		{
			name: "single ldaps adds ldap",
			in:   []string{"ldaps://dc.example.com:636"},
			want: []string{"ldaps://dc.example.com:636", "ldap://dc.example.com:389"},
		},
		{
			name: "both schemes no duplicate, ldaps first",
			in:   []string{"ldap://dc.example.com:389", "ldaps://dc.example.com:636"},
			want: []string{"ldaps://dc.example.com:636", "ldap://dc.example.com:389"},
		},
		{
			name: "two hosts each ldaps first then ldap",
			in:   []string{"ldap://dc1.example.com:389", "ldap://dc2.example.com:389"},
			want: []string{"ldaps://dc1.example.com:636", "ldap://dc1.example.com:389", "ldaps://dc2.example.com:636", "ldap://dc2.example.com:389"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := expandCandidatesWithAlternateSchemes(tt.in, log)
			if len(got) != len(tt.want) {
				t.Errorf("expandCandidatesWithAlternateSchemes() length = %d, want %d; got %v", len(got), len(tt.want), got)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("expandCandidatesWithAlternateSchemes()[%d] = %q, want %q; full got %v", i, got[i], tt.want[i], got)
				}
			}
		})
	}
}
