package cmd

import (
	"flag"

	"github.com/spf13/pflag"

	cmd "github.com/elastic/beats/libbeat/cmd"
	"github.com/elastic/beats/metricbeat/beater"
	"github.com/elastic/beats/metricbeat/cmd/test"

	// import modules
	_ "github.com/elastic/beats/metricbeat/include"
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
