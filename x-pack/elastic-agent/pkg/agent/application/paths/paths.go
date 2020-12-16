// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package paths

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/release"
)

const (
	tempSubdir = "tmp"
)

var (
	topPath    string
	configPath string
	logsPath   string
	tmpCreator sync.Once
)

func init() {
	topPath = initialTop()
	configPath = topPath
	logsPath = topPath

	fs := flag.CommandLine
	fs.StringVar(&topPath, "path.home", topPath, "Agent root path")
	fs.StringVar(&configPath, "path.config", configPath, "Config path is the directory Agent looks for its config file")
	fs.StringVar(&logsPath, "path.logs", logsPath, "Logs path contains Agent log output")
}

// Top returns the top directory for Elastic Agent, all the versioned
// home directories live under this top-level/data/elastic-agent-${hash}
func Top() string {
	return topPath
}

// TempDir returns agent temp dir located within data dir.
func TempDir() string {
	tmpDir := filepath.Join(Data(), tempSubdir)
	tmpCreator.Do(func() {
		// create tempdir as it probably don't exists
		os.MkdirAll(tmpDir, 0750)
	})
	return tmpDir
}

// Home returns a directory where binary lives
func Home() string {
	return versionedHome(topPath)
}

// Config returns a directory where configuration file lives
func Config() string {
	return configPath
}

// Data returns the data directory for Agent
func Data() string {
	return filepath.Join(Top(), "data")
}

// Logs returns a the log directory for Agent
func Logs() string {
	return logsPath
}

// initialTop returns the initial top-level path for the binary
//
// When nested in top-level/data/elastic-agent-${hash}/ the result is top-level/.
func initialTop() string {
	exePath := retrieveExecutablePath()
	if insideData(exePath) {
		return filepath.Dir(filepath.Dir(exePath))
	}
	return exePath
}

// retrieveExecutablePath returns the executing binary, even if the started binary was a symlink
func retrieveExecutablePath() string {
	execPath, err := os.Executable()
	if err != nil {
		panic(err)
	}
	evalPath, err := filepath.EvalSymlinks(execPath)
	if err != nil {
		panic(err)
	}
	return filepath.Dir(evalPath)
}

// insideData returns true when the exePath is inside of the current Agents data path.
func insideData(exePath string) bool {
	expectedPath := filepath.Join("data", fmt.Sprintf("elastic-agent-%s", release.ShortCommit()))
	return strings.HasSuffix(exePath, expectedPath)
}

func versionedHome(base string) string {
	return filepath.Join(base, "data", fmt.Sprintf("elastic-agent-%s", release.ShortCommit()))
}
