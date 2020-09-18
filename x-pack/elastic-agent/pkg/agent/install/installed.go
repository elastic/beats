// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package install

import (
	"github.com/kardianos/service"
	"os"
	"path/filepath"
)

type InstallStatus int

const (
	// NotInstalled returned when Elastic Agent is not installed.
	NotInstalled InstallStatus = iota
	// Installed returned when Elastic Agent is installed currectly.
	Installed
	// Broken returned when Elastic Agent is installed but broken.
	Broken
)

// Status returns the installation status of Agent.
func Status() (InstallStatus, string) {
	expected := filepath.Join(InstallPath, BinaryName)
	_, err := os.Stat(expected)
	if os.IsNotExist(err) {
		return NotInstalled, "binary not located at install path"
	}
	svc, err := newService()
	if err != nil {
		return Broken, "unable to check service status"
	}
	status, err := svc.Status()
	if err != nil {
		return Broken, "unable to check service status"
	}
	if status == service.StatusUnknown {
		return Broken, "service is not installed"
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
	return expected == execPath
}
