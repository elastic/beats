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

package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEquals(t *testing.T) {
	tests := []struct {
		title    string
		first    []string
		second   []string
		expected bool
	}{
		{
			title:    "when we have the same elements, in order",
			first:    []string{"one", "two"},
			second:   []string{"one", "two"},
			expected: true,
		},
		{
			title:    "when we have the same elements, but out of order",
			first:    []string{"one", "two"},
			second:   []string{"two", "one"},
			expected: true,
		},
		{
			title:    "when we have the same elements, with a duplicate",
			first:    []string{"one", "two"},
			second:   []string{"one", "two", "one"},
			expected: true,
		},
		{
			title:    "when we have different number of elements",
			first:    []string{"one", "two"},
			second:   []string{"one", "two", "three"},
			expected: false,
		},
		{
			title:    "when we have different elements",
			first:    []string{"one", "two"},
			second:   []string{"one", "three"},
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.title, func(t *testing.T) {
			assert.Equal(t, test.expected, MakeStringSet(test.first...).Equals(MakeStringSet(test.second...)))
		})
	}
}
