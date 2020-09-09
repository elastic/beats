// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build !windows

package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	// import logp flags
	_ "github.com/elastic/beats/v7/libbeat/logp/configure"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/paths"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/release"
)

// preRunCheck is noop because
// - darwin.tar - symlink created during packaging
// - linux.tar - symlink created during packaging
// - linux.rpm - symlink created using install script
// - linux.deb - symlink created using install script
// - linux.docker - symlink created using Dockerfile
func preRunCheck(flags *globalFlags) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		if sn := paths.ServiceName(); sn != "" {
			// paths were created we're running as child.
			return nil
		}

		// get versioned path
		smallHash := fmt.Sprintf("elastic-agent-%s", smallHash(release.Commit()))
		commitFilepath := filepath.Join(paths.Config(), commitFile) // use other file in the future
		if content, err := ioutil.ReadFile(commitFilepath); err == nil {
			smallHash = hashedDirName(content)
		}

		origExecPath, err := os.Executable()
		if err != nil {
			return err
		}
		reexecPath := filepath.Join(paths.Data(), smallHash, filepath.Base(origExecPath))

		// generate paths
		if err := generatePaths(filepath.Dir(reexecPath), origExecPath); err != nil {
			return err
		}

		paths.UpdatePaths()
		return nil
	}
}
