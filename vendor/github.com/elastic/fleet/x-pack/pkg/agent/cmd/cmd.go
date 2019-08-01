// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/elastic/fleet/x-pack/pkg/agent/application"
	"github.com/elastic/fleet/x-pack/pkg/basecmd"
	"github.com/elastic/fleet/x-pack/pkg/cli"
	"github.com/elastic/fleet/x-pack/pkg/config"
	"github.com/elastic/fleet/x-pack/pkg/core/logger"
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

	return cmd
}

func newRunCommandWithArgs(flags *globalFlags, _ []string, streams *cli.IOStreams) *cobra.Command {
	return &cobra.Command{
		Use:   "run",
		Short: "Start the agent.",
		Run: func(_ *cobra.Command, _ []string) {
			if err := run(flags, streams); err != nil {
				fmt.Fprintf(streams.Err, "%v\n", err)
				os.Exit(1)
			}
		},
	}
}

func run(flags *globalFlags, streams *cli.IOStreams) error {
	config, err := config.LoadYAML(flags.PathConfigFile)
	if err != nil {
		return errors.Wrapf(err, "could not read configuration file %s", flags.PathConfigFile)
	}

	logger, err := logger.NewFromConfig(config)
	if err != nil {
		return err
	}

	app, err := application.New(logger, flags.PathConfigFile)
	if err != nil {
		return err
	}

	if err := app.Start(); err != nil {
		return err
	}

	// listen for kill signal
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Kill, os.Interrupt)

	<-signals

	return app.Stop()
}
