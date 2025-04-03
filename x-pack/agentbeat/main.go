// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"fmt"
	"os"
	_ "time/tzdata" // for timezone handling

	"github.com/elastic/beats/v7/libbeat/cfgfile"
	"github.com/elastic/beats/v7/libbeat/cmd"

	"github.com/spf13/cobra"
)

func main() {
	rootCmd := AgentBeat()
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func AgentBeat() *cobra.Command {
	return prepareRootCommand()
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
