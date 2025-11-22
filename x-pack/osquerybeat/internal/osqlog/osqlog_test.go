// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package osqlog

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLevel_ToSeverity(t *testing.T) {
	tests := []struct {
		name     string
		level    Level
		expected Severity
	}{
		{
			name:     "Info level",
			level:    LevelInfo,
			expected: SeverityInfo,
		},
		{
			name:     "Warning level",
			level:    LevelWarning,
			expected: SeverityWarning,
		},
		{
			name:     "Error level",
			level:    LevelError,
			expected: SeverityError,
		},
		{
			name:     "Unknown level defaults to Info",
			level:    Level("X"),
			expected: SeverityInfo,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.level.ToSeverity()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSeverityConstants(t *testing.T) {
	// Verify the severity constants match osquery's enum values
	assert.Equal(t, Severity(0), SeverityInfo)
	assert.Equal(t, Severity(1), SeverityWarning)
	assert.Equal(t, Severity(2), SeverityError)
	assert.Equal(t, Severity(3), SeverityFatal)
}
