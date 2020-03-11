// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"os"
	"path/filepath"

	"github.com/elastic/beats/v7/x-pack/agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/agent/pkg/config"
)

var (
	homePath string
	dataPath string
)

func init() {
	homePath = retrieveExecutablePath()
	dataPath = retrieveDataPath()
}

// InjectAgentConfig injects config to a provided configuration.
func InjectAgentConfig(c *config.Config) error {
	globalConfig := AgentGlobalConfig()
	if err := c.Merge(globalConfig); err != nil {
		return errors.New("failed to inject agent global config", err, errors.TypeConfig)
	}

	return nil
}

// AgentGlobalConfig gets global config used for resolution of variables inside configuration
// such as ${path.data}.
func AgentGlobalConfig() map[string]interface{} {
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
