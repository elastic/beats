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

//go:build requirefips

package fipsscan

import (
	"reflect"
	"testing"
)

func TestShortestChain(t *testing.T) {
	graph := map[string][]string{
		"root":   {"a", "b"},
		"a":      {"c", "d"},
		"b":      {"d", "e"},
		"c":      {"target"},
		"d":      {"e"},
		"e":      {},
		"target": {},
	}

	tests := []struct {
		name string
		from string
		to   string
		want []string
	}{
		{
			name: "direct child",
			from: "root",
			to:   "a",
			want: []string{"root", "a"},
		},
		{
			name: "two hops",
			from: "root",
			to:   "c",
			want: []string{"root", "a", "c"},
		},
		{
			name: "shortest of two paths",
			from: "root",
			to:   "target",
			want: []string{"root", "a", "c", "target"},
		},
		{
			name: "same node",
			from: "root",
			to:   "root",
			want: []string{"root"},
		},
		{
			name: "no path",
			from: "root",
			to:   "unreachable",
			want: nil,
		},
		{
			name: "leaf with no edges",
			from: "root",
			to:   "e",
			want: []string{"root", "b", "e"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ShortestChain(tc.from, tc.to, graph)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("ShortestChain(%q, %q) = %v, want %v", tc.from, tc.to, got, tc.want)
			}
		})
	}
}

func TestFormatChain(t *testing.T) {
	tests := []struct {
		name  string
		chain []string
		want  string
	}{
		{
			name:  "empty",
			chain: []string{},
			want:  "",
		},
		{
			name:  "single",
			chain: []string{"pkg/a"},
			want:  "pkg/a",
		},
		{
			name:  "two",
			chain: []string{"pkg/a", "pkg/b"},
			want:  "pkg/a\n      -> pkg/b",
		},
		{
			name:  "three",
			chain: []string{"pkg/a", "pkg/b", "pkg/c"},
			want:  "pkg/a\n      -> pkg/b\n      -> pkg/c",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := FormatChain(tc.chain)
			if got != tc.want {
				t.Errorf("FormatChain(%v) =\n%q\nwant:\n%q", tc.chain, got, tc.want)
			}
		})
	}
}
