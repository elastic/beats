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
	// DefaultConfigName is the default name of the configuration file.
	DefaultConfigName = "elastic-agent.yml"
	// AgentLockFileName is the name of the overall Elastic Agent file lock.
	AgentLockFileName = "agent.lock"
	tempSubdir        = "tmp"
)

var (
	topPath         string
	configPath      string
	configFilePath  string
	logsPath        string
	unversionedHome bool
	tmpCreator      sync.Once
)

func init() {
	topPath = initialTop()
	configPath = topPath
	logsPath = topPath
	unversionedHome = false // only versioned by container subcommand

	fs := flag.CommandLine
	fs.StringVar(&topPath, "path.home", topPath, "Agent root path")
	fs.BoolVar(&unversionedHome, "path.home.unversioned", unversionedHome, "Agent root path is not versioned based on build")
	fs.StringVar(&configPath, "path.config", configPath, "Config path is the directory Agent looks for its config file")
	fs.StringVar(&configFilePath, "c", DefaultConfigName, "Configuration file, relative to path.config")
	fs.StringVar(&logsPath, "path.logs", logsPath, "Logs path contains Agent log output")
}

// Top returns the top directory for Elastic Agent, all the versioned
// home directories live under this top-level/data/elastic-agent-${hash}
func Top() string {
	return topPath
}

// SetTop overrides the Top path.
//
// Used by the container subcommand to adjust the overall top path allowing state can be maintained between container
// restarts.
func SetTop(path string) {
	topPath = path
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
	if unversionedHome {
		return topPath
	}
	return VersionedHome(topPath)
}

// IsVersionHome returns true if the Home path is versioned based on build.
func IsVersionHome() bool {
	return !unversionedHome
}

// SetVersionHome sets if the Home path is versioned based on build.
//
// Used by the container subcommand to adjust the home path allowing state can be maintained between container
// restarts.
func SetVersionHome(version bool) {
	unversionedHome = !version
}

// Config returns a directory where configuration file lives
func Config() string {
	return configPath
}

// SetConfig overrides the Config path.
//
// Used by the container subcommand to adjust the overall config path allowing state can be maintained between container
// restarts.
func SetConfig(path string) {
	configPath = path
}

// ConfigFile returns the path to the configuration file.
func ConfigFile() string {
	if configFilePath == "" || configFilePath == DefaultConfigName {
		return filepath.Join(Config(), DefaultConfigName)
	}
	if filepath.IsAbs(configFilePath) {
		return configFilePath
	}
	return filepath.Join(Config(), configFilePath)
}

// Data returns the data directory for Agent
func Data() string {
	if unversionedHome {
		// unversioned means the topPath is the data path
		return topPath
	}
	return filepath.Join(Top(), "data")
}

// Logs returns a the log directory for Agent
func Logs() string {
	return logsPath
}

// SetLogs updates the path for the logs.
func SetLogs(path string) {
	logsPath = path
}

// VersionedHome returns a versioned path based on a TopPath and used commit.
func VersionedHome(base string) string {
	return filepath.Join(base, "data", fmt.Sprintf("elastic-agent-%s", release.ShortCommit()))
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
