// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"os"
	"path/filepath"
)

var (
	homePath string
	dataPath string
)

func init() {
	homePath = retrieveHomePath()
	dataPath = retrieveDataPath()
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

// retrieveDataPath returns a directory where binary lives
// Executable is not supported on nacl.
func retrieveDataPath() string {
	execPath, err := os.Executable()
	if err != nil {
		panic(err)
	}

	return filepath.Base(execPath)
}

// retrieveHomePath returns a home directory of current user
func retrieveHomePath() string {
	// this fails usually for not common OSes
	h, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	return h
}
