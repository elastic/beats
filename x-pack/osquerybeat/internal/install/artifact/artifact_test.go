// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package artifact

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

func TestCleanupOldReleases(t *testing.T) {
	installDir := t.TempDir()
	releasesDir := filepath.Join(installDir, releasesDirName)
	if err := os.MkdirAll(releasesDir, 0750); err != nil {
		t.Fatal(err)
	}

	current := filepath.Join(releasesDir, "new")
	old := filepath.Join(releasesDir, "old")
	if err := os.MkdirAll(current, 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(old, 0750); err != nil {
		t.Fatal(err)
	}

	if err := cleanupOldReleases(installDir, current); err != nil {
		t.Fatalf("cleanup failed: %v", err)
	}

	if _, err := os.Stat(current); err != nil {
		t.Fatalf("current release should remain: %v", err)
	}
	if _, err := os.Stat(old); !os.IsNotExist(err) {
		t.Fatalf("old release should be removed, stat err: %v", err)
	}
}

func TestCleanupOldReleasesMissingDirectory(t *testing.T) {
	installDir := t.TempDir()
	current := filepath.Join(installDir, releasesDirName, "new")
	if err := cleanupOldReleases(installDir, current); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRemoveInstalled(t *testing.T) {
	installDir := t.TempDir()
	releasesDir := filepath.Join(installDir, releasesDirName)
	if err := os.MkdirAll(filepath.Join(releasesDir, "old"), 0750); err != nil {
		t.Fatal(err)
	}

	if err := RemoveInstalled(installDir); err != nil {
		t.Fatalf("remove installed failed: %v", err)
	}

	if _, err := os.Stat(releasesDir); !os.IsNotExist(err) {
		t.Fatalf("releases dir should be removed, got err: %v", err)
	}
}

func TestEnsureChecksumMismatch(t *testing.T) {
	log := logp.NewLogger("artifact_test")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("artifact bytes"))
	}))
	defer server.Close()

	cfg := config.InstallConfig{
		ArtifactURL:      server.URL + "/osquery.tar.gz",
		SHA256:           strings.Repeat("a", 64),
		AllowInsecureURL: true,
	}

	_, err := Ensure(context.Background(), cfg, t.TempDir(), log)
	if err == nil {
		t.Fatal("expected checksum mismatch error")
	}
	if !strings.Contains(err.Error(), "sha256 mismatch") {
		t.Fatalf("expected checksum mismatch error, got: %v", err)
	}
}

func TestExtractArtifactUnsupportedFormat(t *testing.T) {
	err := extractArtifact("/tmp/artifact.bin", "https://example.org/osquery.bin", t.TempDir())
	if err == nil {
		t.Fatal("expected unsupported format error")
	}
	if !strings.Contains(err.Error(), "unsupported artifact format") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLocateBinDirMissingBinary(t *testing.T) {
	root := t.TempDir()
	err := os.WriteFile(filepath.Join(root, "README.txt"), []byte("no binary"), 0600)
	if err != nil {
		t.Fatal(err)
	}

	_, err = locateBinDir(root, "linux")
	if err == nil {
		t.Fatal("expected locateBinDir error")
	}
	if !strings.Contains(err.Error(), "failed to locate osquery binary") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLocateBinDirDarwin(t *testing.T) {
	t.Run("direct app structure", func(t *testing.T) {
		root := t.TempDir()
		binFile := filepath.Join(root, "osquery.app", "Contents", "MacOS", "osqueryd")
		if err := os.MkdirAll(filepath.Dir(binFile), 0750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(binFile, []byte("#!/bin/sh\n"), 0755); err != nil {
			t.Fatal(err)
		}

		got, err := locateBinDir(root, "darwin")
		if err != nil {
			t.Fatalf("locateBinDir failed: %v", err)
		}
		if got != root {
			t.Fatalf("expected root dir %s, got %s", root, got)
		}
	})

	t.Run("prefixed app structure", func(t *testing.T) {
		root := t.TempDir()
		expectedDir := filepath.Join(root, "prefix")
		binFile := filepath.Join(expectedDir, "osquery.app", "Contents", "MacOS", "osqueryd")
		if err := os.MkdirAll(filepath.Dir(binFile), 0750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(binFile, []byte("#!/bin/sh\n"), 0755); err != nil {
			t.Fatal(err)
		}

		got, err := locateBinDir(root, "darwin")
		if err != nil {
			t.Fatalf("locateBinDir failed: %v", err)
		}
		if got != expectedDir {
			t.Fatalf("expected prefixed dir %s, got %s", expectedDir, got)
		}
	})
}

func TestEnsureReuseInstalledByChecksum(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test uses shell-script osqueryd fixture")
	}

	log := logp.NewLogger("artifact_test")
	version := "5.19.0"
	artifactBytes := buildTarGzArtifact(t, version)
	sum := sha256.Sum256(artifactBytes)
	sha := hex.EncodeToString(sum[:])

	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		_, _ = w.Write(artifactBytes)
	}))
	defer server.Close()

	cfg := config.InstallConfig{
		ArtifactURL:      server.URL + "/osquery.tar.gz",
		SHA256:           sha,
		AllowInsecureURL: true,
	}

	installDir := t.TempDir()
	first, err := Ensure(context.Background(), cfg, installDir, log)
	if err != nil {
		t.Fatalf("first ensure failed: %v", err)
	}
	second, err := Ensure(context.Background(), cfg, installDir, log)
	if err != nil {
		t.Fatalf("second ensure failed: %v", err)
	}

	if requests != 1 {
		t.Fatalf("expected one download request, got %d", requests)
	}
	if first.BinDir != second.BinDir {
		t.Fatalf("expected bin dir reuse, first=%s second=%s", first.BinDir, second.BinDir)
	}
	if first.Version != second.Version || first.Version != version {
		t.Fatalf("unexpected version reuse, first=%s second=%s", first.Version, second.Version)
	}
}

func TestEnsureChecksumUpdateCleansOldRelease(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test uses shell-script osqueryd fixture")
	}

	log := logp.NewLogger("artifact_test")
	artifactV1 := buildTarGzArtifact(t, "5.19.0")
	artifactV2 := buildTarGzArtifact(t, "5.20.0")

	sum1 := sha256.Sum256(artifactV1)
	sha1 := hex.EncodeToString(sum1[:])
	sum2 := sha256.Sum256(artifactV2)
	sha2 := hex.EncodeToString(sum2[:])

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "v1") {
			_, _ = w.Write(artifactV1)
			return
		}
		_, _ = w.Write(artifactV2)
	}))
	defer server.Close()

	installDir := t.TempDir()
	cfg1 := config.InstallConfig{
		ArtifactURL:      server.URL + "/v1/osquery.tar.gz",
		SHA256:           sha1,
		AllowInsecureURL: true,
	}
	cfg2 := config.InstallConfig{
		ArtifactURL:      server.URL + "/v2/osquery.tar.gz",
		SHA256:           sha2,
		AllowInsecureURL: true,
	}

	if _, err := Ensure(context.Background(), cfg1, installDir, log); err != nil {
		t.Fatalf("first ensure failed: %v", err)
	}
	res2, err := Ensure(context.Background(), cfg2, installDir, log)
	if err != nil {
		t.Fatalf("second ensure failed: %v", err)
	}

	releasesDir := filepath.Join(installDir, releasesDirName)
	entries, err := os.ReadDir(releasesDir)
	if err != nil {
		t.Fatalf("failed reading releases dir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected exactly one active release dir, got %d", len(entries))
	}
	if entries[0].Name() != sha2 {
		t.Fatalf("expected active release dir %s, got %s", sha2, entries[0].Name())
	}
	if strings.Contains(res2.BinDir, sha1) {
		t.Fatalf("new bin dir should not point to old release: %s", res2.BinDir)
	}
}

func buildTarGzArtifact(t *testing.T, version string) []byte {
	t.Helper()

	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)

	binPath := "osqueryd"
	script := "#!/bin/sh\necho \"osqueryd version " + version + "\"\n"
	hdr := &tar.Header{
		Name: binPath,
		Mode: 0755,
		Size: int64(len(script)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatalf("write tar header failed: %v", err)
	}
	if _, err := tw.Write([]byte(script)); err != nil {
		t.Fatalf("write tar body failed: %v", err)
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("close tar writer failed: %v", err)
	}
	if err := gzw.Close(); err != nil {
		t.Fatalf("close gzip writer failed: %v", err)
	}
	return buf.Bytes()
}
