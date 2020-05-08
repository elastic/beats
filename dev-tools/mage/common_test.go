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

package mage

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseVersion(t *testing.T) {
	var tests = []struct {
		Version             string
		Major, Minor, Patch int
	}{
		{"v1.2.3", 1, 2, 3},
		{"1.2.3", 1, 2, 3},
		{"1.2.3-SNAPSHOT", 1, 2, 3},
		{"1.2.3rc1", 1, 2, 3},
		{"1.2", 1, 2, 0},
	}

	for _, tc := range tests {
		major, minor, patch, err := ParseVersion(tc.Version)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, tc.Major, major)
		assert.Equal(t, tc.Minor, minor)
		assert.Equal(t, tc.Patch, patch)
	}
}
