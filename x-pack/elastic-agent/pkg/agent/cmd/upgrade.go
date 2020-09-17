// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/upgrade"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/configuration"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/cli"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/fleetapi"
)

// TODO: remove before merging

func newUpgradeCommandWithArgs(flags *globalFlags, _ []string, streams *cli.IOStreams) *cobra.Command {
	return &cobra.Command{
		Use:   "upgrade",
		Short: "Start the elastic-agent.",
		Run: func(_ *cobra.Command, _ []string) {
			if err := runUpgrade(flags, streams); err != nil {
				fmt.Fprintf(streams.Err, "%v\n", err)
				os.Exit(1)
			}
		},
	}
}

func runUpgrade(flags *globalFlags, streams *cli.IOStreams) error { // Windows: Mark service as stopped.
	pathConfigFile := flags.Config()
	rawConfig, err := application.LoadConfigFromFile(pathConfigFile)
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

	r := &rexecer{logger: logger}
	upgrader := upgrade.NewUpgrader(
		cfg.Settings.DownloadConfig,
		logger,
		[]context.CancelFunc{},
		r,
		&acker{})

	// https://snapshots.elastic.co/8.0.0-5942871b/downloads/beats/elastic-agent/elastic-agent-8.0.0-SNAPSHOT-darwin-x86_64.tar.gz
	action := &fleetapi.ActionUpgrade{
		ActionID:   "12345-abcd",
		ActionType: "UPGRADE",
		Version:    "8.0.0-SNAPSHOT",
		SourceURI:  "https://snapshots.elastic.co/8.0.0-5942871b/downloads",
	}
	if upgrader.Upgrade(context.Background(), action); err != nil {
		return err
	}
	return nil
}

type rexecer struct {
	logger *logp.Logger
}

func (r *rexecer) ReExec(argOverrides ...string) {
	r.logger.Warn("REEXEC started")
}

type acker struct {
	logger *logp.Logger
}

func (r *acker) Ack(ctx context.Context, action fleetapi.Action) error {
	r.logger.Warn("ACK ", action.ID)
	return nil
}

func (r *acker) Commit(ctx context.Context) error {
	return nil
}
