// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/elastic/beats/v7/x-pack/agent/pkg/agent/application"
	"github.com/elastic/beats/v7/x-pack/agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/agent/pkg/cli"
	"github.com/elastic/beats/v7/x-pack/agent/pkg/config"
	"github.com/elastic/beats/v7/x-pack/agent/pkg/core/logger"
)

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
		return errors.New(err,
			fmt.Sprintf("could not read configuration file %s", flags.PathConfigFile),
			errors.TypeFilesystem,
			errors.M(errors.MetaKeyPath, flags.PathConfigFile))
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
	signal.Notify(signals, syscall.SIGINT, syscall.SIGKILL, syscall.SIGTERM, syscall.SIGQUIT)

	<-signals

	return app.Stop()
}
