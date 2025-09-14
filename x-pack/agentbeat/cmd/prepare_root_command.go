// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !requirefips

package cmd

import (
	"github.com/spf13/cobra"

	auditbeat "github.com/elastic/beats/v7/x-pack/auditbeat/cmd"
	filebeat "github.com/elastic/beats/v7/x-pack/filebeat/cmd"
	heartbeat "github.com/elastic/beats/v7/x-pack/heartbeat/cmd"
	metricbeat "github.com/elastic/beats/v7/x-pack/metricbeat/cmd"
	osquerybeat "github.com/elastic/beats/v7/x-pack/osquerybeat/cmd"
	packetbeat "github.com/elastic/beats/v7/x-pack/packetbeat/cmd"
)

func prepareRootCommand() *cobra.Command {
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
		prepareCommand(metricbeat.Initialize()),
		prepareCommand(osquerybeat.RootCmd),
		prepareCommand(packetbeat.RootCmd),
	)

	return rootCmd
}
