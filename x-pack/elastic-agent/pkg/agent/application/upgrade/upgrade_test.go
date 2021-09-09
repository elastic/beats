// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package upgrade

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/release"
	"github.com/stretchr/testify/require"
)

func TestShutdownCallback(t *testing.T) {
	l, _ := logger.New("test", false)
	tmpDir, err := ioutil.TempDir("", "shutdown-test-")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// make homepath agent consistent (in a form of elastic-agent-hash)
	homePath := filepath.Join(tmpDir, fmt.Sprintf("%s-%s", agentName, release.ShortCommit()))

	filename := "file.test"
	newCommit := "abc123"
	sourceVersion := "7.14.0"
	targetVersion := "7.15.0"

	content := []byte("content")
	newHome := strings.ReplaceAll(homePath, release.ShortCommit(), newCommit)
	sourceDir := filepath.Join(homePath, "run", "default", "process-"+sourceVersion)
	targetDir := filepath.Join(newHome, "run", "default", "process-"+targetVersion)

	require.NoError(t, os.MkdirAll(sourceDir, 0755))
	require.NoError(t, os.MkdirAll(targetDir, 0755))

	cb := shutdownCallback(l, homePath, sourceVersion, targetVersion, newCommit)

	oldFilename := filepath.Join(sourceDir, filename)
	err = ioutil.WriteFile(oldFilename, content, 0640)
	require.NoError(t, err, "preparing file failed")

	err = cb()
	require.NoError(t, err, "callback failed")

	newFilename := filepath.Join(targetDir, filename)
	newContent, err := ioutil.ReadFile(newFilename)
	require.NoError(t, err, "reading file failed")
	require.Equal(t, content, newContent, "contents are not equal")
}
