// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package httplog

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/windows"
)

func TestEvalSymlinksGlobChars(t *testing.T) {
	base := t.TempDir()
	tests := []struct {
		name string
		path string
	}{
		{name: "star", path: filepath.Join(base, "file-*.log")},
		{name: "question", path: filepath.Join(base, "file-?.log")},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := filepath.EvalSymlinks(test.path)
			if err == nil {
				t.Fatal("expected error from EvalSymlinks, got nil")
			}
			if !errors.Is(err, windows.ERROR_INVALID_NAME) {
				t.Fatalf("expected ERROR_INVALID_NAME, got: %v", err)
			}
		})
	}
}

func TestEvalSymlinksReservedNames(t *testing.T) {
	// Reserved device names return ErrNotExist, not ERROR_INVALID_NAME,
	// so they are handled by the existing fs.ErrNotExist check.
	base := t.TempDir()
	for _, name := range []string{"CON", "PRN", "AUX"} {
		t.Run(name, func(t *testing.T) {
			p := filepath.Join(base, name, "somefile.log")
			if err := os.MkdirAll(filepath.Dir(p), 0o750); err == nil {
				return
			}
			_, err := filepath.EvalSymlinks(p)
			if err == nil {
				t.Fatal("expected error from EvalSymlinks, got nil")
			}
			if errors.Is(err, windows.ERROR_INVALID_NAME) {
				t.Fatal("unexpectedly got ERROR_INVALID_NAME; reserved names should return ErrNotExist")
			}
			if !errors.Is(err, fs.ErrNotExist) {
				t.Fatalf("expected fs.ErrNotExist, got: %v", err)
			}
		})
	}
}

func TestResolveSymlinksWindowsGlob(t *testing.T) {
	base := t.TempDir()
	resolved, err := filepath.EvalSymlinks(base)
	if err != nil {
		t.Fatalf("failed to resolve temp dir: %v", err)
	}
	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "star",
			path: filepath.Join(base, "target-*.log"),
			want: filepath.Join(resolved, "target-*.log"),
		},
		{
			name: "question",
			path: filepath.Join(base, "target-?.log"),
			want: filepath.Join(resolved, "target-?.log"),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := resolveSymlinks(test.path)
			if err != nil {
				t.Fatalf("unexpected error from resolveSymlinks: %v", err)
			}
			if got != test.want {
				t.Errorf("unexpected result: got %q, want %q", got, test.want)
			}
		})
	}
}

func TestResolveSymlinksWindowsReservedName(t *testing.T) {
	base := t.TempDir()
	resolved, err := filepath.EvalSymlinks(base)
	if err != nil {
		t.Fatalf("failed to resolve temp dir: %v", err)
	}
	for _, name := range []string{"CON", "PRN", "AUX"} {
		t.Run(name, func(t *testing.T) {
			p := filepath.Join(base, name, "somefile.log")
			got, err := resolveSymlinks(p)
			if err != nil {
				t.Fatalf("unexpected error from resolveSymlinks: %v", err)
			}
			want := filepath.Join(resolved, name, "somefile.log")
			if got != want {
				t.Errorf("unexpected result: got %q, want %q", got, want)
			}
		})
	}
}

func TestIsPathInWindowsGlob(t *testing.T) {
	base := t.TempDir()
	tests := []struct {
		name string
		root string
		path string
		want bool
	}{
		{
			name: "glob_in_root",
			root: base,
			path: filepath.Join(base, "target-*.log"),
			want: true,
		},
		{
			name: "glob_in_subdir",
			root: base,
			path: filepath.Join(base, "subdir", "target-*.log"),
			want: true,
		},
		{
			name: "glob_escapes_root",
			root: base,
			path: filepath.Join(base, "..", "target-*.log"),
			want: false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := IsPathIn(test.root, test.path)
			if err != nil {
				t.Fatalf("unexpected error from IsPathIn: %v", err)
			}
			if got != test.want {
				t.Errorf("unexpected result: got %t, want %t", got, test.want)
			}
		})
	}
}
