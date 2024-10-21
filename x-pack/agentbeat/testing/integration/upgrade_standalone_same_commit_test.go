// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License 2.0;
// you may not use this file except in compliance with the Elastic License 2.0.

//go:build integration

package integration

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent/dev-tools/mage"
	v1 "github.com/elastic/elastic-agent/pkg/api/v1"
	atesting "github.com/elastic/elastic-agent/pkg/testing"
	"github.com/elastic/elastic-agent/pkg/testing/define"
	"github.com/elastic/elastic-agent/pkg/testing/tools/testcontext"
	"github.com/elastic/elastic-agent/pkg/version"
	"github.com/elastic/elastic-agent/testing/upgradetest"
	agtversion "github.com/elastic/elastic-agent/version"
)

func TestStandaloneUpgradeSameCommit(t *testing.T) {
	define.Require(t, define.Requirements{
		Group: Upgrade,
		Local: false, // requires Agent installation
		Sudo:  true,  // requires Agent installation
	})

	// parse the version we are testing
	currentVersion, err := version.ParseVersion(define.Version())
	require.NoError(t, err)

	// 8.13.0-SNAPSHOT is the minimum version we need for testing upgrading with the same hash
	if currentVersion.Less(*upgradetest.Version_8_13_0_SNAPSHOT) {
		t.Skipf("Minimum version for running this test is %q, current version: %q", *upgradetest.Version_8_13_0_SNAPSHOT, currentVersion)
	}

	unprivilegedAvailable := false
	if upgradetest.SupportsUnprivileged(currentVersion) {
		unprivilegedAvailable = true
	}
	unPrivilegedString := "unprivileged"
	if !unprivilegedAvailable {
		unPrivilegedString = "privileged"
	}

	t.Run(fmt.Sprintf("Upgrade on the same version %s to %s (%s)", currentVersion, currentVersion, unPrivilegedString), func(t *testing.T) {
		ctx, cancel := testcontext.WithDeadline(t, context.Background(), time.Now().Add(10*time.Minute))
		defer cancel()

		// ensure we use the same package version
		startFixture, err := define.NewFixtureFromLocalBuild(
			t,
			currentVersion.String(),
		)
		require.NoError(t, err, "error creating start agent fixture")
		err = upgradetest.PerformUpgrade(ctx, startFixture, startFixture, t,
			upgradetest.WithUnprivileged(unprivilegedAvailable),
			upgradetest.WithDisableHashCheck(true),
		)
		assert.ErrorContainsf(t, err, fmt.Sprintf("agent version is already %s", currentVersion), "upgrade should fail indicating we are already at the same version")
	})

	t.Run(fmt.Sprintf("Upgrade on a repackaged version of agent %s (%s)", currentVersion, unPrivilegedString), func(t *testing.T) {
		ctx, cancel := testcontext.WithDeadline(t, context.Background(), time.Now().Add(10*time.Minute))
		defer cancel()

		startFixture, err := define.NewFixtureFromLocalBuild(
			t,
			currentVersion.String(),
		)
		require.NoError(t, err, "error creating start agent fixture")

		// modify the version with the "+buildYYYYMMDDHHMMSS"
		newVersionBuildMetadata := "build" + time.Now().Format("20060102150405")
		parsedNewVersion := version.NewParsedSemVer(currentVersion.Major(), currentVersion.Minor(), currentVersion.Patch(), "", newVersionBuildMetadata)

		err = startFixture.EnsurePrepared(ctx)
		require.NoErrorf(t, err, "fixture should be prepared")

		// retrieve the compressed package file location
		srcPackage, err := startFixture.SrcPackage(ctx)
		require.NoErrorf(t, err, "error retrieving start fixture source package")

		originalPackageFileName := filepath.Base(srcPackage)

		// integration test fixtures and package names treat the version as a string including the "-SNAPSHOT" suffix
		// while the repackage functions below separate version from the snapshot flag.
		// Normally the early release versions are not snapshots but this test runs on PRs and main branch when we test
		// starting from SNAPSHOT packages, so we have to work around the fact that we cannot simply re-generate the packages
		// by defining versions in 2 separate ways for repackage hack and for fixtures
		buildMetadataForAgentFixture := newVersionBuildMetadata
		if currentVersion.IsSnapshot() {
			buildMetadataForAgentFixture += "-SNAPSHOT"
		}
		versionForFixture := version.NewParsedSemVer(currentVersion.Major(), currentVersion.Minor(), currentVersion.Patch(), "", buildMetadataForAgentFixture)

		// calculate the new package name
		newPackageFileName := strings.Replace(originalPackageFileName, currentVersion.String(), versionForFixture.String(), 1)
		t.Logf("originalPackageName: %q newPackageFileName: %q", originalPackageFileName, newPackageFileName)

		newPackageContainingDir := t.TempDir()
		newPackageAbsPath := filepath.Join(newPackageContainingDir, newPackageFileName)

		// hack the package based on type
		ext := filepath.Ext(originalPackageFileName)
		if ext == ".gz" {
			// fetch the next extension
			ext = filepath.Ext(strings.TrimRight(originalPackageFileName, ext)) + ext
		}
		switch ext {
		case ".zip":
			t.Logf("file %q is a .zip package", originalPackageFileName)
			repackageZipArchive(t, srcPackage, newPackageAbsPath, parsedNewVersion)
		case ".tar.gz":
			t.Logf("file %q is a .tar.gz package", originalPackageFileName)
			repackageTarArchive(t, srcPackage, newPackageAbsPath, parsedNewVersion)
		default:
			t.Logf("unknown extension %q for package file %q ", ext, originalPackageFileName)
			t.FailNow()
		}

		// Create hash file for the new package
		err = mage.CreateSHA512File(newPackageAbsPath)
		require.NoErrorf(t, err, "error creating .sha512 for file %q", newPackageAbsPath)

		// I wish I could just pass the location of the package on disk to the whole upgrade tests/fixture/fetcher code
		// but I would have to break too much code for that, when in Rome... add more code on top of inflexible code
		repackagedLocalFetcher := atesting.LocalFetcher(newPackageContainingDir)

		endFixture, err := atesting.NewFixture(t, versionForFixture.String(), atesting.WithFetcher(repackagedLocalFetcher))
		require.NoErrorf(t, err, "error creating end fixture with LocalArtifactFetcher with dir %q", newPackageContainingDir)

		err = upgradetest.PerformUpgrade(ctx, startFixture, endFixture, t,
			upgradetest.WithUnprivileged(unprivilegedAvailable),
			upgradetest.WithDisableHashCheck(true),
		)

		assert.NoError(t, err, "upgrade using version %s from the same commit should succeed")
	})

}

func repackageTarArchive(t *testing.T, srcPackagePath string, newPackagePath string, newVersion *version.ParsedSemVer) {
	oldTopDirectoryName := strings.TrimRight(filepath.Base(srcPackagePath), ".tar.gz")
	newTopDirectoryName := strings.TrimRight(filepath.Base(newPackagePath), ".tar.gz")

	// Open the source package and create readers
	srcPackageFile, err := os.Open(srcPackagePath)
	require.NoErrorf(t, err, "error opening source file %q", srcPackagePath)
	defer func(srcPackageFile *os.File) {
		err := srcPackageFile.Close()
		if err != nil {
			assert.Failf(t, "error closing source file %q: %v", srcPackagePath, err)
		}
	}(srcPackageFile)

	gzReader, err := gzip.NewReader(srcPackageFile)
	require.NoErrorf(t, err, "error creating gzip reader for file %q", srcPackagePath)
	defer func(gzReader *gzip.Reader) {
		err := gzReader.Close()
		if err != nil {
			assert.Failf(t, "error closing gzip reader for source file %q: %v", srcPackagePath, err)
		}
	}(gzReader)

	tarReader := tar.NewReader(gzReader)

	// Create the output file and its writers
	newPackageFile, err := os.OpenFile(newPackagePath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0o750)
	require.NoErrorf(t, err, "error opening output file %q", newPackageFile)
	defer func(newPackageFile *os.File) {
		err := newPackageFile.Close()
		if err != nil {
			assert.Failf(t, "error closing output file %q: %v", newPackagePath, err)
		}
	}(newPackageFile)

	gzWriter := gzip.NewWriter(newPackageFile)
	defer func(gzWriter *gzip.Writer) {
		err := gzWriter.Close()
		if err != nil {
			assert.Failf(t, "error closing gzip writer for file %q: %v", newPackagePath, err)
		}
	}(gzWriter)

	tarWriter := tar.NewWriter(gzWriter)
	defer func(tarWriter *tar.Writer) {
		err := tarWriter.Close()
		if err != nil {
			assert.Failf(t, "error closing tar writer for file %q: %v", newPackagePath, err)
		}
	}(tarWriter)

	hackTarGzPackage(t, tarReader, tarWriter, oldTopDirectoryName, newTopDirectoryName, newVersion)
}

func hackTarGzPackage(t *testing.T, reader *tar.Reader, writer *tar.Writer, oldTopDirName string, newTopDirName string, newVersion *version.ParsedSemVer) {

	for {
		f, err := reader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err, "error reading source package")

		// tar format uses forward slash as path separator, make sure we use only "path" package for checking and manipulation
		switch path.Base(f.Name) {
		case v1.ManifestFileName:
			// read old content and generate the new manifest based on that
			newManifest := generateNewManifestContent(t, reader, newVersion)
			newManifestBytes := []byte(newManifest)

			// fix file length in header
			writeModifiedTarHeader(t, writer, f, oldTopDirName, newTopDirName, int64(len(newManifestBytes)))

			// write the new manifest body
			_, err = writer.Write(newManifestBytes)
			require.NoError(t, err, "error writing out modified manifest")

		case agtversion.PackageVersionFileName:

			t.Logf("writing new package version: %q", newVersion.String())

			// new package version file contents
			newPackageVersionBytes := []byte(newVersion.String())
			// write new header
			writeModifiedTarHeader(t, writer, f, oldTopDirName, newTopDirName, int64(len(newPackageVersionBytes)))
			// write content
			_, err := writer.Write(newPackageVersionBytes)
			require.NoError(t, err, "error writing out modified package version")
		default:
			// write entry header with the size untouched
			writeModifiedTarHeader(t, writer, f, oldTopDirName, newTopDirName, f.Size)

			// copy body
			_, err := io.Copy(writer, reader)
			require.NoErrorf(t, err, "error writing file content for %+v", f)
		}

	}

}

func writeModifiedTarHeader(t *testing.T, writer *tar.Writer, header *tar.Header, oldTopDirName, newTopDirName string, size int64) {
	// replace top dir in the path
	header.Name = strings.Replace(header.Name, oldTopDirName, newTopDirName, 1)
	header.Size = size

	err := writer.WriteHeader(header)
	require.NoErrorf(t, err, "error writing tar header %+v", header)
}

func repackageZipArchive(t *testing.T, srcPackagePath string, newPackagePath string, newVersion *version.ParsedSemVer) {
	oldTopDirectoryName := strings.TrimRight(filepath.Base(srcPackagePath), ".zip")
	newTopDirectoryName := strings.TrimRight(filepath.Base(newPackagePath), ".zip")

	// Open the source package and create readers
	zipReader, err := zip.OpenReader(srcPackagePath)
	require.NoErrorf(t, err, "error opening source file %q", srcPackagePath)
	defer func(zipReader *zip.ReadCloser) {
		err := zipReader.Close()
		if err != nil {
			assert.Failf(t, "error closing source file %q: %v", srcPackagePath, err)
		}
	}(zipReader)

	// Create the output file and its writers
	newPackageFile, err := os.OpenFile(newPackagePath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0o750)
	require.NoErrorf(t, err, "error opening output file %q", newPackageFile)
	defer func(newPackageFile *os.File) {
		err := newPackageFile.Close()
		if err != nil {
			assert.Failf(t, "error closing output file %q: %v", newPackagePath, err)
		}
	}(newPackageFile)

	zipWriter := zip.NewWriter(newPackageFile)
	defer func(zipWriter *zip.Writer) {
		err := zipWriter.Close()
		if err != nil {
			assert.Failf(t, "error closing zip writer for output file %q: %v", newPackagePath, err)
		}
	}(zipWriter)

	hackZipPackage(t, zipReader, zipWriter, oldTopDirectoryName, newTopDirectoryName, newVersion)
}

func hackZipPackage(t *testing.T, reader *zip.ReadCloser, writer *zip.Writer, oldTopDirName string, newTopDirName string, newVersion *version.ParsedSemVer) {
	for _, zippedFile := range reader.File {
		zippedFileHeader := zippedFile.FileHeader

		// zip format uses forward slash as path separator, make sure we use only "path" package for checking and manipulation
		switch path.Base(zippedFile.Name) {
		case v1.ManifestFileName:
			// read old content
			manifestReader, err := zippedFile.Open()
			require.NoError(t, err, "error opening manifest file in zipped package")

			// generate new manifest based on the old manifest and the new version
			newManifest := generateNewManifestContent(t, manifestReader, newVersion)

			// we need to close the file content reader
			err = manifestReader.Close()
			require.NoError(t, err, "error closing manifest file in zipped package")

			newManifestBytes := []byte(newManifest)
			fileContentWriter := writeModifiedZipFileHeader(t, writer, zippedFileHeader, oldTopDirName, newTopDirName, uint64(len(newManifest)))

			_, err = io.Copy(fileContentWriter, bytes.NewReader(newManifestBytes))
			require.NoError(t, err, "error writing out modified manifest")

		case agtversion.PackageVersionFileName:
			t.Logf("writing new package version: %q", newVersion.String())
			// new package version file contents
			newPackageVersionBytes := []byte(newVersion.String())
			fileContentWriter := writeModifiedZipFileHeader(t, writer, zippedFileHeader, oldTopDirName, newTopDirName, uint64(len(newPackageVersionBytes)))

			_, err := io.Copy(fileContentWriter, bytes.NewReader(newPackageVersionBytes))
			require.NoError(t, err, "error writing out modified package version")
		default:
			// write entry header with the size untouched
			fileContentWriter := writeModifiedZipFileHeader(t, writer, zippedFileHeader, oldTopDirName, newTopDirName, zippedFile.UncompressedSize64)
			fileContentReader, err := zippedFile.Open()
			require.NoErrorf(t, err, "error opening zip file content reader for %+v", zippedFileHeader)
			// copy body
			_, err = io.Copy(fileContentWriter, fileContentReader)
			require.NoErrorf(t, err, "error writing file content for %+v", zippedFileHeader)

			// we need to close the file content reader
			err = fileContentReader.Close()
			require.NoError(t, err, "error closing zipped file writer for %+v", zippedFileHeader)
		}
	}
}

func writeModifiedZipFileHeader(t *testing.T, writer *zip.Writer, header zip.FileHeader, oldTopDirName, newTopDirName string, size uint64) io.Writer {
	header.Name = strings.Replace(header.Name, oldTopDirName, newTopDirName, 1)
	header.UncompressedSize64 = size
	fileContentWriter, err := writer.CreateHeader(&header)
	require.NoErrorf(t, err, "error creating header for %+v", header)
	return fileContentWriter
}

func generateNewManifestContent(t *testing.T, manifestReader io.Reader, newVersion *version.ParsedSemVer) string {
	oldManifest, err := v1.ParseManifest(manifestReader)
	require.NoError(t, err, "reading manifest content from tar source archive")

	t.Logf("read old manifest: %+v", oldManifest)

	// replace manifest content
	newManifest, err := mage.GeneratePackageManifest("elastic-agent", newVersion.String(), oldManifest.Package.Snapshot, oldManifest.Package.Hash, oldManifest.Package.Hash[:6])
	require.NoErrorf(t, err, "GeneratePackageManifest(%v, %v, %v, %v) failed", newVersion.String(), oldManifest.Package.Snapshot, oldManifest.Package.Hash, oldManifest.Package.Hash[:6])

	t.Logf("generated new manifest:\n%s", newManifest)
	return newManifest
}
