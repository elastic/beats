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

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// This file contains self-tests for the Docker archive parsing helpers used by
// the package validation suite.

const testOCIConfigJSON = `{"config":{"Entrypoint":["/docker-entrypoint"],"Labels":{"org.label-schema.vendor":"Elastic"},"User":"root","WorkingDir":"/usr/share/testbeat"}}`

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
	image := createTestOCIImageFixture(t, []testTarEntry{
		{name: "docker-entrypoint", mode: 0o755, data: []byte("#!/bin/sh\n")},
		{name: "usr/share/testbeat/testbeat.yml", mode: 0o644, data: []byte("name: testbeat\n")},
		{name: "usr/share/testbeat/LICENSE.txt", mode: 0o644, data: []byte("license\n")},
		{name: "etc/passwd", mode: 0o644, data: []byte("x\n")},
	})

	indexData := mustMarshalTestJSON(t, dockerOCIIndex{
		SchemaVersion: 2,
		Manifests: []dockerOCIManifestDescriptor{
			{
				MediaType: dockerOCIManifestMediaType,
				Digest:    image.manifestDigest,
				Size:      int64(len(image.manifestData)),
			},
		},
	}, "OCI index marshaling should not fail")

	dockerFile := createTestDockerArchive(t, []testTarEntry{
		{name: "oci-layout", mode: 0o644, data: []byte(`{"imageLayoutVersion":"1.0.0"}`)},
		{name: "index.json", mode: 0o644, data: indexData},
		{name: mustOCIPath(t, image.manifestDigest, "manifest"), mode: 0o644, data: image.manifestData},
		{name: mustOCIPath(t, image.configDigest, "config"), mode: 0o644, data: image.configData},
		{name: mustOCIPath(t, image.layerDigest, "layer"), mode: 0o644, data: image.layerData},
	})

	pkg, info, err := readDockerOCI(dockerFile)
	require.NoError(t, err, "reading OCI docker archive should not fail")
	assertParsedTestOCIImage(t, pkg, info)
	_, found := pkg.Contents["docker-entrypoint"]
	require.True(t, found, "entrypoint file should be present in extracted docker package contents")
	_, found = pkg.Contents["etc/passwd"]
	require.False(t, found, "files outside working directory should not be included")
}

func TestReadDockerOCINestedIndexWithAttestation(t *testing.T) {
	image := createTestOCIImageFixture(t, []testTarEntry{
		{name: "docker-entrypoint", mode: 0o755, data: []byte("#!/bin/sh\n")},
		{name: "usr/share/testbeat/testbeat.yml", mode: 0o644, data: []byte("name: testbeat\n")},
		{name: "usr/share/testbeat/LICENSE.txt", mode: 0o644, data: []byte("license\n")},
	})
	attestationConfigData := []byte(`{"architecture":"unknown","os":"unknown","config":{},"rootfs":{"type":"layers","diff_ids":["sha256:133ae3f9bcc385295b66c2d83b28c25a9f294ce20954d5cf922dda860429734a"]}}`)
	attestationLayerData := []byte(`{"_type":"https://in-toto.io/Statement/v0.1","predicateType":"https://slsa.dev/provenance/v1","subject":[],"predicate":{}}`)
	attestationConfigDigest := sha256Digest(attestationConfigData)
	attestationLayerDigest := sha256Digest(attestationLayerData)

	attestationManifestData := mustMarshalTestJSON(t, dockerOCIManifest{
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
	}, "attestation manifest marshaling should not fail")

	attestationManifestDigest := sha256Digest(attestationManifestData)
	nestedIndexData := mustMarshalTestJSON(t, dockerOCIIndex{
		SchemaVersion: 2,
		MediaType:     dockerOCIIndexMediaType,
		Manifests: []dockerOCIManifestDescriptor{
			{
				MediaType: dockerOCIManifestMediaType,
				Digest:    attestationManifestDigest,
				Size:      int64(len(attestationManifestData)),
				Annotations: map[string]string{
					"vnd.docker.reference.digest": image.manifestDigest,
					"vnd.docker.reference.type":   dockerOCIAttestationManifestType,
				},
				Platform: &dockerOCIPlatform{
					Architecture: "unknown",
					OS:           "unknown",
				},
			},
			{
				MediaType: dockerOCIManifestMediaType,
				Digest:    image.manifestDigest,
				Size:      int64(len(image.manifestData)),
				Platform: &dockerOCIPlatform{
					Architecture: "amd64",
					OS:           "linux",
				},
			},
		},
	}, "nested OCI index marshaling should not fail")

	nestedIndexDigest := sha256Digest(nestedIndexData)
	indexData := mustMarshalTestJSON(t, dockerOCIIndex{
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
	}, "top-level OCI index marshaling should not fail")

	dockerFile := createTestDockerArchive(t, []testTarEntry{
		{name: "oci-layout", mode: 0o644, data: []byte(`{"imageLayoutVersion":"1.0.0"}`)},
		{name: "index.json", mode: 0o644, data: indexData},
		{name: mustOCIPath(t, nestedIndexDigest, "nested index"), mode: 0o644, data: nestedIndexData},
		{name: mustOCIPath(t, image.manifestDigest, "manifest"), mode: 0o644, data: image.manifestData},
		{name: mustOCIPath(t, image.configDigest, "config"), mode: 0o644, data: image.configData},
		{name: mustOCIPath(t, image.layerDigest, "layer"), mode: 0o644, data: image.layerData},
		{name: mustOCIPath(t, attestationManifestDigest, "attestation manifest"), mode: 0o644, data: attestationManifestData},
		{name: mustOCIPath(t, attestationConfigDigest, "attestation config"), mode: 0o644, data: attestationConfigData},
		{name: mustOCIPath(t, attestationLayerDigest, "attestation layer"), mode: 0o644, data: attestationLayerData},
	})

	pkg, info, err := readDockerOCI(dockerFile)
	require.NoError(t, err, "reading OCI docker archive with nested index and attestation should not fail")
	assertParsedTestOCIImage(t, pkg, info)
}

func TestReadDockerOCIMissingBlob(t *testing.T) {
	manifestData := mustMarshalTestJSON(t, dockerOCIManifest{
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
	}, "OCI manifest marshaling should not fail")

	manifestDigest := sha256Digest(manifestData)
	indexData := mustMarshalTestJSON(t, dockerOCIIndex{
		SchemaVersion: 2,
		Manifests: []dockerOCIManifestDescriptor{
			{
				MediaType: dockerOCIManifestMediaType,
				Digest:    manifestDigest,
				Size:      int64(len(manifestData)),
			},
		},
	}, "OCI index marshaling should not fail")

	dockerFile := createTestDockerArchive(t, []testTarEntry{
		{name: "oci-layout", mode: 0o644, data: []byte(`{"imageLayoutVersion":"1.0.0"}`)},
		{name: "index.json", mode: 0o644, data: indexData},
		{name: mustOCIPath(t, manifestDigest, "manifest"), mode: 0o644, data: manifestData},
	})

	_, _, err := readDockerOCI(dockerFile)
	require.Error(t, err, "reading sparse OCI docker archive should fail")
	require.ErrorIs(t, err, errDockerArchiveEntryNotFound, "sparse OCI archive should report missing blob references")
}

type testOCIImageFixture struct {
	configData     []byte
	configDigest   string
	layerData      []byte
	layerDigest    string
	manifestData   []byte
	manifestDigest string
}

func createTestOCIImageFixture(t *testing.T, layerEntries []testTarEntry) testOCIImageFixture {
	t.Helper()

	configData := []byte(testOCIConfigJSON)
	layerTar := createTestTarData(t, layerEntries)
	layerData := gzipTestData(t, layerTar)

	configDigest := sha256Digest(configData)
	layerDigest := sha256Digest(layerData)
	manifestData := mustMarshalTestJSON(t, dockerOCIManifest{
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
	}, "OCI manifest marshaling should not fail")

	return testOCIImageFixture{
		configData:     configData,
		configDigest:   configDigest,
		layerData:      layerData,
		layerDigest:    layerDigest,
		manifestData:   manifestData,
		manifestDigest: sha256Digest(manifestData),
	}
}

func assertParsedTestOCIImage(t *testing.T, pkg *packageFile, info *dockerInfo) {
	t.Helper()

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

func mustMarshalTestJSON(t *testing.T, value any, failureMessage string) []byte {
	t.Helper()

	data, err := json.Marshal(value)
	require.NoError(t, err, failureMessage)
	return data
}

func mustOCIPath(t *testing.T, digest, subject string) string {
	t.Helper()

	path, err := ociBlobPathFromDigest(digest)
	require.NoErrorf(t, err, "%s digest should produce a valid OCI blob path", subject)
	return path
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
