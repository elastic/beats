// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/elastic/beats/v7/libbeat/cmd"
	"github.com/elastic/beats/v7/libbeat/ecs"
	"github.com/elastic/beats/v7/libbeat/processors"
	"github.com/elastic/beats/v7/libbeat/publisher/processing"
	"github.com/elastic/beats/v7/metricbeat/beater"
	mbcmd "github.com/elastic/beats/v7/metricbeat/cmd"
	"github.com/elastic/beats/v7/metricbeat/cmd/test"
	"github.com/elastic/beats/v7/x-pack/libbeat/management"
	"github.com/elastic/elastic-agent-libs/mapstr"

	// Register the includes.
	_ "github.com/elastic/beats/v7/x-pack/libbeat/include"
	_ "github.com/elastic/beats/v7/x-pack/metricbeat/include"

	// Import OSS modules.
	_ "github.com/elastic/beats/v7/metricbeat/include"
	_ "github.com/elastic/beats/v7/metricbeat/include/fields"
)

const (
	// Name of the beat
	Name = "metricbeat"
)

// withECSVersion is a modifier that adds ecs.version to events.
var withECSVersion = processing.WithFields(mapstr.M{
	"ecs": mapstr.M{
		"version": ecs.Version,
	},
})

func Initialize() *cmd.BeatsRootCmd {
	globalProcs, err := processors.NewPluginConfigFromList(defaultProcessors())
	if err != nil { // these are hard-coded, shouldn't fail
		panic(fmt.Errorf("error creating global processors: %w", err))
	}
	settings := mbcmd.MetricbeatSettings("")
	settings.ElasticLicensed = true
	settings.Processing = processing.MakeDefaultSupport(true, globalProcs, withECSVersion, processing.WithHost, processing.WithAgentMeta())
	rootCmd := cmd.GenRootCmdWithSettings(beater.DefaultCreator(), settings)
	rootCmd.AddCommand(cmd.GenModulesCmd(Name, "", mbcmd.BuildModulesManager))
	rootCmd.TestCmd.AddCommand(test.GenTestModulesCmd(Name, "", beater.DefaultTestModulesCreator()))
	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		management.ConfigTransform.SetTransform(metricbeatCfg)
	}
	return rootCmd
}

func defaultProcessors() []mapstr.M {
	// processors:
	//   - add_host_metadata: ~
	//   - add_cloud_metadata: ~
	//   - add_docker_metadata: ~
	//   - add_kubernetes_metadata: ~
	return []mapstr.M{
		{"add_host_metadata": nil},
		{"add_cloud_metadata": nil},
		{"add_docker_metadata": nil},
		{"add_kubernetes_metadata": nil},
	}
}
