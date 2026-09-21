// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package distro

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

// distroJSON is the bundled Osquery version and official artifact checksums.
// Update it with `mage updateOsquery`.
//
//go:embed distro.json
var distroJSON []byte

var (
	osqueryVersion string
	specs          map[OSArch]Spec
)

// Metadata is the bundled Osquery release version and artifact checksums.
type Metadata struct {
	Version   string    `json:"version"`
	Checksums Checksums `json:"checksums"`
}

// Checksums holds SHA-256 digests of the official Osquery artifacts.
type Checksums struct {
	// Darwin is the SHA-256 of osquery-<ver>.pkg (shared by amd64 and arm64).
	Darwin string `json:"darwin"`
	// LinuxAMD64 is the SHA-256 of osquery-<ver>_1.linux_x86_64.tar.gz.
	LinuxAMD64 string `json:"linux_amd64"`
	// LinuxARM64 is the SHA-256 of osquery-<ver>_1.linux_aarch64.tar.gz.
	LinuxARM64 string `json:"linux_arm64"`
	// WindowsARM64 is the SHA-256 of osquery-<ver>.windows_arm64.zip.
	WindowsARM64 string `json:"windows_arm64"`
	// WindowsAMD64 is the SHA-256 of osquery-<ver>.windows_x86_64.zip.
	WindowsAMD64 string `json:"windows_amd64"`
}

func init() {
	meta, err := ParseMetadata(distroJSON)
	if err != nil {
		panic("invalid distro.json: " + err.Error())
	}
	osqueryVersion = meta.Version
	specs = specsFromChecksums(meta.Checksums)
}

func specsFromChecksums(c Checksums) map[OSArch]Spec {
	return map[OSArch]Spec{
		{"linux", "amd64"}:   {"_1.linux_x86_64.tar.gz", c.LinuxAMD64, true},
		{"linux", "arm64"}:   {"_1.linux_aarch64.tar.gz", c.LinuxARM64, true},
		{"darwin", "amd64"}:  {osqueryPkgExt, c.Darwin, true},
		{"darwin", "arm64"}:  {osqueryPkgExt, c.Darwin, true},
		{"windows", "amd64"}: {".windows_x86_64.zip", c.WindowsAMD64, true},
		{"windows", "arm64"}: {osqueryZipExt, c.WindowsARM64, true},
	}
}

func (m Metadata) validate() error {
	if strings.TrimSpace(m.Version) == "" {
		return errors.New("version is empty")
	}
	for _, field := range []struct {
		name  string
		value string
	}{
		{"darwin", m.Checksums.Darwin},
		{"linux_amd64", m.Checksums.LinuxAMD64},
		{"linux_arm64", m.Checksums.LinuxARM64},
		{"windows_arm64", m.Checksums.WindowsARM64},
		{"windows_amd64", m.Checksums.WindowsAMD64},
	} {
		if err := validateSHA256(field.value); err != nil {
			return fmt.Errorf("checksums.%s: %w", field.name, err)
		}
	}
	return nil
}

func validateSHA256(s string) error {
	decoded, err := hex.DecodeString(s)
	if err != nil || len(decoded) != sha256.Size {
		return fmt.Errorf("not a SHA-256 hex digest: %q", s)
	}
	return nil
}

// ParseMetadata decodes and validates bundled Osquery metadata.
func ParseMetadata(b []byte) (Metadata, error) {
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	var meta Metadata
	if err := dec.Decode(&meta); err != nil {
		return Metadata{}, err
	}
	if err := meta.validate(); err != nil {
		return Metadata{}, err
	}
	return meta, nil
}

func encodeMetadata(meta Metadata) ([]byte, error) {
	if err := meta.validate(); err != nil {
		return nil, err
	}
	b, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}

// ReadMetadataFile loads bundled Osquery metadata from path.
func ReadMetadataFile(path string) (Metadata, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Metadata{}, err
	}
	return ParseMetadata(b)
}

// WriteMetadataFile writes canonical metadata JSON to path.
// It returns whether the file contents changed.
func WriteMetadataFile(path string, meta Metadata) (bool, error) {
	encoded, err := encodeMetadata(meta)
	if err != nil {
		return false, err
	}
	existing, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return false, err
	}
	if bytes.Equal(existing, encoded) {
		return false, nil
	}
	return true, os.WriteFile(path, encoded, 0o644)
}
