// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package osqlog

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestParseGlogLine(t *testing.T) {
	// Fix reference time to make tests deterministic
	now := time.Date(2025, time.March, 14, 16, 0, 0, 0, time.UTC)

	tests := []struct {
		name        string
		line        string
		expectError bool
		expected    *GlogEntry
	}{
		{
			name: "info log",
			line: "I0314 15:24:36.123456 12345 extensions.cpp:123] Extension manager service starting",
			expected: &GlogEntry{
				Level:      LevelInfo,
				ThreadID:   12345,
				SourceFile: "extensions.cpp",
				SourceLine: 123,
				Message:    "Extension manager service starting",
				Timestamp:  time.Date(2025, time.March, 14, 15, 24, 36, 123456000, time.UTC),
			},
		},
		{
			name: "warning log",
			line: "W1225 09:15:42.987654 54321 database.cpp:456] Database file permissions are too open",
			expected: &GlogEntry{
				Level:      LevelWarning,
				ThreadID:   54321,
				SourceFile: "database.cpp",
				SourceLine: 456,
				Message:    "Database file permissions are too open",
				// With now fixed at 2025-03-14 (March), Dec 25 uses current year 2025
				Timestamp: time.Date(2025, time.December, 25, 9, 15, 42, 987654000, time.UTC),
			},
		},
		{
			name: "error log",
			line: "E0101 00:00:00.000001 1 error.cpp:1] Fatal error occurred",
			expected: &GlogEntry{
				Level:      LevelError,
				ThreadID:   1,
				SourceFile: "error.cpp",
				SourceLine: 1,
				Message:    "Fatal error occurred",
				Timestamp:  time.Date(2025, time.January, 1, 0, 0, 0, 1000, time.UTC),
			},
		},
		{
			name: "log with path in filename",
			line: "I0520 12:30:45.555555 99999 src/osquery/extensions.cpp:789] Extension registered successfully",
			expected: &GlogEntry{
				Level:      LevelInfo,
				ThreadID:   99999,
				SourceFile: "src/osquery/extensions.cpp",
				SourceLine: 789,
				Message:    "Extension registered successfully",
				Timestamp:  time.Date(2025, time.May, 20, 12, 30, 45, 555555000, time.UTC),
			},
		},
		{
			name: "log with empty message",
			line: "I0710 08:45:30.111111 7777 test.cpp:999] ",
			expected: &GlogEntry{
				Level:      LevelInfo,
				ThreadID:   7777,
				SourceFile: "test.cpp",
				SourceLine: 999,
				Message:    "",
				Timestamp:  time.Date(2025, time.July, 10, 8, 45, 30, 111111000, time.UTC),
			},
		},
		{
			name:        "invalid format - no prefix",
			line:        "This is not a valid osquery log",
			expectError: true,
		},
		{
			name:        "invalid format - missing bracket",
			line:        "I0314 15:24:36.123456 12345 file.cpp:123 no bracket",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseGlogLineWithNow(tt.line, now)
			if tt.expectError {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if diff := cmp.Diff(tt.expected.Level, result.Level); diff != "" {
				t.Errorf("level mismatch: %s", diff)
			}
			if diff := cmp.Diff(tt.expected.ThreadID, result.ThreadID); diff != "" {
				t.Errorf("thread id mismatch: %s", diff)
			}
			if diff := cmp.Diff(tt.expected.SourceFile, result.SourceFile); diff != "" {
				t.Errorf("source file mismatch: %s", diff)
			}
			if diff := cmp.Diff(tt.expected.SourceLine, result.SourceLine); diff != "" {
				t.Errorf("source line mismatch: %s", diff)
			}
			if diff := cmp.Diff(tt.expected.Message, result.Message); diff != "" {
				t.Errorf("message mismatch: %s", diff)
			}
			if !result.Timestamp.Equal(tt.expected.Timestamp) {
				t.Errorf("timestamp: expected %v, got %v", tt.expected.Timestamp, result.Timestamp)
			}
		})
	}
}

func TestParseGlogLineYearRollover(t *testing.T) {
	// If now is just after New Year, a Dec 31 entry should resolve to previous year.
	now := time.Date(2026, time.January, 1, 0, 0, 5, 0, time.UTC)
	line := "I1231 23:59:59.999999 1 test.cpp:1] Last log of the year"
	result, err := ParseGlogLineWithNow(line, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Timestamp.Year() != 2025 {
		t.Errorf("expected year %d, got %d", 2025, result.Timestamp.Year())
	}
}
