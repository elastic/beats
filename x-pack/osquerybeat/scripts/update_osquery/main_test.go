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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/distro"
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
				assert.Error(t, err, "expected error, got %q", got)
				return
			}
			require.NoError(t, err, "fetchSidecarHash should succeed")
			assert.Equal(t, validHash, got, "unexpected sidecar hash")
		})
	}
}

func TestUpdateDistroFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "distro.json")
	oldHash := strings.Repeat("a", 64)
	newHash := strings.Repeat("b", 64)
	_, err := distro.WriteMetadataFile(path, distro.Metadata{
		Version: "1.0.0",
		Checksums: distro.Checksums{
			Darwin:       oldHash,
			LinuxAMD64:   oldHash,
			LinuxARM64:   oldHash,
			WindowsARM64: oldHash,
			WindowsAMD64: oldHash,
		},
	})
	require.NoError(t, err, "setup distro.json")

	hashes := map[string]string{
		"darwin":        newHash,
		"linux_amd64":   newHash,
		"linux_arm64":   newHash,
		"windows_arm64": newHash,
		"windows_amd64": newHash,
	}
	changed, err := updateDistroFile(path, "1.0.1", hashes)
	require.NoError(t, err, "updateDistroFile should succeed")
	assert.True(t, changed, "expected distro.json to change")

	got, err := distro.ReadMetadataFile(path)
	require.NoError(t, err, "updated distro.json should parse")
	assert.Equal(t, "1.0.1", got.Version, "version should be updated")
	assert.Equal(t, newHash, got.Checksums.Darwin, "darwin checksum should be updated")
	assert.Equal(t, newHash, got.Checksums.LinuxAMD64, "linux_amd64 checksum should be updated")
	assert.Equal(t, newHash, got.Checksums.LinuxARM64, "linux_arm64 checksum should be updated")
	assert.Equal(t, newHash, got.Checksums.WindowsARM64, "windows_arm64 checksum should be updated")
	assert.Equal(t, newHash, got.Checksums.WindowsAMD64, "windows_amd64 checksum should be updated")

	changed, err = updateDistroFile(path, "1.0.1", hashes)
	require.NoError(t, err, "second updateDistroFile should succeed")
	assert.False(t, changed, "expected no change when metadata is already current")
}

func TestEnsureChangelogFragmentIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	now := time.Unix(123, 0)
	require.NoError(t, ensureChangelogFragment(dir, "1.0.1", now), "first changelog write should succeed")
	require.NoError(t, ensureChangelogFragment(dir, "1.0.1", now.Add(time.Second)), "second changelog write should succeed")
	entries, err := os.ReadDir(dir)
	require.NoError(t, err, "read changelog dir")
	assert.Len(t, entries, 1, "expected one fragment")
}
