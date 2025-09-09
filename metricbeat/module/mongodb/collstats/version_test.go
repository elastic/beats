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

package collstats

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseVersion(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		expected []int
	}{
		{
			name:     "simple version",
			version:  "6.2.0",
			expected: []int{6, 2, 0},
		},
		{
			name:     "version with rc",
			version:  "6.2.0-rc1",
			expected: []int{6, 2, 0},
		},
		{
			name:     "version with build metadata",
			version:  "7.0.1+build123",
			expected: []int{7, 0, 1},
		},
		{
			name:     "major.minor only",
			version:  "5.0",
			expected: []int{5, 0, 0},
		},
		{
			name:     "major only",
			version:  "4",
			expected: []int{4, 0, 0},
		},
		{
			name:     "version with extra parts",
			version:  "6.2.0.1.2",
			expected: []int{6, 2, 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseVersion(tt.version)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsVersionAtLeast(t *testing.T) {
	tests := []struct {
		name     string
		current  string
		target   string
		expected bool
	}{
		{
			name:     "exact match",
			current:  "6.2.0",
			target:   "6.2.0",
			expected: true,
		},
		{
			name:     "current newer major",
			current:  "7.0.0",
			target:   "6.2.0",
			expected: true,
		},
		{
			name:     "current newer minor",
			current:  "6.3.0",
			target:   "6.2.0",
			expected: true,
		},
		{
			name:     "current newer patch",
			current:  "6.2.1",
			target:   "6.2.0",
			expected: true,
		},
		{
			name:     "current older major",
			current:  "5.0.0",
			target:   "6.2.0",
			expected: false,
		},
		{
			name:     "current older minor",
			current:  "6.1.0",
			target:   "6.2.0",
			expected: false,
		},
		{
			name:     "current older patch",
			current:  "6.2.0",
			target:   "6.2.1",
			expected: false,
		},
		{
			name:     "unknown version",
			current:  "unknown",
			target:   "6.2.0",
			expected: false,
		},
		{
			name:     "empty version",
			current:  "",
			target:   "6.2.0",
			expected: false,
		},
		{
			name:     "version with rc suffix",
			current:  "6.2.0-rc1",
			target:   "6.2.0",
			expected: true,
		},
		{
			name:     "version with build metadata",
			current:  "7.0.0+build123",
			target:   "6.2.0",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isVersionAtLeast(tt.current, tt.target)
			assert.Equal(t, tt.expected, result)
		})
	}
}
