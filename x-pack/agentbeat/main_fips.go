// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build requirefips

package main

import (
	"fmt"
	"os"
	_ "time/tzdata" // for timezone handling

	"github.com/elastic/beats/v7/libbeat/cfgfile"
	"github.com/elastic/beats/v7/libbeat/cmd"

	"github.com/spf13/cobra"

	auditbeat "github.com/elastic/beats/v7/x-pack/auditbeat/cmd"
	filebeat "github.com/elastic/beats/v7/x-pack/filebeat/cmd"
	metricbeat "github.com/elastic/beats/v7/x-pack/metricbeat/cmd"
)

func main() {
	rootCmd := AgentBeat()
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func AgentBeat() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "agentbeat",
		Short: "Combined beat ran only by the Elastic Agent",
		Long: `Combines auditbeat, filebeat and metricbeat
into a single agentbeat binary.`,
		Example: "agentbeat filebeat run",
	}

	rootCmd.AddCommand(
		prepareCommand(auditbeat.RootCmd),
		prepareCommand(filebeat.Filebeat()),
		prepareCommand(metricbeat.Initialize()),
	)

	return rootCmd
}

func prepareCommand(rootCmd *cmd.BeatsRootCmd) *cobra.Command {
	var origPersistentPreRun func(cmd *cobra.Command, args []string)
	var origPersistentPreRunE func(cmd *cobra.Command, args []string) error
	origPersistentPreRun = rootCmd.PersistentPreRun
	origPersistentPreRunE = rootCmd.PersistentPreRunE
	rootCmd.PersistentPreRun = nil
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		// same logic is used inside of *cobra.Command; if both are set the E version is used instead
		if origPersistentPreRunE != nil {
			if err := origPersistentPreRunE(cmd, args); err != nil {
				// no context is added by cobra, same approach here
				return err
			}
		} else if origPersistentPreRun != nil {
			origPersistentPreRun(cmd, args)
		}
		// must be set to the correct file before the actual Run is performed otherwise it will not be the correct
		// filename, as all the beats set this in the initialization.
		err := cfgfile.ChangeDefaultCfgfileFlag(rootCmd.Use)
		if err != nil {
			panic(fmt.Errorf("failed to set default config file path: %w", err))
		}
		return nil
	}
	return &rootCmd.Command
}
