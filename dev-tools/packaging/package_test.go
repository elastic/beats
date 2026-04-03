// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package dev_tools

// This file contains tests that can be run on the generated packages.
// To run these tests use `go test package_test.go`.

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"debug/buildinfo"
	"debug/elf"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/blakesmith/ar"
	rpm "github.com/cavaliergopher/rpm"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/docker/client"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/dev-tools/mage"
)

const (
	expectedConfigMode     = os.FileMode(0o600)
	expectedManifestMode   = os.FileMode(0o644)
	expectedModuleFileMode = expectedManifestMode
	expectedModuleDirMode  = os.FileMode(0o755)
)

var (
	excludedPathsPattern   = regexp.MustCompile(`node_modules`)
	configFilePattern      = regexp.MustCompile(`/(\w+beat\.yml|apm-server\.yml|elastic-agent\.yml)$`)
	manifestFilePattern    = regexp.MustCompile(`manifest.yml`)
	modulesDirPattern      = regexp.MustCompile(`module/.+`)
	modulesDDirPattern     = regexp.MustCompile(`modules.d/$`)
	modulesDFilePattern    = regexp.MustCompile(`modules.d/.+`)
	monitorsDFilePattern   = regexp.MustCompile(`monitors.d/.+`)
	systemdUnitFilePattern = regexp.MustCompile(`/lib/systemd/system/.*\.service`)
	fipsPackagePattern     = regexp.MustCompile(`\w+-fips-\w+`)
	licenseFiles           = []string{"LICENSE.txt", "NOTICE.txt"}
)

var (
	files             = flag.String("files", "../build/distributions/*/*", "filepath glob containing package files")
	modules           = flag.Bool("modules", false, "check modules folder contents")
	minModules        = flag.Int("min-modules", 4, "minimum number of modules to expect in modules folder")
	modulesd          = flag.Bool("modules.d", false, "check modules.d folder contents")
	monitorsd         = flag.Bool("monitors.d", false, "check monitors.d folder contents")
	rootOwner         = flag.Bool("root-owner", false, "expect root to own package files")
	rootUserContainer = flag.Bool("root-user-container", false, "expect root in container user")
)

type dockerImageType string

const (
	dockerImageTypeLegacy dockerImageType = "legacy"
	dockerImageTypeOCI    dockerImageType = "oci"
)

var errDockerArchiveWalkDone = errors.New("docker archive walk done")
var errDockerArchiveEntryNotFound = errors.New("docker archive entry not found")

func TestRPM(t *testing.T) {
	rpms := getFiles(t, regexp.MustCompile(`\.rpm$`))
	for _, rpm := range rpms {
		checkRPM(t, rpm)
	}
}

func TestDeb(t *testing.T) {
	debs := getFiles(t, regexp.MustCompile(`\.deb$`))
	buf := new(bytes.Buffer)
	for _, deb := range debs {
		fipsPackage := fipsPackagePattern.MatchString(deb)
		checkDeb(t, deb, buf, fipsPackage)
	}
}

func TestTar(t *testing.T) {
	tars := getFiles(t, regexp.MustCompile(`^-\w+\.tar\.gz$`))
	for _, tarFile := range tars {
		if strings.HasSuffix(tarFile, "docker.tar.gz") {
			// We should skip the docker images archives , since those have their dedicated check
			continue
		}
		fipsPackage := fipsPackagePattern.MatchString(tarFile)
		checkTar(t, tarFile, fipsPackage)
	}
}

func TestZip(t *testing.T) {
	zips := getFiles(t, regexp.MustCompile(`^\w+beat-\S+.zip$`))
	for _, zip := range zips {
		checkZip(t, zip)
	}
}

func TestDocker(t *testing.T) {
	dockers := getFiles(t, regexp.MustCompile(`\.docker\.tar\.gz$`))
	for _, docker := range dockers {
		t.Log(docker)
		checkDocker(t, docker)
	}
}

func TestDetectDockerImageType(t *testing.T) {
	t.Run("legacy archive", func(t *testing.T) {
		dockerFile := createTestDockerArchive(t, []testTarEntry{
			{name: "manifest.json", mode: 0o644, data: []byte("[]")},
		})

		imageType, err := detectDockerImageType(dockerFile)
		require.NoError(t, err, "legacy docker image format detection should not return an error")
		require.Equal(t, dockerImageTypeLegacy, imageType, "expected legacy docker archive type")
	})

	t.Run("oci archive", func(t *testing.T) {
		dockerFile := createTestDockerArchive(t, []testTarEntry{
			{name: "manifest.json", mode: 0o644, data: []byte("[]")},
			{name: "index.json", mode: 0o644, data: []byte(`{"manifests":[]}`)},
			{name: "oci-layout", mode: 0o644, data: []byte(`{"imageLayoutVersion":"1.0.0"}`)},
		})

		imageType, err := detectDockerImageType(dockerFile)
		require.NoError(t, err, "OCI docker image format detection should not return an error")
		require.Equal(t, dockerImageTypeOCI, imageType, "expected OCI docker archive type when OCI markers are present")
	})
}

func TestReadDockerOCI(t *testing.T) {
	configData := []byte(`{"config":{"Entrypoint":["/docker-entrypoint"],"Labels":{"org.label-schema.vendor":"Elastic"},"User":"root","WorkingDir":"/usr/share/testbeat"}}`)

	layerTar := createTestTarData(t, []testTarEntry{
		{name: "docker-entrypoint", mode: 0o755, data: []byte("#!/bin/sh\n")},
		{name: "usr/share/testbeat/testbeat.yml", mode: 0o644, data: []byte("name: testbeat\n")},
		{name: "usr/share/testbeat/LICENSE.txt", mode: 0o644, data: []byte("license\n")},
		{name: "etc/passwd", mode: 0o644, data: []byte("x\n")},
	})
	layerData := gzipTestData(t, layerTar)

	configDigest := sha256Digest(configData)
	layerDigest := sha256Digest(layerData)

	manifest := dockerOCIManifest{
		SchemaVersion: 2,
		MediaType:     dockerOCIManifestMediaType,
		Config: dockerOCIManifestDescriptor{
			MediaType: "application/vnd.oci.image.config.v1+json",
			Digest:    configDigest,
			Size:      int64(len(configData)),
		},
		Layers: []dockerOCIManifestDescriptor{
			{
				MediaType: "application/vnd.oci.image.layer.v1.tar+gzip",
				Digest:    layerDigest,
				Size:      int64(len(layerData)),
			},
		},
	}
	manifestData, err := json.Marshal(manifest)
	require.NoError(t, err, "OCI manifest marshaling should not fail")

	manifestDigest := sha256Digest(manifestData)
	index := dockerOCIIndex{
		SchemaVersion: 2,
		Manifests: []dockerOCIManifestDescriptor{
			{
				MediaType: dockerOCIManifestMediaType,
				Digest:    manifestDigest,
				Size:      int64(len(manifestData)),
			},
		},
	}
	indexData, err := json.Marshal(index)
	require.NoError(t, err, "OCI index marshaling should not fail")

	manifestPath, err := ociBlobPathFromDigest(manifestDigest)
	require.NoError(t, err, "manifest digest should produce a valid OCI blob path")
	configPath, err := ociBlobPathFromDigest(configDigest)
	require.NoError(t, err, "config digest should produce a valid OCI blob path")
	layerPath, err := ociBlobPathFromDigest(layerDigest)
	require.NoError(t, err, "layer digest should produce a valid OCI blob path")

	dockerFile := createTestDockerArchive(t, []testTarEntry{
		{name: "oci-layout", mode: 0o644, data: []byte(`{"imageLayoutVersion":"1.0.0"}`)},
		{name: "index.json", mode: 0o644, data: indexData},
		{name: manifestPath, mode: 0o644, data: manifestData},
		{name: configPath, mode: 0o644, data: configData},
		{name: layerPath, mode: 0o644, data: layerData},
	})

	pkg, info, err := readDockerOCI(dockerFile)
	require.NoError(t, err, "reading OCI docker archive should not fail")
	require.NotNil(t, pkg, "parsed package data should not be nil")
	require.NotNil(t, info, "parsed docker info should not be nil")
	require.Equal(t, []string{"/docker-entrypoint"}, info.Config.Entrypoint, "docker entrypoint should match config")
	require.Equal(t, "/usr/share/testbeat", info.Config.WorkingDir, "docker working directory should match config")

	_, found := pkg.Contents["docker-entrypoint"]
	require.True(t, found, "entrypoint file should be present in extracted docker package contents")
	_, found = pkg.Contents["usr/share/testbeat/testbeat.yml"]
	require.True(t, found, "working directory files should be present in extracted docker package contents")
	_, found = pkg.Contents["usr/share/testbeat/LICENSE.txt"]
	require.True(t, found, "license files should be present in extracted docker package contents")
	_, found = pkg.Contents["etc/passwd"]
	require.False(t, found, "files outside working directory should not be included")
}

func TestReadDockerOCINestedIndexWithAttestation(t *testing.T) {
	configData := []byte(`{"config":{"Entrypoint":["/docker-entrypoint"],"Labels":{"org.label-schema.vendor":"Elastic"},"User":"root","WorkingDir":"/usr/share/testbeat"}}`)

	layerTar := createTestTarData(t, []testTarEntry{
		{name: "docker-entrypoint", mode: 0o755, data: []byte("#!/bin/sh\n")},
		{name: "usr/share/testbeat/testbeat.yml", mode: 0o644, data: []byte("name: testbeat\n")},
		{name: "usr/share/testbeat/LICENSE.txt", mode: 0o644, data: []byte("license\n")},
	})
	layerData := gzipTestData(t, layerTar)

	configDigest := sha256Digest(configData)
	layerDigest := sha256Digest(layerData)

	manifest := dockerOCIManifest{
		SchemaVersion: 2,
		MediaType:     dockerOCIManifestMediaType,
		Config: dockerOCIManifestDescriptor{
			MediaType: "application/vnd.oci.image.config.v1+json",
			Digest:    configDigest,
			Size:      int64(len(configData)),
		},
		Layers: []dockerOCIManifestDescriptor{
			{
				MediaType: "application/vnd.oci.image.layer.v1.tar+gzip",
				Digest:    layerDigest,
				Size:      int64(len(layerData)),
			},
		},
	}
	manifestData, err := json.Marshal(manifest)
	require.NoError(t, err, "OCI manifest marshaling should not fail")

	manifestDigest := sha256Digest(manifestData)
	attestationConfigData := []byte(`{"architecture":"unknown","os":"unknown","config":{},"rootfs":{"type":"layers","diff_ids":["sha256:133ae3f9bcc385295b66c2d83b28c25a9f294ce20954d5cf922dda860429734a"]}}`)
	attestationLayerData := []byte(`{"_type":"https://in-toto.io/Statement/v0.1","predicateType":"https://slsa.dev/provenance/v1","subject":[],"predicate":{}}`)
	attestationConfigDigest := sha256Digest(attestationConfigData)
	attestationLayerDigest := sha256Digest(attestationLayerData)

	attestationManifest := dockerOCIManifest{
		SchemaVersion: 2,
		MediaType:     dockerOCIManifestMediaType,
		Config: dockerOCIManifestDescriptor{
			MediaType: "application/vnd.oci.image.config.v1+json",
			Digest:    attestationConfigDigest,
			Size:      int64(len(attestationConfigData)),
		},
		Layers: []dockerOCIManifestDescriptor{
			{
				MediaType: "application/vnd.in-toto+json",
				Digest:    attestationLayerDigest,
				Size:      int64(len(attestationLayerData)),
			},
		},
	}
	attestationManifestData, err := json.Marshal(attestationManifest)
	require.NoError(t, err, "attestation manifest marshaling should not fail")

	attestationManifestDigest := sha256Digest(attestationManifestData)
	nestedIndex := dockerOCIIndex{
		SchemaVersion: 2,
		MediaType:     dockerOCIIndexMediaType,
		Manifests: []dockerOCIManifestDescriptor{
			{
				MediaType: dockerOCIManifestMediaType,
				Digest:    attestationManifestDigest,
				Size:      int64(len(attestationManifestData)),
				Annotations: map[string]string{
					"vnd.docker.reference.digest": manifestDigest,
					"vnd.docker.reference.type":   dockerOCIAttestationManifestType,
				},
				Platform: &dockerOCIPlatform{
					Architecture: "unknown",
					OS:           "unknown",
				},
			},
			{
				MediaType: dockerOCIManifestMediaType,
				Digest:    manifestDigest,
				Size:      int64(len(manifestData)),
				Platform: &dockerOCIPlatform{
					Architecture: "amd64",
					OS:           "linux",
				},
			},
		},
	}
	nestedIndexData, err := json.Marshal(nestedIndex)
	require.NoError(t, err, "nested OCI index marshaling should not fail")

	nestedIndexDigest := sha256Digest(nestedIndexData)
	index := dockerOCIIndex{
		SchemaVersion: 2,
		MediaType:     dockerOCIIndexMediaType,
		Manifests: []dockerOCIManifestDescriptor{
			{
				MediaType: dockerOCIIndexMediaType,
				Digest:    nestedIndexDigest,
				Size:      int64(len(nestedIndexData)),
				Annotations: map[string]string{
					"org.opencontainers.image.ref.name": "9.4.0-SNAPSHOT",
				},
			},
		},
	}
	indexData, err := json.Marshal(index)
	require.NoError(t, err, "top-level OCI index marshaling should not fail")

	nestedIndexPath, err := ociBlobPathFromDigest(nestedIndexDigest)
	require.NoError(t, err, "nested index digest should produce a valid OCI blob path")
	manifestPath, err := ociBlobPathFromDigest(manifestDigest)
	require.NoError(t, err, "manifest digest should produce a valid OCI blob path")
	configPath, err := ociBlobPathFromDigest(configDigest)
	require.NoError(t, err, "config digest should produce a valid OCI blob path")
	layerPath, err := ociBlobPathFromDigest(layerDigest)
	require.NoError(t, err, "layer digest should produce a valid OCI blob path")
	attestationManifestPath, err := ociBlobPathFromDigest(attestationManifestDigest)
	require.NoError(t, err, "attestation manifest digest should produce a valid OCI blob path")
	attestationConfigPath, err := ociBlobPathFromDigest(attestationConfigDigest)
	require.NoError(t, err, "attestation config digest should produce a valid OCI blob path")
	attestationLayerPath, err := ociBlobPathFromDigest(attestationLayerDigest)
	require.NoError(t, err, "attestation layer digest should produce a valid OCI blob path")

	dockerFile := createTestDockerArchive(t, []testTarEntry{
		{name: "oci-layout", mode: 0o644, data: []byte(`{"imageLayoutVersion":"1.0.0"}`)},
		{name: "index.json", mode: 0o644, data: indexData},
		{name: nestedIndexPath, mode: 0o644, data: nestedIndexData},
		{name: manifestPath, mode: 0o644, data: manifestData},
		{name: configPath, mode: 0o644, data: configData},
		{name: layerPath, mode: 0o644, data: layerData},
		{name: attestationManifestPath, mode: 0o644, data: attestationManifestData},
		{name: attestationConfigPath, mode: 0o644, data: attestationConfigData},
		{name: attestationLayerPath, mode: 0o644, data: attestationLayerData},
	})

	pkg, info, err := readDockerOCI(dockerFile)
	require.NoError(t, err, "reading OCI docker archive with nested index and attestation should not fail")
	require.NotNil(t, pkg, "parsed package data should not be nil")
	require.NotNil(t, info, "parsed docker info should not be nil")
	require.Equal(t, []string{"/docker-entrypoint"}, info.Config.Entrypoint, "docker entrypoint should match config")
	require.Equal(t, "/usr/share/testbeat", info.Config.WorkingDir, "docker working directory should match config")

	_, found := pkg.Contents["docker-entrypoint"]
	require.True(t, found, "entrypoint file should be present in extracted docker package contents")
	_, found = pkg.Contents["usr/share/testbeat/testbeat.yml"]
	require.True(t, found, "working directory files should be present in extracted docker package contents")
	_, found = pkg.Contents["usr/share/testbeat/LICENSE.txt"]
	require.True(t, found, "license files should be present in extracted docker package contents")
}

func TestReadDockerOCIMissingBlob(t *testing.T) {
	manifest := dockerOCIManifest{
		SchemaVersion: 2,
		MediaType:     dockerOCIManifestMediaType,
		Config: dockerOCIManifestDescriptor{
			MediaType: "application/vnd.oci.image.config.v1+json",
			Digest:    "sha256:5117abc6232b4c468263b488fa7cd5a5e07893a6dedad6b4de6ccfb2cafd0a45",
			Size:      1,
		},
		Layers: []dockerOCIManifestDescriptor{
			{
				MediaType: "application/vnd.oci.image.layer.v1.tar+gzip",
				Digest:    "sha256:75bed6ef625ff772ca48f63f12693f16f2b44649aa07030a7c4bc6b85225d5dd",
				Size:      1,
			},
		},
	}
	manifestData, err := json.Marshal(manifest)
	require.NoError(t, err, "OCI manifest marshaling should not fail")

	manifestDigest := sha256Digest(manifestData)
	index := dockerOCIIndex{
		SchemaVersion: 2,
		Manifests: []dockerOCIManifestDescriptor{
			{
				MediaType: dockerOCIManifestMediaType,
				Digest:    manifestDigest,
				Size:      int64(len(manifestData)),
			},
		},
	}
	indexData, err := json.Marshal(index)
	require.NoError(t, err, "OCI index marshaling should not fail")

	manifestPath, err := ociBlobPathFromDigest(manifestDigest)
	require.NoError(t, err, "manifest digest should produce a valid OCI blob path")

	dockerFile := createTestDockerArchive(t, []testTarEntry{
		{name: "oci-layout", mode: 0o644, data: []byte(`{"imageLayoutVersion":"1.0.0"}`)},
		{name: "index.json", mode: 0o644, data: indexData},
		{name: manifestPath, mode: 0o644, data: manifestData},
	})

	_, _, err = readDockerOCI(dockerFile)
	require.Error(t, err, "reading sparse OCI docker archive should fail")
	require.ErrorIs(t, err, errDockerArchiveEntryNotFound, "sparse OCI archive should report missing blob references")
}

// Sub-tests

func checkRPM(t *testing.T, file string) {
	p, _, err := readRPM(file)
	if err != nil {
		t.Error(err)
		return
	}

	checkConfigPermissions(t, p)
	checkConfigOwner(t, p, *rootOwner)
	checkManifestPermissions(t, p)
	checkManifestOwner(t, p, *rootOwner)
	checkModulesOwner(t, p, *rootOwner)
	checkModulesPermissions(t, p)
	checkModulesPresent(t, "/usr/share", p)
	checkModulesDPresent(t, "/etc/", p)
	checkMonitorsDPresent(t, "/etc", p)
	checkLicensesPresent(t, "/usr/share", p)
	checkSystemdUnitPermissions(t, p)
	ensureNoBuildIDLinks(t, p)
}

func checkDeb(t *testing.T, file string, buf *bytes.Buffer, fipsCheck bool) {
	p, err := readDeb(file, buf)
	if err != nil {
		t.Error(err)
		return
	}

	// deb file permissions are managed post-install
	checkConfigPermissions(t, p)
	checkConfigOwner(t, p, true)
	checkManifestPermissions(t, p)
	checkManifestOwner(t, p, true)
	checkModulesPresent(t, "./usr/share", p)
	checkModulesDPresent(t, "./etc/", p)
	checkMonitorsDPresent(t, "./etc/", p)
	checkLicensesPresent(t, "./usr/share", p)
	checkModulesOwner(t, p, true)
	checkModulesPermissions(t, p)
	checkSystemdUnitPermissions(t, p)
	if fipsCheck {
		t.Run(p.Name+"_fips_test", func(t *testing.T) {
			extractDir := t.TempDir()
			t.Logf("Extracting file %s into %s", file, extractDir)
			err := mage.Extract(file, extractDir)
			require.NoError(t, err, "Error extracting file %s", file)

			require.FileExists(t, filepath.Join(extractDir, "debian-binary"))
			require.FileExists(t, filepath.Join(extractDir, "control.tar.gz"))
			dataTarFile := filepath.Join(extractDir, "data.tar.gz")
			require.FileExists(t, dataTarFile)

			dataExtractionDir := filepath.Join(extractDir, "data")
			err = mage.Extract(dataTarFile, dataExtractionDir)
			require.NoError(t, err, "Error extracting data tarball")
			beatName := extractBeatNameFromTarName(t, filepath.Base(file))
			// the expected location for the binary is under /usr/share/<beatName>/bin
			containingDir := filepath.Join(dataExtractionDir, "usr", "share", beatName, "bin")
			checkFIPS(t, beatName, containingDir)
		})
	}
}

func checkTar(t *testing.T, file string, fipsCheck bool) {
	p, err := readTar(file)
	if err != nil {
		t.Error(err)
		return
	}

	checkConfigPermissions(t, p)
	checkConfigOwner(t, p, true)
	checkManifestPermissions(t, p)
	checkModulesPresent(t, "", p)
	checkModulesDPresent(t, "", p)
	checkModulesPermissions(t, p)
	checkModulesOwner(t, p, true)
	checkLicensesPresent(t, "", p)
	if fipsCheck {
		t.Run(p.Name+"_fips_test", func(t *testing.T) {
			extractDir := t.TempDir()
			t.Logf("Extracting file %s into %s", file, extractDir)
			err := mage.Extract(file, extractDir)
			require.NoError(t, err)
			containingDir := strings.TrimSuffix(filepath.Base(file), ".tar.gz")
			beatName := extractBeatNameFromTarName(t, filepath.Base(file))
			checkFIPS(t, beatName, filepath.Join(extractDir, containingDir))
		})
	}
}

func extractBeatNameFromTarName(t *testing.T, fileName string) string {
	// TODO check if cutting at the first '-' is an acceptable shortcut
	t.Logf("Extracting beat name from filename %s", fileName)
	const sep = "-"
	beatName, _, found := strings.Cut(fileName, sep)
	if !found {
		t.Logf("separator %s not found in filename %s: beatName may be incorrect", sep, fileName)
	}

	return beatName
}

func checkZip(t *testing.T, file string) {
	p, err := readZip(t, file, checkNpcapNotices)
	if err != nil {
		t.Error(err)
		return
	}

	checkConfigPermissions(t, p)
	checkManifestPermissions(t, p)
	checkModulesPresent(t, "", p)
	checkModulesDPresent(t, "", p)
	checkModulesPermissions(t, p)
	checkLicensesPresent(t, "", p)
}

const (
	npcapLicense    = `Dependency : Npcap \(https://nmap.org/npcap/\)`
	libpcapLicense  = `Dependency : Libpcap \(http://www.tcpdump.org/\)`
	winpcapLicense  = `Dependency : Winpcap \(https://www.winpcap.org/\)`
	radiotapLicense = `Dependency : ieee80211_radiotap.h Header File`
)

// This reflects the order that the licenses and notices appear in the relevant files.
var npcapLicensePattern = regexp.MustCompile(
	"(?s)" + npcapLicense +
		".*" + libpcapLicense +
		".*" + winpcapLicense +
		".*" + radiotapLicense,
)

func checkNpcapNotices(pkg, file string, contents io.Reader) error {
	if !strings.Contains(pkg, "packetbeat") {
		return nil
	}

	wantNotices := strings.Contains(pkg, "windows") && !strings.Contains(pkg, "oss")

	// If the packetbeat README.md is made to be generated
	// conditionally then it should also be checked here.
	pkg = filepath.Base(pkg)
	file, err := filepath.Rel(pkg[:len(pkg)-len(filepath.Ext(pkg))], file)
	if err != nil {
		return err
	}
	switch file {
	case "NOTICE.txt":
		if npcapLicensePattern.MatchReader(bufio.NewReader(contents)) != wantNotices {
			if wantNotices {
				return fmt.Errorf("Npcap license section not found in %s file in %s", file, pkg)
			}
			return fmt.Errorf("unexpected Npcap license section found in %s file in %s", file, pkg)
		}
	}
	return nil
}

func checkDocker(t *testing.T, file string) {
	imageType, err := detectDockerImageType(file)
	if err != nil {
		t.Errorf("error detecting docker image format for %v: %v", file, err)
		return
	}
	t.Logf("docker image format: %s", imageType)

	var p *packageFile
	var info *dockerInfo
	var daemonImageRef string
	switch imageType {
	case dockerImageTypeLegacy:
		p, info, err = readDocker(file)
		if err != nil {
			t.Errorf("error reading file %v: %v", file, err)
			return
		}
	case dockerImageTypeOCI:
		p, info, err = readDockerOCI(file)
		if err != nil && errors.Is(err, errDockerArchiveEntryNotFound) {
			t.Logf("OCI archive is sparse, hydrating checks from daemon image: %v", err)
			p, info, daemonImageRef, err = readDockerOCIFromDaemon(t, file)
		}
		if err != nil {
			t.Errorf("error reading file %v: %v", file, err)
			return
		}
	default:
		t.Errorf("unsupported docker image format %q for %s", imageType, file)
		return
	}

	checkDockerEntryPoint(t, p, info)
	checkDockerLabels(t, p, info, file)
	checkDockerUser(t, p, info, *rootUserContainer)
	checkConfigPermissionsWithMode(t, p, os.FileMode(0o644))
	checkManifestPermissionsWithMode(t, p, os.FileMode(0o644))
	checkModulesPresent(t, "", p)
	checkModulesDPresent(t, "", p)
	checkLicensesPresent(t, "licenses/", p)
	checkDockerImageRun(t, p, file, daemonImageRef)
}

// Verify that the main configuration file is installed with a 0600 file mode.
func checkConfigPermissions(t *testing.T, p *packageFile) {
	checkConfigPermissionsWithMode(t, p, expectedConfigMode)
}

func checkConfigPermissionsWithMode(t *testing.T, p *packageFile, expectedMode os.FileMode) {
	t.Run(p.Name+" config file permissions", func(t *testing.T) {
		for _, entry := range p.Contents {
			if configFilePattern.MatchString(entry.File) {
				mode := entry.Mode.Perm()
				if expectedMode != mode {
					t.Errorf("file %v has wrong permissions: expected=%v actual=%v",
						entry.File, expectedMode, mode)
				}
				return
			}
		}
		t.Logf("no config file found matching %v", configFilePattern)
	})
}

func checkOwner(t *testing.T, entry packageEntry, expectRoot bool) {
	should := "not "
	if expectRoot {
		should = ""
	}
	if expectRoot != (entry.UID == 0) {
		t.Errorf("file %v should %sbe owned by root user, owner=%v", entry.File, should, entry.UID)
	}
	if expectRoot != (entry.GID == 0) {
		t.Errorf("file %v should %sbe owned by root group, group=%v", entry.File, should, entry.GID)
	}
}

func checkConfigOwner(t *testing.T, p *packageFile, expectRoot bool) {
	t.Run(p.Name+" config file owner", func(t *testing.T) {
		for _, entry := range p.Contents {
			if configFilePattern.MatchString(entry.File) {
				checkOwner(t, entry, expectRoot)
				return
			}
		}
		t.Logf("no config file found matching %v", configFilePattern)
	})
}

// Verify that the modules manifest.yml files are installed with a 0644 file mode.
func checkManifestPermissions(t *testing.T, p *packageFile) {
	checkManifestPermissionsWithMode(t, p, expectedManifestMode)
}

func checkManifestPermissionsWithMode(t *testing.T, p *packageFile, expectedMode os.FileMode) {
	t.Run(p.Name+" manifest file permissions", func(t *testing.T) {
		for _, entry := range p.Contents {
			if manifestFilePattern.MatchString(entry.File) {
				mode := entry.Mode.Perm()
				if expectedMode != mode {
					t.Errorf("file %v has wrong permissions: expected=%v actual=%v",
						entry.File, expectedMode, mode)
				}
			}
		}
	})
}

// Verify that the manifest owner is correct.
func checkManifestOwner(t *testing.T, p *packageFile, expectRoot bool) {
	t.Run(p.Name+" manifest file owner", func(t *testing.T) {
		for _, entry := range p.Contents {
			if manifestFilePattern.MatchString(entry.File) {
				checkOwner(t, entry, expectRoot)
			}
		}
	})
}

// Verify the permissions of the modules.d dir and its contents.
func checkModulesPermissions(t *testing.T, p *packageFile) {
	t.Run(p.Name+" modules.d file permissions", func(t *testing.T) {
		for _, entry := range p.Contents {
			if modulesDFilePattern.MatchString(entry.File) {
				mode := entry.Mode.Perm()
				if expectedModuleFileMode != mode {
					t.Errorf("file %v has wrong permissions: expected=%v actual=%v",
						entry.File, expectedModuleFileMode, mode)
				}
			} else if modulesDDirPattern.MatchString(entry.File) {
				mode := entry.Mode.Perm()
				if expectedModuleDirMode != mode {
					t.Errorf("file %v has wrong permissions: expected=%v actual=%v",
						entry.File, expectedModuleDirMode, mode)
				}
			}
		}
	})
}

// Verify the owner of the modules.d dir and its contents.
func checkModulesOwner(t *testing.T, p *packageFile, expectRoot bool) {
	t.Run(p.Name+" modules.d file owner", func(t *testing.T) {
		for _, entry := range p.Contents {
			if modulesDFilePattern.MatchString(entry.File) || modulesDDirPattern.MatchString(entry.File) {
				checkOwner(t, entry, expectRoot)
			}
		}
	})
}

// Verify that the systemd unit file has a mode of 0644. It should not be
// executable.
func checkSystemdUnitPermissions(t *testing.T, p *packageFile) {
	const expectedMode = os.FileMode(0o644)
	t.Run(p.Name+" systemd unit file permissions", func(t *testing.T) {
		for _, entry := range p.Contents {
			if systemdUnitFilePattern.MatchString(entry.File) {
				mode := entry.Mode.Perm()
				if expectedMode != mode {
					t.Errorf("file %v has wrong permissions: expected=%v actual=%v",
						entry.File, expectedMode, mode)
				}
				return
			}
		}
		t.Errorf("no systemd unit file found matching %v", configFilePattern)
	})
}

// Verify that modules folder is present and has module files in
func checkModulesPresent(t *testing.T, prefix string, p *packageFile) {
	if *modules {
		checkModules(t, "modules", prefix, modulesDirPattern, p)
	}
}

// Verify that modules.d folder is present and has module files in
func checkModulesDPresent(t *testing.T, prefix string, p *packageFile) {
	if *modulesd {
		checkModules(t, "modules.d", prefix, modulesDFilePattern, p)
	}
}

func checkMonitorsDPresent(t *testing.T, prefix string, p *packageFile) {
	if *monitorsd {
		checkMonitors(t, "monitors.d", prefix, monitorsDFilePattern, p)
	}
}

func checkModules(t *testing.T, name, prefix string, r *regexp.Regexp, p *packageFile) {
	t.Run(fmt.Sprintf("%s %s contents", p.Name, name), func(t *testing.T) {
		minExpectedModules := *minModules
		total := 0
		for _, entry := range p.Contents {
			if strings.HasPrefix(entry.File, prefix) && r.MatchString(entry.File) {
				total++
			}
		}

		if total < minExpectedModules {
			t.Errorf("not enough modules found under %s: actual=%d, expected>=%d",
				name, total, minExpectedModules)
		}
	})
}

func checkMonitors(t *testing.T, name, prefix string, r *regexp.Regexp, p *packageFile) {
	t.Run(fmt.Sprintf("%s %s contents", p.Name, name), func(t *testing.T) {
		minExpectedModules := 1
		total := 0
		for _, entry := range p.Contents {
			if strings.HasPrefix(entry.File, prefix) && r.MatchString(entry.File) {
				total++
			}
		}

		if total < minExpectedModules {
			t.Errorf("not enough monitors found under %s: actual=%d, expected>=%d",
				name, total, minExpectedModules)
		}
	})
}

func checkLicensesPresent(t *testing.T, prefix string, p *packageFile) {
	for _, licenseFile := range licenseFiles {
		t.Run("License file "+licenseFile, func(t *testing.T) {
			for _, entry := range p.Contents {
				if strings.HasPrefix(entry.File, prefix) && strings.HasSuffix(entry.File, "/"+licenseFile) {
					return
				}
			}
			if prefix != "" {
				t.Fatalf("not found under %s", prefix)
			}
			t.Fatal("not found")
		})
	}
}

func checkDockerEntryPoint(t *testing.T, p *packageFile, info *dockerInfo) {
	expectedMode := os.FileMode(0o755)

	t.Run(fmt.Sprintf("%s entrypoint", p.Name), func(t *testing.T) {
		if len(info.Config.Entrypoint) == 0 {
			t.Fatal("no entrypoint")
		}

		entrypoint := info.Config.Entrypoint[0]
		if entrypoint, ok := strings.CutPrefix(entrypoint, "/"); ok {
			entry, found := p.Contents[entrypoint]
			if !found {
				t.Fatalf("%s entrypoint not found in docker", entrypoint)
			}
			if mode := entry.Mode.Perm(); mode != expectedMode {
				t.Fatalf("%s entrypoint mode is %s, expected: %s", entrypoint, mode, expectedMode)
			}
		} else {
			t.Fatal("TODO: check if binary is in $PATH")
		}
	})
}

// {BeatName}-oss-{OptionalVariantSuffix}-{version}-{os}-{arch}.docker.tar.gz
// For example, `heartbeat-oss-8.16.0-linux-arm64.docker.tar.gz`
var ossSuffixRegexp = regexp.MustCompile(`^(\w+)-oss-.+$`)

func checkDockerLabels(t *testing.T, p *packageFile, info *dockerInfo, file string) {
	vendor := info.Config.Labels["org.label-schema.vendor"]
	if vendor != "Elastic" {
		return
	}

	t.Run(fmt.Sprintf("%s license labels", p.Name), func(t *testing.T) {
		expectedLicense := "Elastic License"
		if ossSuffixRegexp.MatchString(filepath.Base(file)) {
			expectedLicense = "ASL 2.0"
		}
		licenseLabels := []string{
			"license",
			"org.label-schema.license",
		}
		for _, licenseLabel := range licenseLabels {
			if license, present := info.Config.Labels[licenseLabel]; !present || license != expectedLicense {
				t.Errorf("unexpected license label %s: %s", licenseLabel, license)
			}
		}
	})

	t.Run(fmt.Sprintf("%s required labels", p.Name), func(t *testing.T) {
		// From https://redhat-connect.gitbook.io/partner-guide-for-red-hat-openshift-and-container/program-on-boarding/technical-prerequisites
		requiredLabels := []string{"name", "vendor", "version", "release", "summary", "description"}
		for _, label := range requiredLabels {
			if value, present := info.Config.Labels[label]; !present || value == "" {
				t.Errorf("missing required label %s", label)
			}
		}
	})
}

func checkDockerUser(t *testing.T, p *packageFile, info *dockerInfo, expectRoot bool) {
	t.Run(fmt.Sprintf("%s user", p.Name), func(t *testing.T) {
		if expectRoot != (info.Config.User == "root") {
			t.Errorf("unexpected docker user: %s", info.Config.User)
		}
	})
}

func checkDockerImageRun(t *testing.T, p *packageFile, imagePath, imageRef string) {
	t.Run(fmt.Sprintf("%s check docker images runs", p.Name), func(t *testing.T) {
		ctx, cancel := dockerTestContext(t)
		defer cancel()

		dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
		if err != nil {
			t.Fatalf("failed to get a Docker client: %s", err)
		}

		imageID := imageRef
		if imageID == "" {
			imageID, err = loadDockerImageFromArchive(ctx, dockerClient, imagePath)
			if err != nil {
				t.Fatalf("error loading docker image: %s", err)
			}
		} else {
			_, err = dockerClient.ImageInspect(ctx, imageID)
			if err != nil {
				t.Fatalf("error inspecting docker image %q from daemon: %s", imageID, err)
			}
		}

		var caps strslice.StrSlice
		if strings.Contains(imageID, "packetbeat") {
			caps = append(caps, "NET_ADMIN")
		}

		createResp, err := dockerClient.ContainerCreate(ctx,
			&container.Config{
				Image: imageID,
			},
			&container.HostConfig{
				CapAdd: caps,
			},
			nil,
			nil,
			"")
		if err != nil {
			t.Fatalf("error creating container from image: %s", err)
		}
		defer func() {
			err := dockerClient.ContainerRemove(ctx, createResp.ID, container.RemoveOptions{Force: true})
			if err != nil {
				t.Errorf("error removing container: %s", err)
			}
		}()

		err = dockerClient.ContainerStart(ctx, createResp.ID, container.StartOptions{})
		if err != nil {
			t.Fatalf("failed to start container: %s", err)
		}
		defer func() {
			err := dockerClient.ContainerStop(ctx, createResp.ID, container.StopOptions{})
			if err != nil {
				t.Errorf("error stopping container: %s", err)
			}
		}()

		timer := time.NewTimer(15 * time.Second)
		defer timer.Stop()
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		var logs []byte
		sentinelLog := "Beat ID: "
		for {
			select {
			case <-timer.C:
				t.Fatalf("never saw %q within timeout\nlogs:\n%s", sentinelLog, string(logs))
				return
			case <-ticker.C:
				out, err := dockerClient.ContainerLogs(ctx, createResp.ID, container.LogsOptions{ShowStdout: true, ShowStderr: true})
				if err != nil {
					t.Logf("could not get logs: %s", err)
				}
				logs, err = io.ReadAll(out)
				out.Close()
				if err != nil {
					t.Logf("error reading logs: %s", err)
				}
				if bytes.Contains(logs, []byte(sentinelLog)) {
					return
				}
			}
		}
	})
}

func dockerTestContext(t *testing.T) (context.Context, context.CancelFunc) {
	t.Helper()

	deadline, ok := t.Deadline()
	if !ok {
		return context.Background(), func() {}
	}

	return context.WithDeadline(context.Background(), deadline)
}

func loadDockerImageFromArchive(ctx context.Context, dockerClient *client.Client, imagePath string) (string, error) {
	f, err := os.Open(imagePath)
	if err != nil {
		return "", fmt.Errorf("failed to open docker image %q: %w", imagePath, err)
	}
	defer f.Close()

	loadResp, err := dockerClient.ImageLoad(ctx, f, client.ImageLoadWithQuiet(true))
	if err != nil {
		return "", err
	}
	defer loadResp.Body.Close()

	loadRespBody, err := io.ReadAll(loadResp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read image load response: %w", err)
	}

	imageID, err := parseLoadedImageRef(string(loadRespBody))
	if err != nil {
		return "", err
	}
	return imageID, nil
}

func parseLoadedImageRef(loadResponse string) (string, error) {
	for _, prefix := range []string{"Loaded image: ", "Loaded image ID: "} {
		_, after, ok := strings.Cut(loadResponse, prefix)
		if !ok {
			continue
		}

		end := len(after)
		for i, r := range after {
			if r == '\n' || r == '\r' || r == '"' || r == '\\' {
				end = i
				break
			}
		}

		imageID := strings.TrimSpace(after[:end])
		if imageID != "" {
			return imageID, nil
		}
	}

	return "", fmt.Errorf("image load response was unexpected: %s", loadResponse)
}

func readDockerOCIFromDaemon(t *testing.T, dockerFile string) (*packageFile, *dockerInfo, string, error) {
	t.Helper()

	imageRef, err := dockerImageRefFromArchive(dockerFile)
	if err != nil {
		return nil, nil, "", err
	}

	ctx, cancel := dockerTestContext(t)
	defer cancel()

	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to get a Docker client: %w", err)
	}

	inspectResp, err := dockerClient.ImageInspect(ctx, imageRef)
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed inspecting docker image %q from daemon: %w", imageRef, err)
	}
	if inspectResp.Config == nil {
		return nil, nil, "", fmt.Errorf("docker image %q from daemon has no config", imageRef)
	}

	info := &dockerInfo{}
	info.Config.Entrypoint = append(info.Config.Entrypoint, inspectResp.Config.Entrypoint...)
	info.Config.User = inspectResp.Config.User
	info.Config.WorkingDir = inspectResp.Config.WorkingDir
	info.Config.Labels = make(map[string]string, len(inspectResp.Config.Labels))
	maps.Copy(info.Config.Labels, inspectResp.Config.Labels)

	createResp, err := dockerClient.ContainerCreate(ctx, &container.Config{Image: imageRef}, nil, nil, nil, "")
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed creating container from image %q: %w", imageRef, err)
	}
	defer func() {
		_ = dockerClient.ContainerRemove(ctx, createResp.ID, container.RemoveOptions{Force: true})
	}()

	exportedFilesystem, err := dockerClient.ContainerExport(ctx, createResp.ID)
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed exporting filesystem from image %q: %w", imageRef, err)
	}
	defer exportedFilesystem.Close()

	rootFS, err := readTarContents(filepath.Base(dockerFile), exportedFilesystem)
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed reading exported docker filesystem for %q: %w", imageRef, err)
	}

	pkg, err := buildDockerPackageFile(dockerFile, info, []*packageFile{rootFS})
	if err != nil {
		return nil, nil, "", err
	}

	return pkg, info, imageRef, nil
}

func dockerImageRefFromArchive(dockerFile string) (string, error) {
	manifest, err := readManifest(dockerFile)
	if err != nil {
		return "", fmt.Errorf("failed reading docker manifest for image reference: %w", err)
	}
	for _, repoTag := range manifest.RepoTags {
		if repoTag != "" {
			return repoTag, nil
		}
	}

	return "", fmt.Errorf("manifest.json has no repo tags for %s", dockerFile)
}

// ensureNoBuildIDLinks checks for regressions related to
// https://github.com/elastic/beats/issues/12956.
func ensureNoBuildIDLinks(t *testing.T, p *packageFile) {
	t.Run(fmt.Sprintf("%s no build_id links", p.Name), func(t *testing.T) {
		for name := range p.Contents {
			if strings.Contains(name, "/usr/lib/.build-id") {
				t.Error("found unexpected /usr/lib/.build-id in package")
			}
		}
	})
}

// Helpers

type packageFile struct {
	Name     string
	Contents map[string]packageEntry
}

type packageEntry struct {
	File string
	UID  int
	GID  int
	Mode os.FileMode
}

func getFiles(t *testing.T, pattern *regexp.Regexp) []string {
	matches, err := filepath.Glob(*files)
	if err != nil {
		t.Fatal(err)
	}

	files := matches[:0]
	for _, f := range matches {
		if pattern.MatchString(filepath.Base(f)) {
			files = append(files, f)
		}
	}
	return files
}

func readRPM(rpmFile string) (*packageFile, *rpm.Package, error) {
	p, err := rpm.Open(rpmFile)
	if err != nil {
		return nil, nil, err
	}

	contents := p.Files()
	pf := &packageFile{Name: filepath.Base(rpmFile), Contents: map[string]packageEntry{}}

	for _, file := range contents {
		if excludedPathsPattern.MatchString(file.Name()) {
			continue
		}
		pe := packageEntry{
			File: file.Name(),
			Mode: file.Mode(),
		}
		if file.Owner() != "root" {
			// not 0
			pe.UID = 123
			pe.GID = 123
		}
		pf.Contents[file.Name()] = pe
	}

	return pf, p, nil
}

// readDeb reads the data.tar.gz file from the .deb.
func readDeb(debFile string, dataBuffer *bytes.Buffer) (*packageFile, error) {
	file, err := os.Open(debFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	arReader := ar.NewReader(file)
	for {
		header, err := arReader.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}

		if strings.HasPrefix(header.Name, "data.tar.gz") {
			dataBuffer.Reset()
			_, err := io.Copy(dataBuffer, arReader)
			if err != nil {
				return nil, err
			}

			gz, err := gzip.NewReader(dataBuffer)
			if err != nil {
				return nil, err
			}
			defer gz.Close()

			return readTarContents(filepath.Base(debFile), gz)
		}
	}

	return nil, io.EOF
}

func readTar(tarFile string) (*packageFile, error) {
	file, err := os.Open(tarFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var fileReader io.ReadCloser = file
	if strings.HasSuffix(tarFile, ".gz") {
		if fileReader, err = gzip.NewReader(file); err != nil {
			return nil, err
		}
		defer fileReader.Close()
	}

	return readTarContents(filepath.Base(tarFile), fileReader)
}

func readTarContents(tarName string, data io.Reader) (*packageFile, error) {
	tarReader := tar.NewReader(data)

	p := &packageFile{Name: tarName, Contents: map[string]packageEntry{}}
	for {
		header, err := tarReader.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}

		if excludedPathsPattern.MatchString(header.Name) {
			continue
		}

		p.Contents[header.Name] = packageEntry{
			File: header.Name,
			UID:  header.Uid,
			GID:  header.Gid,
			Mode: os.FileMode(header.Mode), //nolint:gosec // G115 Conversion from int to uint32 is safe here.
		}
	}

	return p, nil
}

func checkFIPS(t *testing.T, beatName, path string) {
	t.Logf("Checking %s for FIPS compliance", beatName)
	binaryPath := filepath.Join(path, beatName) // TODO eventually we'll need to support checking a .exe
	require.FileExistsf(t, binaryPath, "Unable to find beat executable %s", binaryPath)

	info, err := buildinfo.ReadFile(binaryPath)
	require.NoError(t, err)

	foundTags := false
	foundExperiment := false
	for _, setting := range info.Settings {
		switch setting.Key {
		case "-tags":
			foundTags = true
			require.Contains(t, setting.Value, "requirefips")
			continue
		case "GOEXPERIMENT":
			foundExperiment = true
			require.Contains(t, setting.Value, "systemcrypto")
			continue
		}
	}

	require.True(t, foundTags, "Did not find -tags within binary version information")
	require.True(t, foundExperiment, "Did not find GOEXPERIMENT within binary version information")

	// TODO only elf is supported at the moment, in the future we will need to use macho (darwin) and pe (windows)
	f, err := elf.Open(binaryPath)
	require.NoError(t, err, "unable to open ELF file")

	symbols, err := f.Symbols()
	if err != nil {
		t.Logf("no symbols present in %q: %v", binaryPath, err)
		return
	}

	hasOpenSSL := false
	for _, symbol := range symbols {
		if strings.Contains(symbol.Name, "OpenSSL_version") {
			hasOpenSSL = true
			break
		}
	}
	require.True(t, hasOpenSSL, "unable to find OpenSSL_version symbol")
}

// inspector is a file contents inspector. It vets the contents of the file
// within a package for a requirement and returns an error if it is not met.
type inspector func(pkg, file string, contents io.Reader) error

func readZip(t *testing.T, zipFile string, inspectors ...inspector) (*packageFile, error) {
	r, err := zip.OpenReader(zipFile)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	p := &packageFile{Name: filepath.Base(zipFile), Contents: map[string]packageEntry{}}
	for _, f := range r.File {
		if excludedPathsPattern.MatchString(f.Name) {
			continue
		}
		p.Contents[f.Name] = packageEntry{
			File: f.Name,
			Mode: f.Mode(),
		}
		for _, inspect := range inspectors {
			r, err := f.Open()
			if err != nil {
				t.Errorf("failed to open %s in %s: %v", f.Name, zipFile, err)
				break
			}
			err = inspect(zipFile, f.Name, r)
			if err != nil {
				t.Error(err)
			}
			r.Close()
		}
	}

	return p, nil
}

func normalizeDockerArchivePath(name string) string {
	return strings.TrimPrefix(name, "./")
}

func walkDockerArchive(dockerFile string, onEntry func(header *tar.Header, r io.Reader) error) error {
	file, err := os.Open(dockerFile)
	if err != nil {
		return err
	}
	defer file.Close()

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)
	for {
		header, err := tarReader.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}

		err = onEntry(header, tarReader)
		if err != nil {
			if errors.Is(err, errDockerArchiveWalkDone) {
				return nil
			}
			return err
		}
	}
}

func detectDockerImageType(dockerFile string) (dockerImageType, error) {
	var legacyFormat bool
	var ociFormat bool

	err := walkDockerArchive(dockerFile, func(header *tar.Header, _ io.Reader) error {
		entryName := normalizeDockerArchivePath(header.Name)
		switch entryName {
		case "manifest.json":
			legacyFormat = true
		case "index.json", "oci-layout":
			ociFormat = true
			return errDockerArchiveWalkDone
		}
		return nil
	})
	if err != nil {
		return "", err
	}

	switch {
	case ociFormat:
		return dockerImageTypeOCI, nil
	case legacyFormat:
		return dockerImageTypeLegacy, nil
	default:
		return "", fmt.Errorf("unable to determine docker archive format for %s", dockerFile)
	}
}

func readDockerArchiveEntry(dockerFile, entryName string) ([]byte, error) {
	target := normalizeDockerArchivePath(entryName)
	var data []byte
	var found bool

	err := walkDockerArchive(dockerFile, func(header *tar.Header, r io.Reader) error {
		if normalizeDockerArchivePath(header.Name) != target {
			return nil
		}

		var err error
		data, err = io.ReadAll(r)
		if err != nil {
			return fmt.Errorf("failed reading docker archive entry %q: %w", target, err)
		}
		found = true
		return errDockerArchiveWalkDone
	})
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("%w: %q", errDockerArchiveEntryNotFound, target)
	}

	return data, nil
}

func readDocker(dockerFile string) (*packageFile, *dockerInfo, error) {
	manifest, err := readManifest(dockerFile)
	if err != nil {
		return nil, nil, err
	}

	layerNames := make([]string, 0, len(manifest.Layers))
	for _, layer := range manifest.Layers {
		layerNames = append(layerNames, normalizeDockerArchivePath(layer))
	}

	configName := normalizeDockerArchivePath(manifest.Config)
	layers := make(map[string]*packageFile, len(layerNames))
	var info *dockerInfo

	err = walkDockerArchive(dockerFile, func(header *tar.Header, r io.Reader) error {
		entryName := normalizeDockerArchivePath(header.Name)
		switch {
		case entryName == configName:
			info, err = readDockerInfo(r)
			if err != nil {
				return fmt.Errorf("failed to read docker config %q: %w", entryName, err)
			}
		case slices.Contains(layerNames, entryName):
			layer, err := readTarContents(entryName, r)
			if err != nil {
				return fmt.Errorf("failed to read docker layer %q: %w", entryName, err)
			}
			layers[entryName] = layer
		}

		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	if info == nil {
		return nil, nil, fmt.Errorf("docker config %q not found", configName)
	}

	orderedLayers := make([]*packageFile, 0, len(layerNames))
	for _, layerName := range layerNames {
		layer, found := layers[layerName]
		if !found {
			return nil, nil, fmt.Errorf("docker layer %q not found", layerName)
		}
		orderedLayers = append(orderedLayers, layer)
	}

	p, err := buildDockerPackageFile(dockerFile, info, orderedLayers)
	if err != nil {
		return nil, nil, err
	}

	return p, info, nil
}

func readDockerOCI(dockerFile string) (*packageFile, *dockerInfo, error) {
	indexData, err := readDockerArchiveEntry(dockerFile, "index.json")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read OCI index: %w", err)
	}

	index, err := readDockerOCIIndex(bytes.NewReader(indexData))
	if err != nil {
		return nil, nil, err
	}

	manifest, err := resolveDockerOCIManifestFromIndex(dockerFile, index, map[string]struct{}{})
	if err != nil {
		return nil, nil, err
	}

	configPath, err := ociBlobPathFromDigest(manifest.Config.Digest)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid OCI config digest %q: %w", manifest.Config.Digest, err)
	}

	layerPaths := make([]string, len(manifest.Layers))
	layerIndexes := make(map[string]int, len(manifest.Layers))
	for i, layer := range manifest.Layers {
		layerPath, err := ociBlobPathFromDigest(layer.Digest)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid OCI layer digest %q: %w", layer.Digest, err)
		}
		layerPaths[i] = layerPath
		layerIndexes[layerPath] = i
	}

	layers := make([]*packageFile, len(manifest.Layers))
	var info *dockerInfo
	err = walkDockerArchive(dockerFile, func(header *tar.Header, r io.Reader) error {
		entryName := normalizeDockerArchivePath(header.Name)
		switch entryName {
		case configPath:
			info, err = readDockerInfo(r)
			if err != nil {
				return fmt.Errorf("failed to read OCI docker config %q: %w", entryName, err)
			}
		default:
			index, found := layerIndexes[entryName]
			if !found {
				return nil
			}

			layer, err := readDockerLayerContents(entryName, manifest.Layers[index], r)
			if err != nil {
				return err
			}
			layers[index] = layer
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	if info == nil {
		return nil, nil, fmt.Errorf("%w: %q", errDockerArchiveEntryNotFound, configPath)
	}
	for i, layer := range layers {
		if layer == nil {
			return nil, nil, fmt.Errorf("%w: %q", errDockerArchiveEntryNotFound, layerPaths[i])
		}
	}

	p, err := buildDockerPackageFile(dockerFile, info, layers)
	if err != nil {
		return nil, nil, err
	}

	return p, info, nil
}

func readDockerLayerContents(layerName string, descriptor dockerOCIManifestDescriptor, r io.Reader) (*packageFile, error) {
	layerData := r
	if strings.Contains(strings.ToLower(descriptor.MediaType), "gzip") {
		gzipLayer, err := gzip.NewReader(r)
		if err != nil {
			return nil, fmt.Errorf("failed to open gzip docker layer %q: %w", layerName, err)
		}
		defer gzipLayer.Close()
		layerData = gzipLayer
	}

	layer, err := readTarContents(layerName, layerData)
	if err != nil {
		return nil, fmt.Errorf("failed reading docker layer %q: %w", layerName, err)
	}
	return layer, nil
}

func buildDockerPackageFile(dockerFile string, info *dockerInfo, layers []*packageFile) (*packageFile, error) {
	if info == nil {
		return nil, errors.New("docker info cannot be nil")
	}
	if len(info.Config.Entrypoint) == 0 {
		return nil, fmt.Errorf("no entrypoint")
	}

	workingDir := info.Config.WorkingDir
	entrypoint := info.Config.Entrypoint[0]

	// Read layers in order and for each file keep only the entry seen in the later layer.
	p := &packageFile{Name: filepath.Base(dockerFile), Contents: map[string]packageEntry{}}
	for _, layerFile := range layers {
		for name, entry := range layerFile.Contents {
			// Check only files in working dir and entrypoint.
			if strings.HasPrefix("/"+name, workingDir) || "/"+name == entrypoint {
				p.Contents[name] = entry
			}
			if excludedPathsPattern.MatchString(name) {
				continue
			}
			// Add licenses regardless of path.
			for _, licenseFile := range licenseFiles {
				if strings.Contains(name, licenseFile) {
					p.Contents[name] = entry
				}
			}
		}
	}

	if len(p.Contents) == 0 {
		return nil, fmt.Errorf("no files found in docker working directory (%s)", info.Config.WorkingDir)
	}

	return p, nil
}

func readManifest(dockerFile string) (*dockerManifest, error) {
	var manifest *dockerManifest
	err := walkDockerArchive(dockerFile, func(header *tar.Header, r io.Reader) error {
		if normalizeDockerArchivePath(header.Name) != "manifest.json" {
			return nil
		}

		var err error
		manifest, err = readDockerManifest(r)
		if err != nil {
			return err
		}
		return errDockerArchiveWalkDone
	})
	if err != nil {
		return nil, err
	}
	if manifest == nil {
		return nil, fmt.Errorf("manifest.json not found in docker archive %s", dockerFile)
	}
	return manifest, nil
}

type dockerManifest struct {
	Config   string
	RepoTags []string
	Layers   []string
}

func readDockerManifest(r io.Reader) (*dockerManifest, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	var manifests []*dockerManifest
	err = json.Unmarshal(data, &manifests)
	if err != nil {
		return nil, err
	}

	if len(manifests) != 1 {
		return nil, fmt.Errorf("one and only one manifest expected, %d found", len(manifests))
	}

	return manifests[0], nil
}

const (
	dockerOCIManifestMediaType                = "application/vnd.oci.image.manifest.v1+json"
	dockerOCIIndexMediaType                   = "application/vnd.oci.image.index.v1+json"
	dockerDistributionV2ManifestMediaType     = "application/vnd.docker.distribution.manifest.v2+json"
	dockerDistributionV2ManifestListMediaType = "application/vnd.docker.distribution.manifest.list.v2+json"
	dockerOCIAttestationManifestType          = "attestation-manifest"
)

type dockerOCIIndex struct {
	SchemaVersion int                           `json:"schemaVersion"`
	MediaType     string                        `json:"mediaType,omitempty"`
	Manifests     []dockerOCIManifestDescriptor `json:"manifests"`
}

type dockerOCIManifest struct {
	SchemaVersion int                           `json:"schemaVersion"`
	MediaType     string                        `json:"mediaType,omitempty"`
	Config        dockerOCIManifestDescriptor   `json:"config"`
	Layers        []dockerOCIManifestDescriptor `json:"layers"`
}

type dockerOCIManifestDescriptor struct {
	MediaType   string             `json:"mediaType,omitempty"`
	Digest      string             `json:"digest"`
	Size        int64              `json:"size,omitempty"`
	Annotations map[string]string  `json:"annotations,omitempty"`
	Platform    *dockerOCIPlatform `json:"platform,omitempty"`
}

type dockerOCIPlatform struct {
	Architecture string `json:"architecture,omitempty"`
	OS           string `json:"os,omitempty"`
}

func readDockerOCIIndex(r io.Reader) (*dockerOCIIndex, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	var index dockerOCIIndex
	err = json.Unmarshal(data, &index)
	if err != nil {
		return nil, fmt.Errorf("failed to decode OCI index: %w", err)
	}
	if len(index.Manifests) == 0 {
		return nil, fmt.Errorf("no manifests found in OCI index")
	}

	return &index, nil
}

func readDockerOCIManifest(r io.Reader) (*dockerOCIManifest, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	var manifest dockerOCIManifest
	err = json.Unmarshal(data, &manifest)
	if err != nil {
		return nil, fmt.Errorf("failed to decode OCI manifest: %w", err)
	}
	if manifest.Config.Digest == "" {
		return nil, fmt.Errorf("OCI manifest is missing config digest")
	}
	if len(manifest.Layers) == 0 {
		return nil, fmt.Errorf("OCI manifest has no layers")
	}

	return &manifest, nil
}

var errDockerOCIDescriptorSkipped = errors.New("docker OCI descriptor skipped")

func resolveDockerOCIManifestFromIndex(dockerFile string, index *dockerOCIIndex, visited map[string]struct{}) (*dockerOCIManifest, error) {
	for _, descriptor := range index.Manifests {
		manifest, err := resolveDockerOCIManifestFromDescriptor(dockerFile, descriptor, visited)
		if err == nil {
			return manifest, nil
		}
		if errors.Is(err, errDockerOCIDescriptorSkipped) {
			continue
		}
		return nil, err
	}

	return nil, fmt.Errorf("OCI index does not contain a runnable manifest descriptor")
}

func resolveDockerOCIManifestFromDescriptor(dockerFile string, descriptor dockerOCIManifestDescriptor, visited map[string]struct{}) (*dockerOCIManifest, error) {
	if descriptor.Digest == "" {
		return nil, fmt.Errorf("%w: descriptor is missing digest", errDockerOCIDescriptorSkipped)
	}
	if isDockerOCIAttestationDescriptor(descriptor) {
		return nil, fmt.Errorf("%w: descriptor %q is an attestation manifest", errDockerOCIDescriptorSkipped, descriptor.Digest)
	}
	if _, found := visited[descriptor.Digest]; found {
		return nil, fmt.Errorf("OCI descriptor recursion detected at %q", descriptor.Digest)
	}

	visited[descriptor.Digest] = struct{}{}
	defer delete(visited, descriptor.Digest)

	descriptorPath, err := ociBlobPathFromDigest(descriptor.Digest)
	if err != nil {
		return nil, fmt.Errorf("invalid OCI descriptor digest %q: %w", descriptor.Digest, err)
	}

	descriptorData, err := readDockerArchiveEntry(dockerFile, descriptorPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read OCI descriptor %q: %w", descriptorPath, err)
	}

	switch descriptor.MediaType {
	case dockerOCIIndexMediaType, dockerDistributionV2ManifestListMediaType:
		nestedIndex, err := readDockerOCIIndex(bytes.NewReader(descriptorData))
		if err != nil {
			return nil, fmt.Errorf("failed to decode OCI index %q: %w", descriptorPath, err)
		}
		return resolveDockerOCIManifestFromIndex(dockerFile, nestedIndex, visited)
	case "", dockerOCIManifestMediaType, dockerDistributionV2ManifestMediaType:
		manifest, err := readDockerOCIManifest(bytes.NewReader(descriptorData))
		if err == nil {
			return manifest, nil
		}
		if descriptor.MediaType != "" {
			return nil, err
		}

		nestedIndex, nestedErr := readDockerOCIIndex(bytes.NewReader(descriptorData))
		if nestedErr == nil {
			return resolveDockerOCIManifestFromIndex(dockerFile, nestedIndex, visited)
		}

		return nil, fmt.Errorf("%w: descriptor %q could not be decoded as OCI manifest or OCI index", errDockerOCIDescriptorSkipped, descriptor.Digest)
	default:
		manifest, err := readDockerOCIManifest(bytes.NewReader(descriptorData))
		if err == nil {
			return manifest, nil
		}
		nestedIndex, nestedErr := readDockerOCIIndex(bytes.NewReader(descriptorData))
		if nestedErr == nil {
			return resolveDockerOCIManifestFromIndex(dockerFile, nestedIndex, visited)
		}

		return nil, fmt.Errorf("%w: unsupported OCI descriptor media type %q", errDockerOCIDescriptorSkipped, descriptor.MediaType)
	}
}

func isDockerOCIAttestationDescriptor(descriptor dockerOCIManifestDescriptor) bool {
	if descriptor.Annotations["vnd.docker.reference.type"] == dockerOCIAttestationManifestType {
		return true
	}
	if descriptor.Platform == nil {
		return false
	}

	return descriptor.Platform.Architecture == "unknown" && descriptor.Platform.OS == "unknown"
}

func ociBlobPathFromDigest(digest string) (string, error) {
	algorithm, encodedDigest, found := strings.Cut(digest, ":")
	if !found || algorithm == "" || encodedDigest == "" {
		return "", fmt.Errorf("invalid OCI digest %q", digest)
	}

	return fmt.Sprintf("blobs/%s/%s", algorithm, encodedDigest), nil
}

type dockerInfo struct {
	Config struct {
		Entrypoint []string
		Labels     map[string]string
		User       string
		WorkingDir string
	} `json:"config"`
}

func readDockerInfo(r io.Reader) (*dockerInfo, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	var info dockerInfo
	err = json.Unmarshal(data, &info)
	if err != nil {
		return nil, err
	}

	return &info, nil
}

type testTarEntry struct {
	name string
	mode int64
	data []byte
}

func createTestDockerArchive(t *testing.T, entries []testTarEntry) string {
	t.Helper()

	dockerFile := filepath.Join(t.TempDir(), "test.docker.tar.gz")
	file, err := os.Create(dockerFile)
	require.NoError(t, err, "creating test docker archive should not fail")

	gzipWriter := gzip.NewWriter(file)
	tarWriter := tar.NewWriter(gzipWriter)
	writeTestTarEntries(t, tarWriter, entries)

	require.NoError(t, tarWriter.Close(), "closing test docker archive tar writer should not fail")
	require.NoError(t, gzipWriter.Close(), "closing test docker archive gzip writer should not fail")
	require.NoError(t, file.Close(), "closing test docker archive file should not fail")

	return dockerFile
}

func createTestTarData(t *testing.T, entries []testTarEntry) []byte {
	t.Helper()

	var buffer bytes.Buffer
	tarWriter := tar.NewWriter(&buffer)
	writeTestTarEntries(t, tarWriter, entries)
	require.NoError(t, tarWriter.Close(), "closing test layer tar writer should not fail")

	return buffer.Bytes()
}

func writeTestTarEntries(t *testing.T, tarWriter *tar.Writer, entries []testTarEntry) {
	t.Helper()

	for _, entry := range entries {
		header := &tar.Header{
			Name: entry.name,
			Mode: entry.mode,
			Size: int64(len(entry.data)),
		}
		require.NoErrorf(t, tarWriter.WriteHeader(header), "writing tar header for %s should not fail", entry.name)
		_, err := tarWriter.Write(entry.data)
		require.NoErrorf(t, err, "writing tar contents for %s should not fail", entry.name)
	}
}

func gzipTestData(t *testing.T, data []byte) []byte {
	t.Helper()

	var buffer bytes.Buffer
	gzipWriter := gzip.NewWriter(&buffer)
	_, err := gzipWriter.Write(data)
	require.NoError(t, err, "writing gzip test data should not fail")
	require.NoError(t, gzipWriter.Close(), "closing gzip test data writer should not fail")

	return buffer.Bytes()
}

func sha256Digest(data []byte) string {
	sum := sha256.Sum256(data)
	return fmt.Sprintf("sha256:%x", sum)
}
