// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cmd

import (
	"flag"
	"fmt"

	"github.com/spf13/pflag"

	"github.com/elastic/beats/v7/libbeat/cmd"
	"github.com/elastic/beats/v7/libbeat/cmd/instance"
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

// RootCmd to handle beats cli
var RootCmd *cmd.BeatsRootCmd

// withECSVersion is a modifier that adds ecs.version to events.
var withECSVersion = processing.WithFields(mapstr.M{
	"ecs": mapstr.M{
		"version": ecs.Version,
	},
})

func init() {
	management.ConfigTransform.SetTransform(metricbeatCfg)
	var runFlags = pflag.NewFlagSet(Name, pflag.ExitOnError)
	runFlags.AddGoFlag(flag.CommandLine.Lookup("system.hostfs"))
	globalProcs, err := processors.NewPluginConfigFromList(defaultProcessors())
	if err != nil { // these are hard-coded, shouldn't fail
		panic(fmt.Errorf("error creating global processors: %w", err))
	}
	settings := instance.Settings{
		RunFlags:        runFlags,
		Name:            Name,
		HasDashboards:   true,
		ElasticLicensed: true,
		Processing:      processing.MakeDefaultSupport(true, globalProcs, withECSVersion, processing.WithHost, processing.WithAgentMeta()),
	}
	RootCmd = cmd.GenRootCmdWithSettings(beater.DefaultCreator(), settings)
	RootCmd.AddCommand(cmd.GenModulesCmd(Name, "", mbcmd.BuildModulesManager))
	RootCmd.TestCmd.AddCommand(test.GenTestModulesCmd(Name, "", beater.DefaultTestModulesCreator()))
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
