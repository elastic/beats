// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/elastic/beats/v7/x-pack/agent/pkg/basecmd"
	"github.com/elastic/beats/v7/x-pack/agent/pkg/cli"
)

var defaultConfig = "agent.yml"

type globalFlags struct {
	PathConfigFile  string
	PathConfig      string
	PathData        string
	PathHome        string
	PathLogs        string
	FlagStrictPerms bool
}

func (f *globalFlags) Config() string {
	if len(f.PathConfigFile) == 0 {
		return filepath.Join(f.PathHome, defaultConfig)
	}
	return f.PathConfigFile
}

func (f *globalFlags) StrictPermission() bool {
	return f.FlagStrictPerms
}

// NewCommand returns the default command for the agent.
func NewCommand() *cobra.Command {
	return NewCommandWithArgs(os.Args, cli.NewIOStreams())
}

// NewCommandWithArgs returns a new agent with the flags and the subcommand.
func NewCommandWithArgs(args []string, streams *cli.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use: "agent [subcommand]",
	}

	flags := &globalFlags{}

	cmd.PersistentFlags().StringVarP(&flags.PathConfigFile, "", "c", defaultConfig, fmt.Sprintf(`Configuration file, relative to path.config (default "%s")`, defaultConfig))
	cmd.PersistentFlags().StringVarP(&flags.PathHome, "path.home", "", "", "Home path")
	cmd.PersistentFlags().StringVarP(&flags.PathConfig, "path.config", "", "${path.home}", "Configuration path")
	cmd.PersistentFlags().StringVarP(&flags.PathData, "path.data", "", "${path.home}/data", "Data path")
	cmd.PersistentFlags().StringVarP(&flags.PathLogs, "path.logs", "", "${path.home}/logs", "Logs path")
	cmd.PersistentFlags().BoolVarP(&flags.FlagStrictPerms, "strict.perms", "", true, "Strict permission checking on config files")

	// Add version.
	cmd.AddCommand(basecmd.NewDefaultCommandsWithArgs(args, streams)...)
	cmd.AddCommand(newRunCommandWithArgs(flags, args, streams))
	cmd.AddCommand(newEnrollCommandWithArgs(flags, args, streams))

	return cmd
}
