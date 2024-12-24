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
