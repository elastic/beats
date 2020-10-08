// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package install

import (
	"os"
	"path/filepath"

	"github.com/kardianos/service"
)

// StatusType is the return status types.
type StatusType int

const (
	// NotInstalled returned when Elastic Agent is not installed.
	NotInstalled StatusType = iota
	// Installed returned when Elastic Agent is installed currectly.
	Installed
	// Broken returned when Elastic Agent is installed but broken.
	Broken
)

// Status returns the installation status of Agent.
func Status() (StatusType, string) {
	expected := filepath.Join(InstallPath, BinaryName)
	status, reason := checkService()
	_, err := os.Stat(expected)
	if os.IsNotExist(err) {
		if status == Installed {
			// service installed, but no install path
			return Broken, "service exists but installation path is missing"
		}
		return NotInstalled, "no install path or service"
	}
	if status == NotInstalled {
		// install path present, but not service
		return Broken, reason
	}
	return Installed, ""
}

// RunningInstalled returns true when executing Agent is the installed Agent.
//
// This verifies the running executable path based on hard-coded paths
// for each platform type.
func RunningInstalled() bool {
	expected := filepath.Join(InstallPath, BinaryName)
	execPath, _ := os.Executable()
	execPath, _ = filepath.Abs(execPath)
	execName := filepath.Base(execPath)
	execDir := filepath.Dir(execPath)
	if insideData(execDir) {
		// executable path is being reported as being down inside of data path
		// move up to directories to perform the comparison
		execDir = filepath.Dir(filepath.Dir(execDir))
		execPath = filepath.Join(execDir, execName)
	}
	return expected == execPath
}

// checkService only checks the status of the service.
func checkService() (StatusType, string) {
	svc, err := newService()
	if err != nil {
		return NotInstalled, "unable to check service status"
	}
	status, _ := svc.Status()
	if status == service.StatusUnknown {
		return NotInstalled, "service is not installed"
	}
	return Installed, ""
}
