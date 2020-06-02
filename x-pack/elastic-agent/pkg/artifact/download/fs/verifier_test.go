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

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact"
)

const (
	beatName      = "filebeat"
	version       = "7.5.1"
	sourcePattern = "/downloads/beats/filebeat/"
)

type testCase struct {
	system string
	arch   string
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
		Timeout:         timeout,
		OperatingSystem: "linux",
		Architecture:    "32",
	}

	if err := prepareTestCase(beatName, version, config); err != nil {
		t.Fatal(err)
	}

	testClient := NewDownloader(config)
	artifact, err := testClient.Download(context.Background(), beatName, version)
	if err != nil {
		t.Fatal(err)
	}

	_, err = os.Stat(artifact)
	if err != nil {
		t.Fatal(err)
	}

	testVerifier, err := NewVerifier(config)
	if err != nil {
		t.Fatal(err)
	}

	isOk, err := testVerifier.Verify(beatName, version)
	if err != nil {
		t.Fatal(err)
	}

	if !isOk {
		t.Fatal("verify failed")
	}

	os.Remove(artifact)
	os.Remove(artifact + ".sha512")
	os.RemoveAll(config.DropPath)
}

func prepareTestCase(beatName, version string, cfg *artifact.Config) error {
	filename, err := artifact.GetArtifactName(beatName, version, cfg.OperatingSystem, cfg.Architecture)
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
