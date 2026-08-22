// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestFetchSidecarHash(t *testing.T) {
	const validHash = "9f40cea0358759ab2ee871c577055657e3cc2c7cbe5c1247f764245941178aa6"
	cases := []struct {
		name    string
		body    string
		wantErr bool
	}{
		{"bare hash", validHash, false},
		{"shasum format", validHash + "  osquery-5.23.1.pkg", false},
		{"empty", "", true},
		{"invalid hash", "notahash", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte(tc.body))
			}))
			defer srv.Close()
			got, err := fetchSidecarHash(http.DefaultClient, srv.URL)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %q", got)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != validHash {
				t.Fatalf("got %q, want %q", got, validHash)
			}
		})
	}
}

func TestUpdateDistroFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "distro.go")
	input := `const (
	osqueryVersion = "1.0.0"
	osqueryDistroDarwinSHA256 = "old"
)
`
	if err := os.WriteFile(path, []byte(input), 0o644); err != nil {
		t.Fatal(err)
	}
	changed, err := updateDistroFile(path, "1.0.1", map[string]string{"osqueryDistroDarwinSHA256": "new"})
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected file to change")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `osqueryVersion = "1.0.1"`) || !strings.Contains(string(b), `osqueryDistroDarwinSHA256 = "new"`) {
		t.Fatalf("unexpected output:\n%s", b)
	}
}

func TestEnsureChangelogFragmentIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	now := time.Unix(123, 0)
	if err := ensureChangelogFragment(dir, "1.0.1", now); err != nil {
		t.Fatal(err)
	}
	if err := ensureChangelogFragment(dir, "1.0.1", now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected one fragment, got %d", len(entries))
	}
}
