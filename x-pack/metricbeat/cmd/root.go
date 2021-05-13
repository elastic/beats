// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cmd

import (
	"flag"

	"github.com/spf13/pflag"

	"github.com/elastic/beats/v7/libbeat/cmd"
	"github.com/elastic/beats/v7/libbeat/cmd/instance"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/publisher/processing"
	"github.com/elastic/beats/v7/metricbeat/beater"
	mbcmd "github.com/elastic/beats/v7/metricbeat/cmd"
	"github.com/elastic/beats/v7/metricbeat/cmd/test"
	xpackcmd "github.com/elastic/beats/v7/x-pack/libbeat/cmd"

	// Register the includes.
	_ "github.com/elastic/beats/v7/x-pack/metricbeat/include"

	// Import OSS modules.
	_ "github.com/elastic/beats/v7/metricbeat/include"
	_ "github.com/elastic/beats/v7/metricbeat/include/fields"
)

const (
	// Name of the beat
	Name = "metricbeat"

	// ecsVersion specifies the version of ECS that this beat is implementing.
	ecsVersion = "1.9.0"
)

// RootCmd to handle beats cli
var RootCmd *cmd.BeatsRootCmd

// withECSVersion is a modifier that adds ecs.version to events.
var withECSVersion = processing.WithFields(common.MapStr{
	"ecs": common.MapStr{
		"version": ecsVersion,
	},
})

func init() {
	var runFlags = pflag.NewFlagSet(Name, pflag.ExitOnError)
	runFlags.AddGoFlag(flag.CommandLine.Lookup("system.hostfs"))
	settings := instance.Settings{
		RunFlags:        runFlags,
		Name:            Name,
		HasDashboards:   true,
		ElasticLicensed: true,
		Processing:      processing.MakeDefaultSupport(true, withECSVersion, processing.WithHost, processing.WithAgentMeta()),
	}
	RootCmd = cmd.GenRootCmdWithSettings(beater.DefaultCreator(), settings)
	RootCmd.AddCommand(cmd.GenModulesCmd(Name, "", mbcmd.BuildModulesManager))
	RootCmd.TestCmd.AddCommand(test.GenTestModulesCmd(Name, "", beater.DefaultTestModulesCreator()))
	xpackcmd.AddXPack(RootCmd, Name)
}
