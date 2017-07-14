package cmd

import (
	"flag"

	"github.com/spf13/pflag"

	// import modules
	_ "github.com/elastic/beats/metricbeat/include"

	cmd "github.com/elastic/beats/libbeat/cmd"
	"github.com/elastic/beats/metricbeat/beater"
)

// Name of this beat
var Name = "metricbeat"

// RootCmd to handle beats cli
var RootCmd *cmd.BeatsRootCmd

func init() {
	var runFlags = pflag.NewFlagSet(Name, pflag.ExitOnError)
	runFlags.AddGoFlag(flag.CommandLine.Lookup("system.hostfs"))

	RootCmd = cmd.GenRootCmdWithRunFlags(Name, "", beater.New, runFlags)
	RootCmd.AddCommand(cmd.GenModulesCmd(Name, "", buildModulesManager))
}
