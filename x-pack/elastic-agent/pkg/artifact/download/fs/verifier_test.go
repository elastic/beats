// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fs

import (
	"context"
	"crypto/sha512"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/common/transport/httpcommon"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/program"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact/download"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/release"
)

const (
	version = "7.5.1"
)

var (
	beatSpec = program.Spec{Name: "Filebeat", Cmd: "filebeat", Artifact: "beat/filebeat"}
)

func TestFetchVerify(t *testing.T) {
	timeout := 15 * time.Second
	dropPath := filepath.Join("testdata", "drop")
	installPath := filepath.Join("testdata", "install")
	targetPath := filepath.Join("testdata", "download")
	ctx := context.Background()
	s := program.Spec{Name: "Beat", Cmd: "beat", Artifact: "beats/filebeat"}
	version := "8.0.0"

	targetFilePath := filepath.Join(targetPath, "beat-8.0.0-darwin-x86_64.tar.gz")
	hashTargetFilePath := filepath.Join(targetPath, "beat-8.0.0-darwin-x86_64.tar.gz.sha512")

	// cleanup
	defer os.RemoveAll(targetPath)

	config := &artifact.Config{
		TargetDirectory: targetPath,
		DropPath:        dropPath,
		InstallPath:     installPath,
		OperatingSystem: "darwin",
		Architecture:    "32",
		HTTPTransportSettings: httpcommon.HTTPTransportSettings{
			Timeout: timeout,
		},
	}

	err := prepareFetchVerifyTests(dropPath, targetPath, targetFilePath, hashTargetFilePath)
	assert.NoError(t, err)

	downloader := NewDownloader(config)
	verifier, err := NewVerifier(config, true, nil)
	assert.NoError(t, err)

	// first download verify should fail:
	// download skipped, as invalid package is prepared upfront
	// verify fails and cleans download
	err = verifier.Verify(s, version)
	var checksumErr *download.ChecksumMismatchError
	assert.ErrorAs(t, err, &checksumErr)

	_, err = os.Stat(targetFilePath)
	assert.True(t, os.IsNotExist(err))

	_, err = os.Stat(hashTargetFilePath)
	assert.True(t, os.IsNotExist(err))

	// second one should pass
	// download not skipped: package missing
	// verify passes because hash is not correct
	_, err = downloader.Download(ctx, s, version)
	assert.NoError(t, err)

	// file downloaded ok
	_, err = os.Stat(targetFilePath)
	assert.NoError(t, err)

	_, err = os.Stat(hashTargetFilePath)
	assert.NoError(t, err)

	err = verifier.Verify(s, version)
	assert.NoError(t, err)

	// Enable GPG signature validation.
	verifier.allowEmptyPgp = false

	// Bad GPG public key.
	{
		verifier.pgpBytes = []byte("garbage")

		// Don't delete anything.
		assertFileExists(t, targetFilePath)
		assertFileExists(t, targetFilePath+".sha512")
	}

	// Setup proper GPG public key.
	_, verifier.pgpBytes = release.PGP()

	// Missing .asc file.
	{
		err = verifier.Verify(s, version)
		require.Error(t, err)

		// Don't delete these files when GPG validation failure.
		assertFileExists(t, targetFilePath)
		assertFileExists(t, targetFilePath+".sha512")
	}

	// Invalid signature.
	{
		err = ioutil.WriteFile(targetFilePath+".asc", []byte("bad sig"), 0o600)
		require.NoError(t, err)

		err = verifier.Verify(s, version)
		var invalidSigErr *download.InvalidSignatureError
		assert.ErrorAs(t, err, &invalidSigErr)

		// Don't delete these files when GPG validation failure.
		assertFileExists(t, targetFilePath)
		assertFileExists(t, targetFilePath+".sha512")

		// Bad .asc file should be removed.
		assertFileNotExists(t, targetFilePath+".asc")
	}
}

func prepareFetchVerifyTests(dropPath, targetDir, targetFilePath, hashTargetFilePath string) error {
	sourceFilePath := filepath.Join(dropPath, "beat-8.0.0-darwin-x86_64.tar.gz")
	hashSourceFilePath := filepath.Join(dropPath, "beat-8.0.0-darwin-x86_64.tar.gz.sha512")

	// clean targets
	os.Remove(targetFilePath)
	os.Remove(hashTargetFilePath)

	if err := os.MkdirAll(targetDir, 0775); err != nil {
		return err
	}

	sourceFile, err := os.Open(sourceFilePath)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	targretFile, err := os.OpenFile(targetFilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}
	defer targretFile.Close()

	hashContent, err := ioutil.ReadFile(hashSourceFilePath)
	if err != nil {
		return err
	}

	corruptedHash := append([]byte{1, 2, 3, 4, 5, 6}, hashContent[6:]...)
	return ioutil.WriteFile(hashTargetFilePath, corruptedHash, 0666)
}

func TestVerify(t *testing.T) {
	targetDir, err := ioutil.TempDir(os.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}

	timeout := 30 * time.Second

	config := &artifact.Config{
		TargetDirectory: targetDir,
		DropPath:        filepath.Join(targetDir, "drop"),
		OperatingSystem: "linux",
		Architecture:    "32",
		HTTPTransportSettings: httpcommon.HTTPTransportSettings{
			Timeout: timeout,
		},
	}

	if err := prepareTestCase(beatSpec, version, config); err != nil {
		t.Fatal(err)
	}

	testClient := NewDownloader(config)
	artifact, err := testClient.Download(context.Background(), beatSpec, version)
	if err != nil {
		t.Fatal(err)
	}

	_, err = os.Stat(artifact)
	if err != nil {
		t.Fatal(err)
	}

	testVerifier, err := NewVerifier(config, true, nil)
	if err != nil {
		t.Fatal(err)
	}

	err = testVerifier.Verify(beatSpec, version)
	require.NoError(t, err)

	os.Remove(artifact)
	os.Remove(artifact + ".sha512")
	os.RemoveAll(config.DropPath)
}

func prepareTestCase(beatSpec program.Spec, version string, cfg *artifact.Config) error {
	filename, err := artifact.GetArtifactName(beatSpec, version, cfg.OperatingSystem, cfg.Architecture)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(cfg.DropPath, 0777); err != nil {
		return err
	}

	content := []byte("sample content")
	hash := sha512.Sum512(content)
	hashContent := fmt.Sprintf("%x %s", hash, filename)

	if err := ioutil.WriteFile(filepath.Join(cfg.DropPath, filename), []byte(content), 0644); err != nil {
		return err
	}

	return ioutil.WriteFile(filepath.Join(cfg.DropPath, filename+".sha512"), []byte(hashContent), 0644)
}

func assertFileExists(t testing.TB, path string) {
	t.Helper()
	_, err := os.Stat(path)
	assert.NoError(t, err, "file %s does not exist", path)
}

func assertFileNotExists(t testing.TB, path string) {
	t.Helper()
	_, err := os.Stat(path)
	assert.ErrorIs(t, err, os.ErrNotExist)
}
