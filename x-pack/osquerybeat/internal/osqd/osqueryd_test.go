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

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/fileutil"
	"github.com/elastic/elastic-agent-libs/logp"

	"github.com/gofrs/uuid"
	"github.com/google/go-cmp/cmp"
)

func TestNew(t *testing.T) {

	socketPath := "/var/run/foobar"

	extensionsTimeout := 5
	configurationRefreshIntervalSecs := 12
	configPluginName := "config_plugin_test"
	loggerPluginName := "logger_plugin_test"

	osq, err := New(
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
