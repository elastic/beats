// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows
// +build windows

package install

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/paths"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/release"
)

// postInstall performs post installation for Windows systems.
func postInstall() error {
	// delete the top-level elastic-agent.exe
	binary := filepath.Join(paths.InstallPath, paths.BinaryName)
	err := os.Remove(binary)
	if err != nil {
		// do not handle does not exist, it should have existed
		return err
	}

	// create top-level symlink to nested binary
	realBinary := filepath.Join(paths.InstallPath, "data", fmt.Sprintf("elastic-agent-%s", release.ShortCommit()), paths.BinaryName)
	err = os.Symlink(realBinary, binary)
	if err != nil {
		return err
	}

	return nil
}

// checkPackageInstall is used for unix based systems to see if the Elastic-Agent was installed through a package manager.
// returns false
func checkPackageInstall() bool {
	return false
}
