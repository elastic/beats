// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/elastic/elastic-agent-libs/logp"
)

func TestResolveOsqueryRuntime_DisabledInstallCleansCustomState(t *testing.T) {
	bundledDir := t.TempDir()
	releasesDir := filepath.Join(bundledDir, "releases", "old-sha")
	if err := os.MkdirAll(releasesDir, 0750); err != nil {
		t.Fatal(err)
	}
	activeReleasePath := filepath.Join(bundledDir, "active_release")
	if err := os.WriteFile(activeReleasePath, []byte(releasesDir), 0640); err != nil {
		t.Fatal(err)
	}

	bt := &osquerybeat{
		log: logp.NewLogger("osquerybeat_runtime_test"),
		executablePath: func() (string, error) {
			return filepath.Join(bundledDir, "osquerybeat"), nil
		},
	}

	resolved, err := bt.resolveOsqueryRuntime(context.Background())
	if err != nil {
		t.Fatalf("resolve runtime failed: %v", err)
	}
	if resolved.Source != "bundled" {
		t.Fatalf("expected bundled source, got %s", resolved.Source)
	}
	if resolved.BinPath != "" {
		t.Fatalf("expected empty custom bin path, got %s", resolved.BinPath)
	}

	if _, err := os.Stat(filepath.Join(bundledDir, "releases")); !os.IsNotExist(err) {
		t.Fatalf("expected releases directory to be removed, got err=%v", err)
	}
	if _, err := os.Stat(activeReleasePath); !os.IsNotExist(err) {
		t.Fatalf("expected active_release to be removed, got err=%v", err)
	}
}
