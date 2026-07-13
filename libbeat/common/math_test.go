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
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRound(t *testing.T) {
	testCases := []struct {
		name     string
		input    float64
		expected float64
	}{
		{
			name:     "keep exact half",
			input:    0.5,
			expected: 0.5,
		},
		{
			name:     "truncate below midpoint",
			input:    0.50004,
			expected: 0.5,
		},
		{
			name:     "round up at midpoint",
			input:    0.50005,
			expected: 0.5001,
		},
		{
			name:     "keep exact integer plus half",
			input:    1234.5,
			expected: 1234.5,
		},
		{
			name:     "truncate larger number below midpoint",
			input:    1234.50004,
			expected: 1234.5,
		},
		{
			name:     "round up larger number at midpoint",
			input:    1234.50005,
			expected: 1234.5001,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.InDelta(t, tc.expected, Round(tc.input, DefaultDecimalPlacesCount), math.Pow10(-DefaultDecimalPlacesCount), "rounding %v should be %v", tc.input, tc.expected)
		})
	}
}
