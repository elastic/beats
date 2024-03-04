// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"fmt"
	"github.com/elastic/beats/v7/libbeat/cfgfile"
	"github.com/elastic/beats/v7/libbeat/cmd"
	"os"

	"github.com/spf13/cobra"

	auditbeat "github.com/elastic/beats/v7/x-pack/auditbeat/cmd"
	filebeat "github.com/elastic/beats/v7/x-pack/filebeat/cmd"
	heartbeat "github.com/elastic/beats/v7/x-pack/heartbeat/cmd"
	metricbeat "github.com/elastic/beats/v7/x-pack/metricbeat/cmd"
	osquerybeat "github.com/elastic/beats/v7/x-pack/osquerybeat/cmd"
	packetbeat "github.com/elastic/beats/v7/x-pack/packetbeat/cmd"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "agentbeat",
		Short: "Combined beat ran only by the Elastic Agent",
		Long: `Combines auditbeat, filebeat, heartbeat, metricbeat, osquerybeat, and packetbeat
into a single agentbeat binary.`,
		Example: "agentbeat filebeat run",
	}

	rootCmd.AddCommand(
		prepareCommand(auditbeat.RootCmd),
		prepareCommand(filebeat.Filebeat()),
		prepareCommand(heartbeat.RootCmd),
		prepareCommand(metricbeat.RootCmd),
		prepareCommand(osquerybeat.RootCmd),
		prepareCommand(packetbeat.RootCmd),
	)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func prepareCommand(rootCmd *cmd.BeatsRootCmd) *cobra.Command {
	if rootCmd.PersistentPreRun != nil || rootCmd.PersistentPreRunE != nil {
		panic("command cannot already have PersistentPreRun or PersistentPreRunE set")
	}
	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		// must be set to the correct file before the actual Run is performed otherwise it will not be the correct
		// filename, as all the beats set this in the initialization.
		err := cfgfile.ChangeDefaultCfgfileFlag(rootCmd.Use)
		if err != nil {
			panic(fmt.Errorf("failed to set default config file path: %v", err))
		}
	}
	return &rootCmd.Command
}
