// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cmd

import (
	"flag"

	"github.com/spf13/pflag"

	"github.com/elastic/beats/libbeat/cmd"
	"github.com/elastic/beats/libbeat/cmd/instance"
	"github.com/elastic/beats/metricbeat/beater"
	mbcmd "github.com/elastic/beats/metricbeat/cmd"
	"github.com/elastic/beats/metricbeat/cmd/test"
	xpackcmd "github.com/elastic/beats/x-pack/libbeat/cmd"

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

func init() {
	var runFlags = pflag.NewFlagSet(Name, pflag.ExitOnError)
	runFlags.AddGoFlag(flag.CommandLine.Lookup("system.hostfs"))
	settings := instance.Settings{
		RunFlags:      runFlags,
		Name:          Name,
		HasDashboards: true,
	}
	RootCmd = cmd.GenRootCmdWithSettings(beater.DefaultCreator(), settings)
	RootCmd.AddCommand(cmd.GenModulesCmd(Name, "", mbcmd.BuildModulesManager))
	RootCmd.TestCmd.AddCommand(test.GenTestModulesCmd(Name, "", beater.DefaultTestModulesCreator()))
	xpackcmd.AddXPack(RootCmd, Name)
}
