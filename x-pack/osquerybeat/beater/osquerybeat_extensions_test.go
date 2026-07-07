// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/config"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/osqd"
	"github.com/elastic/elastic-agent-libs/logp"
)

func TestExtensionsDiagnosticsPayload(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission-based pre-check differs on windows")
	}

	dataDir := t.TempDir()

	// Extension folder containing one valid (executable) extension and one
	// non-executable (skipped) extension binary.
	extDir := filepath.Join(dataDir, "extensions")
	if err := os.Mkdir(extDir, 0755); err != nil {
		t.Fatal(err)
	}
	validExt := filepath.Join(extDir, "valid.ext")
	if err := os.WriteFile(validExt, nil, 0700); err != nil {
		t.Fatal(err)
	}
	skippedExt := filepath.Join(extDir, "skipped.ext")
	if err := os.WriteFile(skippedExt, nil, 0600); err != nil {
		t.Fatal(err)
	}
	// A configured folder that does not exist should be reported as an error.
	missingDir := filepath.Join(dataDir, "missing")

	// Simulate an autoload file written by prepare().
	autoloadPath := osqd.AutoloadPath(dataDir)
	mandatory := filepath.Join(dataDir, "osquery-extension.ext")
	if err := os.WriteFile(autoloadPath, []byte(mandatory+"\n"+validExt), 0600); err != nil {
		t.Fatal(err)
	}

	loaded := []map[string]interface{}{
		{"name": "valid_ext", "version": "1.0", "path": validExt},
	}

	bt := &osquerybeat{log: logp.NewLogger("ext_diag_test")}
	bt.setExtensionsDiagnostics(config.ExtensionsConfig{
		Paths:   []string{extDir, missingDir},
		Timeout: 20,
	}, dataDir)
	bt.setDiagnosticsQueryExecutor(&mockExecutor{result: loaded})

	payload := bt.extensionsDiagnosticsPayload(context.Background())

	if got := payload["configured_entries_count"]; got != 2 {
		t.Fatalf("expected 2 configured entries, got %v", got)
	}
	if got := payload["discovered_extensions_count"]; got != 1 {
		t.Fatalf("expected 1 discovered extension, got %v", got)
	}
	if got := payload["extensions_timeout"]; got != 20 {
		t.Fatalf("expected extensions_timeout 20, got %v", got)
	}

	entries, ok := payload["configured_entries"].([]map[string]interface{})
	if !ok || len(entries) != 2 {
		t.Fatalf("unexpected configured_entries: %v", payload["configured_entries"])
	}
	if entries[0]["status"] != "ok" {
		t.Fatalf("expected first entry ok, got %v", entries[0])
	}
	if loadedPaths, ok := entries[0]["loaded"].([]string); !ok || len(loadedPaths) != 1 || loadedPaths[0] != validExt {
		t.Fatalf("expected %q loaded, got %v", validExt, entries[0]["loaded"])
	}
	if skipped, ok := entries[0]["skipped"].([]map[string]interface{}); !ok || len(skipped) != 1 || skipped[0]["path"] != skippedExt {
		t.Fatalf("expected %q skipped, got %v", skippedExt, entries[0]["skipped"])
	}
	if entries[1]["status"] != "error" || entries[1]["reason"] == nil {
		t.Fatalf("expected second entry error with reason, got %v", entries[1])
	}

	autoload, ok := payload["autoload_entries"].([]string)
	if !ok || len(autoload) != 2 || autoload[0] != mandatory || autoload[1] != validExt {
		t.Fatalf("unexpected autoload_entries: %v", payload["autoload_entries"])
	}

	if _, ok := payload["loaded_extensions"]; !ok {
		t.Fatalf("expected loaded_extensions in payload, got %v", payload)
	}
	if _, ok := payload["unsupported_notice"]; !ok {
		t.Fatalf("expected unsupported_notice in payload")
	}
}
