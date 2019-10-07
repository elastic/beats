// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cmd

import (
	"flag"

	"github.com/spf13/pflag"

	cmd "github.com/elastic/beats/libbeat/cmd"
	"github.com/elastic/beats/libbeat/cmd/instance"
	"github.com/elastic/beats/metricbeat/beater"
	mbcmd "github.com/elastic/beats/metricbeat/cmd"
	"github.com/elastic/beats/metricbeat/cmd/test"
	"github.com/elastic/beats/metricbeat/mb/module"
	xpackcmd "github.com/elastic/beats/x-pack/libbeat/cmd"
	xpackbeater "github.com/elastic/beats/x-pack/metricbeat/beater"

	// Register the includes.
	_ "github.com/elastic/beats/x-pack/metricbeat/include"

	// Import OSS modules.
	_ "github.com/elastic/beats/metricbeat/include"
	_ "github.com/elastic/beats/metricbeat/include/fields"
)

// Name of this beat
var Name = "metricbeat"

// RootCmd to handle beats cli
var RootCmd *cmd.BeatsRootCmd

var (
	rootCreator = beater.Creator(
		xpackbeater.WithLightModules(),
		beater.WithModuleOptions(
			module.WithMetricSetInfo(),
			module.WithServiceName(),
		),
	)

	// Use a customized instance of Metricbeat where startup delay has
	// been disabled to workaround the fact that Modules() will return
	// the static modules (not the dynamic ones) with a start delay.
	testModulesCreator = beater.Creator(
		xpackbeater.WithLightModules(),
		beater.WithModuleOptions(
			module.WithMetricSetInfo(),
			module.WithMaxStartDelay(0),
		),
	)
)

func init() {
	var runFlags = pflag.NewFlagSet(Name, pflag.ExitOnError)
	runFlags.AddGoFlag(flag.CommandLine.Lookup("system.hostfs"))
	RootCmd = cmd.GenRootCmdWithSettings(rootCreator, instance.Settings{RunFlags: runFlags, Name: Name})
	RootCmd.AddCommand(cmd.GenModulesCmd(Name, "", mbcmd.BuildModulesManager))
	RootCmd.TestCmd.AddCommand(test.GenTestModulesCmd(Name, "", testModulesCreator))
	xpackcmd.AddXPack(RootCmd, Name)
}
