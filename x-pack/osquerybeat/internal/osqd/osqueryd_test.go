// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package osqd

import (
	"bufio"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/fileutil"
	"github.com/elastic/elastic-agent-libs/logp"

	"github.com/gofrs/uuid/v5"
	"github.com/google/go-cmp/cmp"
)

func TestNew(t *testing.T) {

	socketPath := "/var/run/foobar"

	extensionsTimeout := 5
	configurationRefreshIntervalSecs := 12
	configPluginName := "config_plugin_test"
	loggerPluginName := "logger_plugin_test"

	osq, err := newOsqueryD(
		socketPath,
		WithExtensionsTimeout(extensionsTimeout),
		WithConfigRefresh(configurationRefreshIntervalSecs),
		WithConfigPlugin(configPluginName),
		WithLoggerPlugin(loggerPluginName),
	)
	if err != nil {
		t.Fatal(err)
	}

	diff := cmp.Diff(extensionsTimeout, osq.extensionsTimeout)
	if diff != "" {
		t.Error(diff)
	}

	diff = cmp.Diff(configurationRefreshIntervalSecs, osq.configRefreshInterval)
	if diff != "" {
		t.Error(diff)
	}
	diff = cmp.Diff(configPluginName, osq.configPlugin)
	if diff != "" {
		t.Error(diff)
	}

	diff = cmp.Diff(loggerPluginName, osq.loggerPlugin)
	if diff != "" {
		t.Error(diff)
	}
}

func TestVerifyAutoloadFileMissing(t *testing.T) {
	dir := uuid.Must(uuid.NewV4()).String()
	extensionAutoloadPath := filepath.Join(dir, osqueryAutoload)
	mandatoryExtensionPath := filepath.Join(dir, extensionName)
	err := verifyAutoloadFile(extensionAutoloadPath, mandatoryExtensionPath)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected error: %v, got: %v", os.ErrNotExist, err)
	}
}

// TestPrepareAutoloadFile tests possibly different states of the osquery.autoload file and that it is restored into the workable state
func TestPrepareAutoloadFile(t *testing.T) {
	validLogger := logp.NewLogger("osqueryd_test")

	// Prepare the directory with extension
	dir := t.TempDir()
	mandatoryExtensionPath := filepath.Join(dir, extensionName)

	// Write fake extension file for testing
	err := os.WriteFile(mandatoryExtensionPath, nil, 0600)
	if err != nil {
		t.Fatal(err)
	}

	randomContent := func(sz int) []byte {
		b, err := common.RandomBytes(sz)
		if err != nil {
			t.Fatal(err)
		}
		return b
	}

	tests := []struct {
		Name        string
		FileContent []byte
	}{
		{
			Name:        "Empty file",
			FileContent: nil,
		},
		{
			Name:        "File with mandatory extension",
			FileContent: []byte(mandatoryExtensionPath),
		},
		{
			Name:        "Missing mandatory extension, should restore the file",
			FileContent: []byte(filepath.Join(dir, "foobar.ext")),
		},
		{
			Name:        "User extension path doesn't exists",
			FileContent: []byte(mandatoryExtensionPath + "\n" + filepath.Join(dir, "foobar.ext")),
		},
		{
			Name:        "Random garbage",
			FileContent: randomContent(1234),
		},
	}

	for _, tc := range tests {
		t.Run(tc.Name, func(t *testing.T) {

			// Setup
			extensionAutoloadPath := filepath.Join(t.TempDir(), osqueryAutoload)

			err = os.WriteFile(extensionAutoloadPath, tc.FileContent, 0600)
			if err != nil {
				t.Fatal(err)
			}

			err = prepareAutoloadFile(extensionAutoloadPath, mandatoryExtensionPath, validLogger)
			if err != nil {
				t.Fatal(err)
			}

			// Check the content, should have our mandatory extension and possibly the other extension paths with each extension existing on the disk
			f, err := os.Open(extensionAutoloadPath)
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close()
			scanner := bufio.NewScanner(f)
			for i := 0; scanner.Scan(); i++ {
				line := scanner.Text()
				if i == 0 {
					if line != mandatoryExtensionPath {
						t.Fatalf("expected the fist line of the file to be: %v , got: %v", mandatoryExtensionPath, line)
					}
				}
				// Check that it is a valid path to the file on the disk
				ok, err := fileutil.FileExists(line)
				if err != nil {
					t.Fatal(err)
				}
				if !ok {
					t.Fatalf("expected to have only valid paths to the extensions files that exists, got: %v", line)
				}
			}

			err = scanner.Err()
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestGetEnabledDisabledTables(t *testing.T) {

	tests := []struct {
		name             string
		flags            Flags
		expectedEnabled  []string
		expectedDisabled []string
	}{
		{
			name:             "default",
			expectedEnabled:  []string{},
			expectedDisabled: defaultDisabledTables,
		},
		{
			name: "enable all",
			flags: map[string]interface{}{
				"enable_tables": "curl,carves",
			},
			expectedEnabled:  []string{},
			expectedDisabled: []string{},
		},
		{
			name: "enable curl",
			flags: map[string]interface{}{
				"enable_tables": "curl",
			},
			expectedEnabled:  []string{},
			expectedDisabled: []string{"carves"},
		},
		{
			name: "enable curl and carves",
			flags: map[string]interface{}{
				"enable_tables": "curl, carves",
			},
			expectedEnabled:  []string{},
			expectedDisabled: []string{},
		},
		{
			name: "enable os_info",
			flags: map[string]interface{}{
				"enable_tables": "os_info",
			},
			expectedEnabled:  []string{"os_info"},
			expectedDisabled: defaultDisabledTables,
		},
		{
			name: "disable os_info",
			flags: map[string]interface{}{
				"disable_tables": "os_info",
			},
			expectedEnabled:  []string{},
			expectedDisabled: append(defaultDisabledTables, "os_info"),
		},
		{
			name: "disable curl os_info",
			flags: map[string]interface{}{
				"disable_tables": "curl, os_info",
			},
			expectedEnabled:  []string{},
			expectedDisabled: append(defaultDisabledTables, "os_info"),
		},
		{
			name: "disable curl os_info, enable os_info",
			flags: map[string]interface{}{
				"disable_tables": "curl, os_info",
				"enable_tables":  "os_info",
			},
			expectedEnabled:  []string{},
			expectedDisabled: defaultDisabledTables,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			enabled, disabled := getEnabledDisabledTables(tc.flags)
			diff := cmp.Diff(tc.expectedEnabled, enabled)
			if diff != "" {
				t.Error(diff)
			}
			diff = cmp.Diff(tc.expectedDisabled, disabled)
			if diff != "" {
				t.Error(diff)
			}
		})
	}
}

func TestParseOsqueryLog(t *testing.T) {
	currentYear := 2025

	tests := []struct {
		name        string
		line        string
		expectError bool
		expected    *osqueryLogEntry
	}{
		{
			name: "info log",
			line: "I0314 15:24:36.123456 12345 extensions.cpp:123] Extension manager service starting",
			expected: &osqueryLogEntry{
				level:      "I",
				threadID:   12345,
				sourceFile: "extensions.cpp",
				sourceLine: 123,
				message:    "Extension manager service starting",
				timestamp:  time.Date(2025, time.March, 14, 15, 24, 36, 123456000, time.UTC),
			},
		},
		{
			name: "warning log",
			line: "W1225 09:15:42.987654 54321 database.cpp:456] Database file permissions are too open",
			expected: &osqueryLogEntry{
				level:      "W",
				threadID:   54321,
				sourceFile: "database.cpp",
				sourceLine: 456,
				message:    "Database file permissions are too open",
				timestamp:  time.Date(2025, time.December, 25, 9, 15, 42, 987654000, time.UTC),
			},
		},
		{
			name: "error log",
			line: "E0101 00:00:00.000001 1 error.cpp:1] Fatal error occurred",
			expected: &osqueryLogEntry{
				level:      "E",
				threadID:   1,
				sourceFile: "error.cpp",
				sourceLine: 1,
				message:    "Fatal error occurred",
				timestamp:  time.Date(2025, time.January, 1, 0, 0, 0, 1000, time.UTC),
			},
		},
		{
			name: "log with path in filename",
			line: "I0520 12:30:45.555555 99999 src/osquery/extensions.cpp:789] Extension registered successfully",
			expected: &osqueryLogEntry{
				level:      "I",
				threadID:   99999,
				sourceFile: "src/osquery/extensions.cpp",
				sourceLine: 789,
				message:    "Extension registered successfully",
				timestamp:  time.Date(2025, time.May, 20, 12, 30, 45, 555555000, time.UTC),
			},
		},
		{
			name: "log with empty message",
			line: "I0710 08:45:30.111111 7777 test.cpp:999] ",
			expected: &osqueryLogEntry{
				level:      "I",
				threadID:   7777,
				sourceFile: "test.cpp",
				sourceLine: 999,
				message:    "",
				timestamp:  time.Date(2025, time.July, 10, 8, 45, 30, 111111000, time.UTC),
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
			result, err := parseOsqueryLog(tt.line, currentYear)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result.level != tt.expected.level {
				t.Errorf("level: expected %q, got %q", tt.expected.level, result.level)
			}

			if result.threadID != tt.expected.threadID {
				t.Errorf("threadID: expected %d, got %d", tt.expected.threadID, result.threadID)
			}

			if result.sourceFile != tt.expected.sourceFile {
				t.Errorf("sourceFile: expected %q, got %q", tt.expected.sourceFile, result.sourceFile)
			}

			if result.sourceLine != tt.expected.sourceLine {
				t.Errorf("sourceLine: expected %d, got %d", tt.expected.sourceLine, result.sourceLine)
			}

			if result.message != tt.expected.message {
				t.Errorf("message: expected %q, got %q", tt.expected.message, result.message)
			}

			if !result.timestamp.Equal(tt.expected.timestamp) {
				t.Errorf("timestamp: expected %v, got %v", tt.expected.timestamp, result.timestamp)
			}
		})
	}
}

func TestParseOsqueryLogYearRollover(t *testing.T) {
	// Test that parsing works correctly when the year changes
	// If we're in January and see a December log, it should be from the previous year

	// This is a simplified test - in production, you might want to handle year rollover
	// by checking if the parsed date is in the future and adjusting accordingly
	currentYear := 2025

	line := "I1231 23:59:59.999999 1 test.cpp:1] Last log of the year"
	result, err := parseOsqueryLog(line, currentYear)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.timestamp.Year() != currentYear {
		t.Errorf("expected year %d, got %d", currentYear, result.timestamp.Year())
	}
}
