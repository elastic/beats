// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cmd

import (
	"flag"

	"github.com/spf13/pflag"

	cmd "github.com/elastic/beats/libbeat/cmd"
	"github.com/elastic/beats/x-pack/metricbeat/beater"
	"github.com/elastic/beats/x-pack/metricbeat/cmd/test"

	// import modules
	_ "github.com/elastic/beats/x-pack/metricbeat/include"
	_ "github.com/elastic/beats/x-pack/metricbeat/include/fields"
)

// Name of this beat
var Name = "metricbeat"

// RootCmd to handle beats cli
var RootCmd *cmd.BeatsRootCmd

func init() {
	var runFlags = pflag.NewFlagSet(Name, pflag.ExitOnError)
	runFlags.AddGoFlag(flag.CommandLine.Lookup("system.hostfs"))

	RootCmd = cmd.GenRootCmdWithRunFlags(Name, "", beater.DefaultCreator(), runFlags)
	RootCmd.AddCommand(cmd.GenModulesCmd(Name, "", buildModulesManager))
	RootCmd.TestCmd.AddCommand(test.GenTestModulesCmd(Name, ""))
}
