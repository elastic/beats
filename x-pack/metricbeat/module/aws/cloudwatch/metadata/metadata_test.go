// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package metadata

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractResourceID(t *testing.T) {
	tests := []struct {
		name            string
		eventIdentifier string
		expected        string
	}{
		{
			name:            "EC2 instance ID",
			eventIdentifier: "123456789012-i-0abcd1234efgh5678-0",
			expected:        "i-0abcd1234efgh5678",
		},
		{
			name:            "RDS with multiple dashes",
			eventIdentifier: "123456789012-my-database-instance-0",
			expected:        "my-database-instance",
		},
		{
			name:            "11-digit prefix is not stripped",
			eventIdentifier: "12345678901-resource-0",
			expected:        "12345678901-resource",
		},
		{
			name:            "Single part returns as is",
			eventIdentifier: "singlevalue",
			expected:        "singlevalue",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractResourceID(tt.eventIdentifier)
			assert.Equal(t, tt.expected, result)
		})
	}
}
