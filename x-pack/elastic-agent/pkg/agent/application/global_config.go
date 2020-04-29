// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"os"
	"path/filepath"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/config"
)

var (
	homePath   string
	dataPath   string
	overwrites *common.Config
)

func init() {
	homePath = retrieveExecutablePath()
	dataPath = retrieveDataPath()
	overwrites = common.NewConfig()
	common.ConfigOverwriteFlag(nil, overwrites, "path.home", "path.home", "", "Agent root path")
	common.ConfigOverwriteFlag(nil, overwrites, "path.data", "path.data", "", "Data path contains Agent managed binaries")
}

// HomePath returns home path where.
func HomePath() string {
	if val, err := overwrites.String("path.home", -1); err == nil {
		return val
	}

	return homePath
}

// DataPath returns data path where.
func DataPath() string {
	if val, err := overwrites.String("path.data", -1); err == nil {
		return val
	}

	return dataPath
}

// InjectAgentConfig injects config to a provided configuration.
func InjectAgentConfig(c *config.Config) error {
	globalConfig := agentGlobalConfig()
	if err := c.Merge(globalConfig); err != nil {
		return errors.New("failed to inject agent global config", err, errors.TypeConfig)
	}

	return injectOverwrites(c)
}

func injectOverwrites(c *config.Config) error {
	if err := c.Merge(overwrites); err != nil {
		return errors.New("failed to inject agent overwrites", err, errors.TypeConfig)
	}

	return nil
}

// agentGlobalConfig gets global config used for resolution of variables inside configuration
// such as ${path.data}.
func agentGlobalConfig() map[string]interface{} {
	return map[string]interface{}{
		"path": map[string]interface{}{
			"data": dataPath,
			"home": homePath,
		},
	}
}

// retrieveExecutablePath returns a directory where binary lives
// Executable is not supported on nacl.
func retrieveExecutablePath() string {
	execPath, err := os.Executable()
	if err != nil {
		panic(err)
	}

	return filepath.Dir(execPath)
}

// retrieveHomePath returns a home directory of current user
func retrieveDataPath() string {
	return filepath.Join(retrieveExecutablePath(), "data")
}
