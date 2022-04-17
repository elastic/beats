// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package osqd

import (
	"bufio"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/menderesk/beats/v7/libbeat/common"
	"github.com/menderesk/beats/v7/libbeat/logp"
	"github.com/menderesk/beats/v7/x-pack/osquerybeat/internal/fileutil"

	"github.com/gofrs/uuid"
	"github.com/google/go-cmp/cmp"
)

func TestNew(t *testing.T) {

	socketPath := "/var/run/foobar"

	extensionsTimeout := 5
	configurationRefreshIntervalSecs := 12
	configPluginName := "config_plugin_test"
	loggerPluginName := "logger_plugin_test"

	osq := New(
		socketPath,
		WithExtensionsTimeout(extensionsTimeout),
		WithConfigRefresh(configurationRefreshIntervalSecs),
		WithConfigPlugin(configPluginName),
		WithLoggerPlugin(loggerPluginName),
	)

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
	dir, err := os.MkdirTemp("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	mandatoryExtensionPath := filepath.Join(dir, extensionName)

	// Write fake extension file for testing
	err = ioutil.WriteFile(mandatoryExtensionPath, nil, 0644)
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
			dir, err := os.MkdirTemp("", "")
			if err != nil {
				t.Fatal(err)
			}

			defer os.RemoveAll(dir)

			extensionAutoloadPath := filepath.Join(dir, osqueryAutoload)

			err = ioutil.WriteFile(extensionAutoloadPath, tc.FileContent, 0644)
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
