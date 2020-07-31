// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package paths

import (
	"flag"
	"os"
	"path/filepath"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/config"
)

var (
	homePath   string
	configPath string
	dataPath   string
	logsPath   string
)

type paths struct {
	HomePath   string `yaml:"path.home"`
	ConfigPath string `yaml:"path.config"`
	DataPath   string `yaml:"path.data"`
	LogsPath   string `yaml:"path.logs"`
}

func init() {
	defaults := getDefaultValues()

	fs := flag.CommandLine
	fs.StringVar(&homePath, "path.home", defaults.HomePath, "Agent root path")
	fs.StringVar(&configPath, "path.config", defaults.ConfigPath, "Config path is the directory Agent looks for its config file")
	fs.StringVar(&dataPath, "path.data", defaults.DataPath, "Data path contains Agent managed binaries")
	fs.StringVar(&logsPath, "path.logs", defaults.LogsPath, "Logs path contains Agent log output")
}

func getDefaultValues() paths {
	exePath := retrieveExecutablePath()
	defaults := paths{
		HomePath:   exePath,
		ConfigPath: exePath,
		DataPath:   filepath.Join(exePath, "data"),
		LogsPath:   exePath,
	}

	if rawConfig, err := config.LoadYAML("paths.yml"); err == nil {
		rawConfig.Unpack(&defaults)
	}

	return defaults
}

// Home returns a directory where binary lives
// Executable is not supported on nacl.
func Home() string {
	return homePath
}

// Config returns a directory where configuration file lives
func Config() string {
	return configPath
}

// Data returns the data directory for Agent
func Data() string {
	return dataPath
}

// Logs returns a the log directory for Agent
func Logs() string {
	return logsPath
}

func retrieveExecutablePath() string {
	execPath, err := os.Executable()
	if err != nil {
		panic(err)
	}

	return filepath.Dir(execPath)
}
