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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

// This file contains Docker archive parsing helpers used by the package tests.
// It understands the legacy `docker save` layout and the OCI image layout that
// current Docker/BuildKit exports use for Beats packaging tests.

type dockerImageType string

const (
	dockerImageTypeLegacy dockerImageType = "legacy"
	dockerImageTypeOCI    dockerImageType = "oci"
)

var errDockerArchiveWalkDone = errors.New("docker archive walk done")
var errDockerArchiveEntryNotFound = errors.New("docker archive entry not found")
var errDockerOCIDescriptorSkipped = errors.New("docker OCI descriptor skipped")

// readDockerOCIFromDaemon hydrates package checks from an already-loaded daemon
// image when the OCI archive is missing blobs. The current fallback relies on
// the compatibility `manifest.json` file to discover the image ref.
func readDockerOCIFromDaemon(t *testing.T, dockerFile string) (*packageFile, *dockerInfo, string, error) {
	t.Helper()

	imageRef, err := dockerImageRefFromArchive(dockerFile)
	if err != nil {
		return nil, nil, "", fmt.Errorf("daemon fallback requires manifest.json with a repo tag in %q: %w", filepath.Base(dockerFile), err)
	}

	ctx, cancel := dockerTestContext(t)
	defer cancel()

	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to get a Docker client: %w", err)
	}

	inspectResp, err := dockerClient.ImageInspect(ctx, imageRef)
	if err != nil {
		return nil, nil, "", fmt.Errorf("daemon fallback requires Docker image %q to already be loaded: %w", imageRef, err)
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

// dockerImageRefFromArchive extracts the image reference from the compatibility
// `manifest.json` file bundled in the current Docker/BuildKit exports.
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

// normalizeDockerArchivePath strips the optional `./` prefix used by some tar
// writers so entry matching is stable across archive producers.
func normalizeDockerArchivePath(name string) string {
	return strings.TrimPrefix(name, "./")
}

// walkDockerArchive iterates the top-level entries in a `.docker.tar.gz`
// archive and lets callers stop the walk early with errDockerArchiveWalkDone.
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

// detectDockerImageType distinguishes the legacy `docker save` layout from the
// OCI image layout by looking for their marker files in the outer archive.
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

// readDockerArchiveEntry reads a single file from the outer `.docker.tar.gz`
// archive without unpacking the rest of the image.
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

// readDocker parses the legacy `docker save` archive layout described by
// `manifest.json`.
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

// readDockerOCI parses the OCI image layout exported by current Docker/BuildKit
// tooling and returns only the subset of files needed by the package checks.
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

// readDockerLayerContents opens a single OCI layer and transparently ungzips it
// when the descriptor media type declares gzip compression.
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

// buildDockerPackageFile merges layers in order and keeps only the files that
// the package assertions care about: the image entrypoint, the working
// directory, and bundled license files.
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

// readManifest loads the compatibility `manifest.json` file used by legacy
// `docker save` archives and by current OCI exports that still include it.
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

// readDockerManifest decodes the compatibility `manifest.json` format and
// expects the archive to contain exactly one image manifest entry.
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

// Media types used while traversing OCI indexes, image manifests, and Docker's
// attestation descriptors.
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

// readDockerOCIIndex decodes an OCI index or Docker manifest list blob.
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

// readDockerOCIManifest decodes a runnable OCI image manifest and rejects
// descriptors that do not carry image config or layer references.
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

// resolveDockerOCIManifestFromIndex walks the descriptors in an OCI index until
// it finds a runnable image manifest. Descriptors marked as skippable are
// ignored so attestation entries do not fail the package checks.
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

// resolveDockerOCIManifestFromDescriptor recursively resolves a descriptor to a
// runnable image manifest. It understands nested OCI indexes and skips
// descriptors that represent attestations or other non-runnable artifacts.
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

// isDockerOCIAttestationDescriptor identifies the BuildKit attestation
// descriptors that should be ignored when choosing the runnable image manifest.
func isDockerOCIAttestationDescriptor(descriptor dockerOCIManifestDescriptor) bool {
	if descriptor.Annotations["vnd.docker.reference.type"] == dockerOCIAttestationManifestType {
		return true
	}
	if descriptor.Platform == nil {
		return false
	}

	return descriptor.Platform.Architecture == "unknown" && descriptor.Platform.OS == "unknown"
}

// ociBlobPathFromDigest converts an OCI digest into its on-disk blob path
// inside the archive layout.
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

// readDockerInfo decodes the subset of image config fields used by the package
// checks.
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
