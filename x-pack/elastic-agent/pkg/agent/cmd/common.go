// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cmd

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	// import logp flags
	_ "github.com/elastic/beats/v7/libbeat/logp/configure"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/paths"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/reexec"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/configuration"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/basecmd"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/cli"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/config"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
)

const (
	defaultConfig = "elastic-agent.yml"
	hashLen       = 6
	commitFile    = ".build_hash.txt"
)

type globalFlags struct {
	PathConfigFile string
}

// Config returns path which identifies configuration file.
func (f *globalFlags) Config() string {
	if len(f.PathConfigFile) == 0 || f.PathConfigFile == defaultConfig {
		return filepath.Join(paths.Config(), defaultConfig)
	}
	return f.PathConfigFile
}

// NewCommand returns the default command for the agent.
func NewCommand() *cobra.Command {
	return NewCommandWithArgs(os.Args, cli.NewIOStreams())
}

// NewCommandWithArgs returns a new agent with the flags and the subcommand.
func NewCommandWithArgs(args []string, streams *cli.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use: "elastic-agent [subcommand]",
	}

	flags := &globalFlags{}

	// path flags
	cmd.PersistentFlags().AddGoFlag(flag.CommandLine.Lookup("path.home"))
	cmd.PersistentFlags().AddGoFlag(flag.CommandLine.Lookup("path.config"))
	cmd.PersistentFlags().AddGoFlag(flag.CommandLine.Lookup("path.data"))
	cmd.PersistentFlags().AddGoFlag(flag.CommandLine.Lookup("path.logs"))
	cmd.PersistentFlags().StringVarP(&flags.PathConfigFile, "c", "c", defaultConfig, `Configuration file, relative to path.config`)

	// logging flags
	cmd.PersistentFlags().AddGoFlag(flag.CommandLine.Lookup("v"))
	cmd.PersistentFlags().AddGoFlag(flag.CommandLine.Lookup("e"))
	cmd.PersistentFlags().AddGoFlag(flag.CommandLine.Lookup("d"))
	cmd.PersistentFlags().AddGoFlag(flag.CommandLine.Lookup("environment"))

	// sub-commands
	run := newRunCommandWithArgs(flags, args, streams)
	cmd.AddCommand(basecmd.NewDefaultCommandsWithArgs(args, streams)...)
	cmd.AddCommand(run)
	cmd.AddCommand(newEnrollCommandWithArgs(flags, args, streams))
	cmd.AddCommand(newIntrospectCommandWithArgs(flags, args, streams))

	// windows special hidden sub-command (only added on windows)
	reexec := newReExecWindowsCommand(flags, args, streams)
	if reexec != nil {
		cmd.AddCommand(reexec)
	}
	cmd.PersistentPreRunE = preRunCheck(flags)
	cmd.Run = run.Run

	return cmd
}

func preRunCheck(flags *globalFlags) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		commitFilepath := filepath.Join(paths.Home(), commitFile)
		content, err := ioutil.ReadFile(commitFilepath)

		// file is not present we are at child
		if os.IsNotExist(err) {
			return nil
		} else if err != nil {
			return err
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
		reexecPath := filepath.Join(paths.Data(), hashedDirName(content), filepath.Base(os.Args[0]))
		if err := os.Symlink(reexecPath, origExecPath); err != nil {
			return err
		}

		// generate paths
		if err := generatePaths(filepath.Dir(reexecPath)); err != nil {
			return err
		}

		// reexec
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
		rm := reexec.Manager(rexLogger, reexecPath)

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
		return nil
	}
}

func hashedDirName(filecontent []byte) string {
	s := strings.TrimSpace(string(filecontent))
	if len(s) == 0 {
		return "elastic-agent"
	}

	if len(s) > hashLen {
		s = s[:hashLen]
	}

	return fmt.Sprintf("elastic-agent-%s", s)
}
func generatePaths(dir string) error {
	pathsCfg := map[string]interface{}{
		"--path.data":   paths.Data(),
		"--path.home":   dir,
		"--path.config": paths.Config(),
	}
	pathsCfgPath := filepath.Join(dir, "paths.yml")
	pathsContent, err := yaml.Marshal(pathsCfg)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(pathsCfgPath, pathsContent, 0740)
}
