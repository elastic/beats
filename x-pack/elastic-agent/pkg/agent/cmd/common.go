// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cmd

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/paths"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/basecmd"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/cli"
)

const defaultConfig = "elastic-agent.yml"

type globalFlags struct {
	PathConfig      string
	PathConfigFile  string
	FlagStrictPerms bool
}

// Config returns path which identifies configuration file.
func (f *globalFlags) Config() string {
	if len(f.PathConfigFile) == 0 || f.PathConfigFile == defaultConfig {
		return filepath.Join(paths.Home(), defaultConfig)
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
		Use: "elastic-agent [subcommand]",
	}

	flags := &globalFlags{}

	cmd.PersistentFlags().AddGoFlag(flag.CommandLine.Lookup("path.home"))
	cmd.PersistentFlags().AddGoFlag(flag.CommandLine.Lookup("path.data"))

	cmd.PersistentFlags().StringVarP(&flags.PathConfigFile, "", "c", defaultConfig, fmt.Sprintf(`Configuration file, relative to path.config (default "%s")`, defaultConfig))
	cmd.PersistentFlags().StringVarP(&flags.PathConfig, "path.config", "", "${path.home}", "Configuration path")
	cmd.PersistentFlags().BoolVarP(&flags.FlagStrictPerms, "strict.perms", "", true, "Strict permission checking on config files")

	// Add version.
	cmd.AddCommand(basecmd.NewDefaultCommandsWithArgs(args, streams)...)
	cmd.AddCommand(newRunCommandWithArgs(flags, args, streams))
	cmd.AddCommand(newEnrollCommandWithArgs(flags, args, streams))
	cmd.AddCommand(newIntrospectCommandWithArgs(flags, args, streams))

	return cmd
}
