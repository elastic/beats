// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build windows

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
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/reexec"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/configuration"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/config"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/release"
)

func preRunCheck(flags *globalFlags) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		if sn := paths.ServiceName(); sn != "" {
			// replacing with correct service name so we
			// can talk to service manager.
			if !filepath.IsAbs(os.Args[0]) {
				os.Args[0] = sn
			}

			// paths were created we're running as child.
			return nil
		}

		smallHash := fmt.Sprintf("elastic-agent-%s", smallHash(release.Commit()))
		commitFilepath := filepath.Join(paths.Config(), commitFile)
		if content, err := ioutil.ReadFile(commitFilepath); err == nil {
			smallHash = hashedDirName(content)
		}

		// rename itself
		origExecPath, err := os.Executable()
		if err != nil {
			return err
		}

		if err := os.Rename(origExecPath, origExecPath+".bak"); err != nil {
			return err
		}

		// create symlink to elastic-agent-{hash}
		reexecPath := filepath.Join(paths.Data(), smallHash, filepath.Base(origExecPath))
		if err := os.Symlink(reexecPath, origExecPath); err != nil {
			return err
		}

		// generate paths
		if err := generatePaths(filepath.Dir(reexecPath), origExecPath); err != nil {
			return err
		}

		paths.UpdatePaths()

		// reexec if running run
		if cmd.Use == "run" {
			pathConfigFile := flags.Config()
			rawConfig, err := config.LoadYAML(pathConfigFile)
			if err != nil {
				return errors.New(err,
					fmt.Sprintf("could not read configuration file %s", pathConfigFile),
					errors.TypeFilesystem,
					errors.M(errors.MetaKeyPath, pathConfigFile))
			}

			cfg, err := configuration.NewFromConfig(rawConfig)
			if err != nil {
				return errors.New(err,
					fmt.Sprintf("could not parse configuration file %s", pathConfigFile),
					errors.TypeFilesystem,
					errors.M(errors.MetaKeyPath, pathConfigFile))
			}

			logger, err := logger.NewFromConfig("", cfg.Settings.LoggingConfig)
			if err != nil {
				return err
			}

			rexLogger := logger.Named("reexec")
			rm := reexec.NewManager(rexLogger, reexecPath)

			argsOverrides := []string{
				"--path.data", paths.Data(),
				"--path.home", filepath.Dir(reexecPath),
				"--path.config", paths.Config(),
			}
			rm.ReExec(argsOverrides...)

			// trigger reexec
			rm.ShutdownComplete()

			// return without running Run method
			os.Exit(0)
		}

		return nil
	}
}
