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

//go:build windows

package wmi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShouldSkipNilOrEmptyValue(t *testing.T) {
	tests := []struct {
		key                string
		fieldValue         interface{}
		includeNull        bool
		includeEmptyString bool
		expectedShouldSkip bool
	}{
		// Test Case 1: fieldValue is nil, and IncludeNull is false
		{
			key:                "Skip nil value with IncludeNull false",
			fieldValue:         nil,
			includeNull:        false,
			includeEmptyString: false,
			expectedShouldSkip: true, // Should skip because IncludeNull is false
		},

		// Test Case 2: fieldValue is an empty string, and IncludeEmptyString is false
		{
			key:                "Skip Empty string with IncludeEmptyString false",
			fieldValue:         "",
			includeNull:        false,
			includeEmptyString: false,
			expectedShouldSkip: true, // Should skip because IncludeEmptyString is false
		},

		// Test Case 3: fieldValue is nil, and IncludeNull is true
		{
			key:                "Don't skip Nil value with IncludeNull true",
			fieldValue:         nil,
			includeNull:        true,
			includeEmptyString: false,
			expectedShouldSkip: false, // Should not skip because IncludeNull is true
		},

		// Test Case 4: fieldValue is a non-empty string, and IncludeEmptyString is false
		{
			key:                "Don't skip Non-empty string with IncludeEmptyString false",
			fieldValue:         "non-empty",
			includeNull:        false,
			includeEmptyString: false,
			expectedShouldSkip: false, // Should not skip because the string is non-empty
		},

		// Test Case 5: fieldValue is a non-empty string, and IncludeEmptyString is true
		{
			key:                "Non-empty string with IncludeEmptyString true",
			fieldValue:         "non-empty",
			includeNull:        true,
			includeEmptyString: true,
			expectedShouldSkip: false, // Should not skip because IncludeEmptyString is true
		},
	}

	for _, test := range tests {
		t.Run(test.key, func(t *testing.T) {

			// Arrange: Create a MetricSet with the configuration based on the test case
			config := Config{
				IncludeNull:        test.includeNull,
				IncludeEmptyString: test.includeEmptyString,
			}

			metricSet := &MetricSet{
				config: config,
			}

			// Act: Call shouldSkipNilOrEmptyValue with the test case fieldValue
			result := metricSet.shouldSkipNilOrEmptyValue(test.fieldValue)

			// Assert: Check if the result matches the expected result
			assert.Equal(t, test.expectedShouldSkip, result)
		})
	}
}
