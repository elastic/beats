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
	"github.com/elastic/elastic-agent-libs/logp"
)

func TestResolveOsqueryRuntime_DisabledInstallCleansCustomState(t *testing.T) {
	bundledDir := t.TempDir()
	releasesDir := filepath.Join(bundledDir, "releases", "old-sha")
	if err := os.MkdirAll(releasesDir, 0750); err != nil {
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
	if resolved.BinDir != "" {
		t.Fatalf("expected empty custom bin dir, got %s", resolved.BinDir)
	}

	if _, err := os.Stat(filepath.Join(bundledDir, "releases")); !os.IsNotExist(err) {
		t.Fatalf("expected releases directory to be removed, got err=%v", err)
	}
}

func TestResolveOsqueryRuntime_OtherPlatformConfigFallsBackToBundled(t *testing.T) {
	bundledDir := t.TempDir()
	bt := &osquerybeat{
		log: logp.NewLogger("osquerybeat_runtime_test"),
		executablePath: func() (string, error) {
			return filepath.Join(bundledDir, "osquerybeat"), nil
		},
	}

	platformCfg := &config.InstallArtifactConfig{
		ArtifactURL: "https://example.org/osquery-custom.tar.gz",
		SHA256:      "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	}
	switch runtime.GOOS {
	case "linux":
		bt.osqueryInstallConfig.Windows = platformCfg
	case "darwin":
		bt.osqueryInstallConfig.Linux = platformCfg
	case "windows":
		bt.osqueryInstallConfig.Darwin = platformCfg
	}

	resolved, err := bt.resolveOsqueryRuntime(context.Background())
	if err != nil {
		t.Fatalf("resolve runtime failed: %v", err)
	}
	if resolved.Source != "bundled" {
		t.Fatalf("expected bundled source, got %s", resolved.Source)
	}
	if resolved.BinDir != "" {
		t.Fatalf("expected empty custom bin dir, got %s", resolved.BinDir)
	}
}
