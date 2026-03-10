// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package tar

import (
	"archive/tar"
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestExtractAllowsSymlinkEntries(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("creating symlinks may require elevated privileges on windows")
	}

	var b bytes.Buffer
	tw := tar.NewWriter(&b)
	t.Cleanup(func() { _ = tw.Close() })

	requireWriteHeader(t, tw, &tar.Header{Name: "usr/", Typeflag: tar.TypeDir, Mode: 0755})
	requireWriteHeader(t, tw, &tar.Header{Name: "usr/bin/", Typeflag: tar.TypeDir, Mode: 0755})

	content := []byte("osqueryd")
	requireWriteHeader(t, tw, &tar.Header{
		Name:     "usr/bin/osqueryd",
		Typeflag: tar.TypeReg,
		Mode:     0755,
		Size:     int64(len(content)),
	})
	_, err := tw.Write(content)
	if err != nil {
		t.Fatalf("failed to write regular file content: %v", err)
	}

	requireWriteHeader(t, tw, &tar.Header{
		Name:     "usr/bin/osqueryi",
		Typeflag: tar.TypeSymlink,
		Linkname: "osqueryd",
		Mode:     0755,
	})

	if err := tw.Close(); err != nil {
		t.Fatalf("failed to close tar writer: %v", err)
	}

	dir := t.TempDir()
	if err := Extract(bytes.NewReader(b.Bytes()), dir); err != nil {
		t.Fatalf("extract failed: %v", err)
	}

	linkPath := filepath.Join(dir, "usr", "bin", "osqueryi")
	info, err := os.Lstat(linkPath)
	if err != nil {
		t.Fatalf("expected symlink to exist: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("expected %s to be a symlink", linkPath)
	}
}

func TestExtractRejectsSymlinkEscapingDestination(t *testing.T) {
	var b bytes.Buffer
	tw := tar.NewWriter(&b)
	t.Cleanup(func() { _ = tw.Close() })

	requireWriteHeader(t, tw, &tar.Header{Name: "usr/", Typeflag: tar.TypeDir, Mode: 0755})
	requireWriteHeader(t, tw, &tar.Header{Name: "usr/bin/", Typeflag: tar.TypeDir, Mode: 0755})
	requireWriteHeader(t, tw, &tar.Header{
		Name:     "usr/bin/osqueryi",
		Typeflag: tar.TypeSymlink,
		Linkname: "../../../outside",
		Mode:     0755,
	})

	if err := tw.Close(); err != nil {
		t.Fatalf("failed to close tar writer: %v", err)
	}

	dir := t.TempDir()
	err := Extract(bytes.NewReader(b.Bytes()), dir)
	if err == nil {
		t.Fatal("expected error for escaping symlink target")
	}
}

func TestExtractSkipEscapingSkipsAbsoluteSymlinks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("creating symlinks may require elevated privileges on windows")
	}

	var b bytes.Buffer
	tw := tar.NewWriter(&b)
	t.Cleanup(func() { _ = tw.Close() })

	requireWriteHeader(t, tw, &tar.Header{Name: "opt/", Typeflag: tar.TypeDir, Mode: 0755})
	requireWriteHeader(t, tw, &tar.Header{Name: "opt/osquery/", Typeflag: tar.TypeDir, Mode: 0755})
	requireWriteHeader(t, tw, &tar.Header{Name: "opt/osquery/bin/", Typeflag: tar.TypeDir, Mode: 0755})

	content := []byte("osqueryd-binary")
	requireWriteHeader(t, tw, &tar.Header{
		Name:     "opt/osquery/bin/osqueryd",
		Typeflag: tar.TypeReg,
		Mode:     0755,
		Size:     int64(len(content)),
	})
	if _, err := tw.Write(content); err != nil {
		t.Fatalf("failed to write file content: %v", err)
	}

	requireWriteHeader(t, tw, &tar.Header{Name: "usr/", Typeflag: tar.TypeDir, Mode: 0755})
	requireWriteHeader(t, tw, &tar.Header{Name: "usr/bin/", Typeflag: tar.TypeDir, Mode: 0755})
	requireWriteHeader(t, tw, &tar.Header{
		Name:     "usr/bin/osqueryd",
		Typeflag: tar.TypeSymlink,
		Linkname: "/opt/osquery/bin/osqueryd",
		Mode:     0755,
	})

	if err := tw.Close(); err != nil {
		t.Fatalf("failed to close tar writer: %v", err)
	}

	t.Run("Extract hard-fails on absolute escaping symlink", func(t *testing.T) {
		dir := t.TempDir()
		err := Extract(bytes.NewReader(b.Bytes()), dir)
		if err == nil {
			t.Fatal("expected error from Extract for escaping symlink")
		}
		if !strings.Contains(err.Error(), "illegal symlink target") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("ExtractSkipEscaping skips absolute symlink and extracts regular file", func(t *testing.T) {
		dir := t.TempDir()
		if err := ExtractSkipEscaping(bytes.NewReader(b.Bytes()), dir); err != nil {
			t.Fatalf("ExtractSkipEscaping failed: %v", err)
		}

		binPath := filepath.Join(dir, "opt", "osquery", "bin", "osqueryd")
		if _, err := os.Stat(binPath); err != nil {
			t.Fatalf("expected regular binary to be extracted: %v", err)
		}

		symlinkPath := filepath.Join(dir, "usr", "bin", "osqueryd")
		if _, err := os.Lstat(symlinkPath); !os.IsNotExist(err) {
			t.Fatalf("expected escaping symlink to be skipped, got err=%v", err)
		}
	})
}

func TestExtractSkipEscapingKeepsValidSymlinks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("creating symlinks may require elevated privileges on windows")
	}

	var b bytes.Buffer
	tw := tar.NewWriter(&b)
	t.Cleanup(func() { _ = tw.Close() })

	requireWriteHeader(t, tw, &tar.Header{Name: "bin/", Typeflag: tar.TypeDir, Mode: 0755})
	content := []byte("osqueryd")
	requireWriteHeader(t, tw, &tar.Header{
		Name: "bin/osqueryd", Typeflag: tar.TypeReg, Mode: 0755, Size: int64(len(content)),
	})
	if _, err := tw.Write(content); err != nil {
		t.Fatal(err)
	}
	requireWriteHeader(t, tw, &tar.Header{
		Name: "bin/osqueryi", Typeflag: tar.TypeSymlink, Linkname: "osqueryd", Mode: 0755,
	})
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	if err := ExtractSkipEscaping(bytes.NewReader(b.Bytes()), dir); err != nil {
		t.Fatalf("ExtractSkipEscaping failed: %v", err)
	}

	linkPath := filepath.Join(dir, "bin", "osqueryi")
	info, err := os.Lstat(linkPath)
	if err != nil {
		t.Fatalf("expected valid symlink to be extracted: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("expected %s to be a symlink", linkPath)
	}
}

func requireWriteHeader(t *testing.T, tw *tar.Writer, hdr *tar.Header) {
	t.Helper()
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatalf("failed to write tar header %s: %v", hdr.Name, err)
	}
}
