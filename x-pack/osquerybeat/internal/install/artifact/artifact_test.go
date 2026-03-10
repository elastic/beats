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
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

func testInstallConfig(url, sha string) config.InstallConfig {
	cfg := config.InstallConfig{
		AllowInsecureURL: true,
	}
	platformCfg := &config.InstallPlatformConfig{
		AMD64: &config.InstallArtifactConfig{
			ArtifactURL: url,
			SHA256:      sha,
		},
		ARM64: &config.InstallArtifactConfig{
			ArtifactURL: url,
			SHA256:      sha,
		},
	}
	switch runtime.GOOS {
	case "linux":
		cfg.Linux = platformCfg
	case "darwin":
		cfg.Darwin = platformCfg
	case "windows":
		cfg.Windows = platformCfg
	}
	return cfg
}

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

	cfg := testInstallConfig(server.URL+"/osquery.tar.gz", strings.Repeat("a", 64))

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

func TestExtractArtifactFromSignedURLPath(t *testing.T) {
	artifactBytes := buildTarGzArtifact(t, "5.19.0", true)
	artifactFile := filepath.Join(t.TempDir(), "artifact.tar.gz")
	if err := os.WriteFile(artifactFile, artifactBytes, 0600); err != nil {
		t.Fatalf("write artifact failed: %v", err)
	}

	destDir := t.TempDir()
	err := extractArtifact(artifactFile, "https://example.org/osquery.tar.gz?X-Amz-Signature=test", destDir)
	if err != nil {
		t.Fatalf("expected extraction success from signed URL, got: %v", err)
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

func TestLocateBinDirIgnoresSymlinkEscapingRoot(t *testing.T) {
	root := t.TempDir()

	// Valid extracted binary location.
	localBin := filepath.Join(root, "opt", "osquery", "bin", "osqueryd")
	if err := os.MkdirAll(filepath.Dir(localBin), 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(localBin, []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatal(err)
	}

	// Escaping symlink path that should be ignored.
	external := filepath.Join(t.TempDir(), "osqueryd-external")
	if err := os.WriteFile(external, []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatal(err)
	}
	symlinkPath := filepath.Join(root, "usr", "bin", "osqueryd")
	if err := os.MkdirAll(filepath.Dir(symlinkPath), 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(external, symlinkPath); err != nil {
		t.Fatal(err)
	}

	got, err := locateBinDir(root, "linux")
	if err != nil {
		t.Fatalf("locateBinDir failed: %v", err)
	}
	expected := filepath.Dir(localBin)
	if got != expected {
		t.Fatalf("expected local binary dir %s, got %s", expected, got)
	}
}

func TestEnsureReuseInstalledByChecksum(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test uses shell-script osqueryd fixture")
	}

	log := logp.NewLogger("artifact_test")
	version := "5.19.0"
	artifactBytes := buildTarGzArtifact(t, version, true)
	sum := sha256.Sum256(artifactBytes)
	sha := hex.EncodeToString(sum[:])

	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		_, _ = w.Write(artifactBytes)
	}))
	defer server.Close()

	cfg := testInstallConfig(server.URL+"/osquery.tar.gz", sha)

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
	artifactV1 := buildTarGzArtifact(t, "5.19.0", true)
	artifactV2 := buildTarGzArtifact(t, "5.20.0", true)

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
	cfg1 := testInstallConfig(server.URL+"/v1/osquery.tar.gz", sha1)
	cfg2 := testInstallConfig(server.URL+"/v2/osquery.tar.gz", sha2)

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

func TestEnsureConcurrentCallsUseSingleInstallFlow(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test uses shell-script osqueryd fixture")
	}

	log := logp.NewLogger("artifact_test")
	version := "5.19.0"
	artifactBytes := buildTarGzArtifact(t, version, true)
	sum := sha256.Sum256(artifactBytes)
	sha := hex.EncodeToString(sum[:])

	var (
		requests int
		mu       sync.Mutex
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requests++
		mu.Unlock()
		_, _ = w.Write(artifactBytes)
	}))
	defer server.Close()

	cfg := testInstallConfig(server.URL+"/osquery.tar.gz", sha)
	installDir := t.TempDir()

	var wg sync.WaitGroup
	errCh := make(chan error, 2)
	results := make(chan Result, 2)

	wg.Add(2)
	for i := 0; i < 2; i++ {
		go func() {
			defer wg.Done()
			res, err := Ensure(context.Background(), cfg, installDir, log)
			if err != nil {
				errCh <- err
				return
			}
			results <- res
		}()
	}
	wg.Wait()
	close(errCh)
	close(results)

	for err := range errCh {
		if err != nil {
			t.Fatalf("ensure failed: %v", err)
		}
	}

	got := make([]Result, 0, 2)
	for res := range results {
		got = append(got, res)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 successful results, got %d", len(got))
	}
	if got[0].BinDir != got[1].BinDir {
		t.Fatalf("expected same bin dir, got %s and %s", got[0].BinDir, got[1].BinDir)
	}
	if got[0].Version != version || got[1].Version != version {
		t.Fatalf("unexpected versions: %s and %s", got[0].Version, got[1].Version)
	}

	mu.Lock()
	defer mu.Unlock()
	if requests != 1 {
		t.Fatalf("expected a single download request, got %d", requests)
	}
}

func TestEnsureWithoutExtensionInArtifactSucceeds(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test uses shell-script osqueryd fixture")
	}

	log := logp.NewLogger("artifact_test")
	version := "5.19.0"
	artifactBytes := buildTarGzArtifact(t, version, false)
	sum := sha256.Sum256(artifactBytes)
	sha := hex.EncodeToString(sum[:])

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(artifactBytes)
	}))
	defer server.Close()

	cfg := testInstallConfig(server.URL+"/osquery.tar.gz", sha)

	res, err := Ensure(context.Background(), cfg, t.TempDir(), log)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Version != version {
		t.Fatalf("expected version %s, got %s", version, res.Version)
	}
}

func TestEnsurePersistsMinimalRuntimePayload(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test uses shell-script osqueryd fixture")
	}

	log := logp.NewLogger("artifact_test")
	version := "5.19.0"
	artifactBytes := buildTarGzArtifactWithExtra(t, version, "docs/README.txt", []byte("extra"))
	sum := sha256.Sum256(artifactBytes)
	sha := hex.EncodeToString(sum[:])

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(artifactBytes)
	}))
	defer server.Close()

	cfg := testInstallConfig(server.URL+"/osquery.tar.gz", sha)
	installDir := t.TempDir()

	res, err := Ensure(context.Background(), cfg, installDir, log)
	if err != nil {
		t.Fatalf("ensure failed: %v", err)
	}
	if res.BinDir != filepath.Join(installDir, releasesDirName, sha) {
		t.Fatalf("unexpected bin dir: %s", res.BinDir)
	}
	if _, err := os.Stat(filepath.Join(res.BinDir, "docs", "README.txt")); !os.IsNotExist(err) {
		t.Fatalf("unexpected extra extracted file persisted, err=%v", err)
	}
}

func buildTarGzArtifact(t *testing.T, version string, includeExtension bool) []byte {
	t.Helper()

	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)

	binPath := "osqueryd"
	if runtime.GOOS == "darwin" {
		binPath = filepath.Join("osquery.app", "Contents", "MacOS", "osqueryd")
		// The tar extractor creates files directly and expects parent directories
		// to already exist, so add explicit dir headers for Darwin app bundles.
		for _, dir := range []string{
			"osquery.app",
			filepath.Join("osquery.app", "Contents"),
			filepath.Join("osquery.app", "Contents", "MacOS"),
		} {
			dirHdr := &tar.Header{
				Name:     dir,
				Mode:     0755,
				Typeflag: tar.TypeDir,
			}
			if err := tw.WriteHeader(dirHdr); err != nil {
				t.Fatalf("write tar dir header failed: %v", err)
			}
		}
	}
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

	if includeExtension {
		extPath := "osquery-extension.ext"
		extContent := []byte("extension")
		extHdr := &tar.Header{
			Name: extPath,
			Mode: 0644,
			Size: int64(len(extContent)),
		}
		if err := tw.WriteHeader(extHdr); err != nil {
			t.Fatalf("write extension tar header failed: %v", err)
		}
		if _, err := tw.Write(extContent); err != nil {
			t.Fatalf("write extension tar body failed: %v", err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("close tar writer failed: %v", err)
	}
	if err := gzw.Close(); err != nil {
		t.Fatalf("close gzip writer failed: %v", err)
	}
	return buf.Bytes()
}

func buildTarGzArtifactWithExtra(t *testing.T, version, extraPath string, extraContent []byte) []byte {
	t.Helper()
	base := buildTarGzArtifact(t, version, false)

	reader, err := gzip.NewReader(bytes.NewReader(base))
	if err != nil {
		t.Fatalf("open gzip reader failed: %v", err)
	}
	defer reader.Close()
	tr := tar.NewReader(reader)

	var out bytes.Buffer
	gzw := gzip.NewWriter(&out)
	tw := tar.NewWriter(gzw)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("read tar entry failed: %v", err)
		}
		copiedHdr := *hdr
		if err := tw.WriteHeader(&copiedHdr); err != nil {
			t.Fatalf("write tar header failed: %v", err)
		}
		if _, err := io.Copy(tw, tr); err != nil {
			t.Fatalf("copy tar entry failed: %v", err)
		}
	}

	extraHdr := &tar.Header{
		Name: extraPath,
		Mode: 0644,
		Size: int64(len(extraContent)),
	}
	if err := tw.WriteHeader(extraHdr); err != nil {
		t.Fatalf("write extra tar header failed: %v", err)
	}
	if _, err := tw.Write(extraContent); err != nil {
		t.Fatalf("write extra tar body failed: %v", err)
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("close tar writer failed: %v", err)
	}
	if err := gzw.Close(); err != nil {
		t.Fatalf("close gzip writer failed: %v", err)
	}
	return out.Bytes()
}
