// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package distro

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmbeddedMetadata(t *testing.T) {
	assert.NotEmpty(t, OsquerydVersion(), "embedded osquery version should be set")

	platforms := []OSArch{
		{OS: "linux", Arch: "amd64"},
		{OS: "linux", Arch: "arm64"},
		{OS: "darwin", Arch: "amd64"},
		{OS: "darwin", Arch: "arm64"},
		{OS: "windows", Arch: "amd64"},
		{OS: "windows", Arch: "arm64"},
	}
	for _, osarch := range platforms {
		spec, err := GetSpec(osarch)
		require.NoError(t, err, "missing spec for %s", osarch)
		assert.Len(t, spec.SHA256Hash, 64, "checksum for %s should be a SHA-256 hex digest", osarch)
		assert.NotEmpty(t, spec.PackSuffix, "pack suffix for %s should be set", osarch)
	}

	_, err := GetSpec(OSArch{OS: "plan9", Arch: "amd64"})
	assert.ErrorIs(t, err, ErrUnsupportedOS, "unsupported platform should return ErrUnsupportedOS")
}

func TestEmbeddedMetadataIsCanonical(t *testing.T) {
	meta, err := ParseMetadata(distroJSON)
	require.NoError(t, err, "embedded distro.json must parse")

	encoded, err := encodeMetadata(meta)
	require.NoError(t, err, "embedded metadata must encode")

	got := bytes.ReplaceAll(distroJSON, []byte("\r\n"), []byte("\n"))
	assert.Equal(t, string(encoded), string(got), "distro.json should use canonical encoding")
}

func TestParseMetadata(t *testing.T) {
	valid := testMetadata("1.0.0", strings.Repeat("a", 64))
	validJSON, err := encodeMetadata(valid)
	require.NoError(t, err, "valid metadata should encode")

	tests := []struct {
		name    string
		raw     string
		wantErr string
	}{
		{name: "valid", raw: string(validJSON)},
		{name: "invalid json", raw: `{`, wantErr: "unexpected EOF"},
		{name: "unknown field", raw: `{"version":"1.0.0","extra":true}`, wantErr: "unknown field"},
		{name: "empty version", raw: `{"version":"","checksums":{"darwin":"` + strings.Repeat("a", 64) + `","linux_amd64":"` + strings.Repeat("a", 64) + `","linux_arm64":"` + strings.Repeat("a", 64) + `","windows_arm64":"` + strings.Repeat("a", 64) + `","windows_amd64":"` + strings.Repeat("a", 64) + `"}}`, wantErr: "version is empty"},
		{name: "bad checksum", raw: `{"version":"1.0.0","checksums":{"darwin":"notahash","linux_amd64":"` + strings.Repeat("a", 64) + `","linux_arm64":"` + strings.Repeat("a", 64) + `","windows_arm64":"` + strings.Repeat("a", 64) + `","windows_amd64":"` + strings.Repeat("a", 64) + `"}}`, wantErr: "checksums.darwin"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseMetadata([]byte(tc.raw))
			if tc.wantErr == "" {
				assert.NoError(t, err, "valid metadata should parse")
				return
			}
			assert.ErrorContains(t, err, tc.wantErr, "expected parse error")
		})
	}
}

func TestWriteMetadataFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "distro.json")
	meta := testMetadata("1.0.0", strings.Repeat("a", 64))

	changed, err := WriteMetadataFile(path, meta)
	require.NoError(t, err, "first write should succeed")
	assert.True(t, changed, "first write should create the file")

	changed, err = WriteMetadataFile(path, meta)
	require.NoError(t, err, "second write should succeed")
	assert.False(t, changed, "second write of the same metadata should be a no-op")

	got, err := ReadMetadataFile(path)
	require.NoError(t, err, "written metadata should parse")
	assert.Equal(t, meta, got, "round-trip should preserve metadata")
}

func testMetadata(version, hash string) Metadata {
	return Metadata{
		Version: version,
		Checksums: Checksums{
			Darwin:       hash,
			LinuxAMD64:   hash,
			LinuxARM64:   hash,
			WindowsARM64: hash,
			WindowsAMD64: hash,
		},
	}
}
