// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package osqd

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

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

	// Run prepareAutoloadFile and verify the file is created and valid
	extensionAutoloadPath := filepath.Join(dir, osqueryAutoload)
	if err := prepareAutoloadFile(extensionAutoloadPath, mandatoryExtensionPath, validLogger); err != nil {
		t.Fatalf("prepareAutoloadFile failed: %v", err)
	}
	if err := verifyAutoloadFile(extensionAutoloadPath, mandatoryExtensionPath); err != nil {
		t.Fatalf("verifyAutoloadFile failed: %v", err)
	}

	// Idempotency: running it again should still validate
	if err := prepareAutoloadFile(extensionAutoloadPath, mandatoryExtensionPath, validLogger); err != nil {
		t.Fatalf("prepareAutoloadFile second run failed: %v", err)
	}
	if err := verifyAutoloadFile(extensionAutoloadPath, mandatoryExtensionPath); err != nil {
		t.Fatalf("verifyAutoloadFile second run failed: %v", err)
	}
}
