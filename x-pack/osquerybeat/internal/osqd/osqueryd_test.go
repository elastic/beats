// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package osqd

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
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
	if err := prepareAutoloadFile(extensionAutoloadPath, mandatoryExtensionPath, nil, validLogger); err != nil {
		t.Fatalf("prepareAutoloadFile failed: %v", err)
	}
	if err := verifyAutoloadFile(extensionAutoloadPath, mandatoryExtensionPath); err != nil {
		t.Fatalf("verifyAutoloadFile failed: %v", err)
	}

	// Idempotency: running it again should still validate
	if err := prepareAutoloadFile(extensionAutoloadPath, mandatoryExtensionPath, nil, validLogger); err != nil {
		t.Fatalf("prepareAutoloadFile second run failed: %v", err)
	}
	if err := verifyAutoloadFile(extensionAutoloadPath, mandatoryExtensionPath); err != nil {
		t.Fatalf("verifyAutoloadFile second run failed: %v", err)
	}
}

// TestPrepareAutoloadFileWithExtensions verifies that customer-managed extension
// paths are appended after the mandatory Elastic extension, that the mandatory
// extension always stays on the first line, and that removing an extension rewrites
// the file.
func TestPrepareAutoloadFileWithExtensions(t *testing.T) {
	validLogger := logp.NewLogger("osqueryd_test")

	dir := t.TempDir()
	mandatoryExtensionPath := filepath.Join(dir, extensionName)
	if err := os.WriteFile(mandatoryExtensionPath, nil, 0600); err != nil {
		t.Fatal(err)
	}

	ext1 := filepath.Join(dir, "custom1.ext")
	ext2 := filepath.Join(dir, "custom2.ext")
	for _, p := range []string{ext1, ext2} {
		if err := os.WriteFile(p, nil, 0700); err != nil {
			t.Fatal(err)
		}
	}

	extensionAutoloadPath := filepath.Join(dir, osqueryAutoload)

	// Write with two extra extensions.
	if err := prepareAutoloadFile(extensionAutoloadPath, mandatoryExtensionPath, []string{ext1, ext2, ext1}, validLogger); err != nil {
		t.Fatalf("prepareAutoloadFile failed: %v", err)
	}
	got, err := os.ReadFile(extensionAutoloadPath)
	if err != nil {
		t.Fatal(err)
	}
	want := mandatoryExtensionPath + "\n" + ext1 + "\n" + ext2
	if string(got) != want {
		t.Fatalf("autoload content mismatch:\n got: %q\nwant: %q", string(got), want)
	}

	// Removing an extension should rewrite the file with only the remaining one.
	if err := prepareAutoloadFile(extensionAutoloadPath, mandatoryExtensionPath, []string{ext2}, validLogger); err != nil {
		t.Fatalf("prepareAutoloadFile rewrite failed: %v", err)
	}
	got, err = os.ReadFile(extensionAutoloadPath)
	if err != nil {
		t.Fatal(err)
	}
	want = mandatoryExtensionPath + "\n" + ext2
	if string(got) != want {
		t.Fatalf("autoload content mismatch after removal:\n got: %q\nwant: %q", string(got), want)
	}
}

// TestResolveExtensionsDirectory verifies directory entries are scanned for
// extension binaries, non-matching or invalid files are skipped, and entry-level
// errors are reported.
func TestResolveExtensionsDirectory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission-based pre-check differs on windows")
	}

	dir := t.TempDir()

	okExt := filepath.Join(dir, "good.ext")
	if err := os.WriteFile(okExt, nil, 0700); err != nil {
		t.Fatal(err)
	}
	// Non-executable extension binary: skipped.
	nonExec := filepath.Join(dir, "noexec.ext")
	if err := os.WriteFile(nonExec, nil, 0600); err != nil {
		t.Fatal(err)
	}
	// Wrong suffix: ignored (not counted as an extension at all).
	if err := os.WriteFile(filepath.Join(dir, "readme.txt"), nil, 0700); err != nil {
		t.Fatal(err)
	}
	// Nested dir: ignored.
	if err := os.Mkdir(filepath.Join(dir, "nested"), 0700); err != nil {
		t.Fatal(err)
	}

	results := ResolveExtensions([]string{dir, "relative/dir", filepath.Join(dir, "missing")})
	if len(results) != 3 {
		t.Fatalf("expected 3 entry results, got %d: %+v", len(results), results)
	}

	// First entry (directory): one loaded (good.ext), one skipped (noexec.ext).
	first := results[0]
	if first.Error != "" {
		t.Fatalf("unexpected entry error: %v", first.Error)
	}
	if len(first.Loaded) != 1 || first.Loaded[0] != okExt {
		t.Fatalf("expected only %q loaded, got %v", okExt, first.Loaded)
	}
	if len(first.Skipped) != 1 || first.Skipped[0].Path != nonExec {
		t.Fatalf("expected %q skipped, got %v", nonExec, first.Skipped)
	}

	// Relative path is rejected.
	if results[1].Error == "" {
		t.Fatalf("expected error for relative path")
	}
	// Missing literal path is rejected.
	if results[2].Error == "" {
		t.Fatalf("expected error for missing path")
	}
}

// TestResolveExtensionsFile verifies a literal file entry is autoloaded directly
// (no suffix filter) and a missing file entry is an error.
func TestResolveExtensionsFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission-based pre-check differs on windows")
	}

	dir := t.TempDir()
	// No .ext suffix on purpose: an explicit file is loaded regardless of suffix.
	file := filepath.Join(dir, "myextension")
	if err := os.WriteFile(file, nil, 0700); err != nil {
		t.Fatal(err)
	}

	results := ResolveExtensions([]string{file})
	if len(results) != 1 || results[0].Error != "" {
		t.Fatalf("unexpected results: %+v", results)
	}
	if len(results[0].Loaded) != 1 || results[0].Loaded[0] != file {
		t.Fatalf("expected %q loaded, got %v", file, results[0].Loaded)
	}
}

// TestResolveExtensionsGlob verifies a glob pattern resolves to matching files,
// skipping invalid matches.
func TestResolveExtensionsGlob(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission-based pre-check differs on windows")
	}

	dir := t.TempDir()
	a := filepath.Join(dir, "a.ext")
	b := filepath.Join(dir, "b.ext")
	if err := os.WriteFile(a, nil, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b, nil, 0600); err != nil { // non-executable -> skipped
		t.Fatal(err)
	}
	// Does not match the pattern.
	if err := os.WriteFile(filepath.Join(dir, "c.so"), nil, 0700); err != nil {
		t.Fatal(err)
	}

	results := ResolveExtensions([]string{filepath.Join(dir, "*.ext")})
	if len(results) != 1 || results[0].Error != "" {
		t.Fatalf("unexpected results: %+v", results)
	}
	if len(results[0].Loaded) != 1 || results[0].Loaded[0] != a {
		t.Fatalf("expected %q loaded, got %v", a, results[0].Loaded)
	}
	if len(results[0].Skipped) != 1 || results[0].Skipped[0].Path != b {
		t.Fatalf("expected %q skipped, got %v", b, results[0].Skipped)
	}
}

// TestCollectExtensionBinaries verifies resolved entries are flattened into the
// ordered list of valid binary paths used for the autoload file, deduplicating
// overlapping entries.
func TestCollectExtensionBinaries(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission-based pre-check differs on windows")
	}

	dir := t.TempDir()
	a := filepath.Join(dir, "a.ext")
	b := filepath.Join(dir, "b.ext")
	for _, p := range []string{a, b} {
		if err := os.WriteFile(p, nil, 0700); err != nil {
			t.Fatal(err)
		}
	}

	q := &OSQueryD{log: logp.NewLogger("osqueryd_test")}
	// Directory entry plus an overlapping explicit file entry: a is not duplicated.
	got := q.collectExtensionBinaries([]string{dir, a})
	if len(got) != 2 || got[0] != a || got[1] != b {
		t.Fatalf("unexpected collected binaries: %v", got)
	}
}
