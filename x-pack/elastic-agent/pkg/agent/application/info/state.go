// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package info

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/paths"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/release"
)

// RunningInstalled returns true when executing Agent is the installed Agent.
//
// This verifies the running executable path based on hard-coded paths
// for each platform type.
func RunningInstalled() bool {
	expected := filepath.Join(paths.InstallPath, paths.BinaryName)
	execPath, _ := os.Executable()
	execPath, _ = filepath.Abs(execPath)
	execName := filepath.Base(execPath)
	execDir := filepath.Dir(execPath)
	if IsInsideData(execDir) {
		// executable path is being reported as being down inside of data path
		// move up to directories to perform the comparison
		execDir = filepath.Dir(filepath.Dir(execDir))
		execPath = filepath.Join(execDir, execName)
	}
	return paths.ArePathsEqual(expected, execPath)
}

// IsInsideData returns true when the exePath is inside of the current Agents data path.
func IsInsideData(exePath string) bool {
	expectedPath := filepath.Join("data", fmt.Sprintf("elastic-agent-%s", release.ShortCommit()))
	return strings.HasSuffix(exePath, expectedPath)
}
