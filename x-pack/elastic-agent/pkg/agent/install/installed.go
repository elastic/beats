// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package install

import (
	"os"
	"path/filepath"
)

// RunningInstalled returns true when executing Agent is the installed Agent.
//
// This verifies the running executable path based on hard-coded paths
// for each platform type.
func RunningInstalled() bool {
	expected := filepath.Join(InstallPath, BinaryName)
	execPath, _ := os.Executable()
	return expected == execPath
}

// Installed returns installed path of Agent when it is installed on the system.
//
// This returns path even if the executing Agent is not the system installed Agent.
func Installed() string {
	expected := filepath.Join(InstallPath, BinaryName)
	_, err := os.Stat(expected)
	if !os.IsNotExist(err) {
		return InstallPath
	}
	return ""
}
