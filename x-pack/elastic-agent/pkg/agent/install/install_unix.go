// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !windows
// +build !windows

package install

import (
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/paths"
)

// postInstall performs post installation for unix-based systems.
func postInstall() error {
	// do nothing
	return nil
}

func checkPackageInstall() bool {
	if runtime.GOOS != "linux" {
		return false
	}

	// NOTE searching for english words might not be a great idea as far as portability goes.
	// list all installed packages then search for paths.BinaryName?
	// dpkg is strange as the remove and purge processes leads to the package bing isted after a remove, but not after a purge

	// check debian based systems (or systems that use dpkg)
	// If the package has been installed, the status starts with "install"
	// If the package has been removed (but not pruged) status starts with "deinstall"
	// If purged or never installed, rc is 1
	if _, err := os.Stat("/etc/dpkg"); err == nil {
		out, err := exec.Command("dpkg-query", "-W", "-f", "${Status}", paths.BinaryName).Output()
		if err != nil {
			return false
		}
		if strings.HasPrefix(string(out), "deinstall") {
			return false
		}
		return true
	}

	// check rhel and sles based systems (or systems that use rpm)
	// if package has been installed query retuns with a list of associated files.
	// otherwise if uninstalled, or has never been installled status ends with "not installed"
	if _, err := os.Stat("/etc/rpm"); err == nil {
		out, err := exec.Command("rpm", "-q", paths.BinaryName, "--state").Output()
		if err != nil {
			return false
		}
		if strings.HasSuffix(string(out), "not installed") {
			return false
		}
		return true

	}

	return false
}
