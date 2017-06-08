package cmd

import (
	"flag"

	"github.com/spf13/pflag"

	"github.com/elastic/beats/metricbeat/beater"

	cmd "github.com/elastic/beats/libbeat/cmd"
)

// Name of this beat
var Name = "metricbeat"

// RootCmd to handle beats cli
var RootCmd *cmd.BeatsRootCmd

func init() {
	var runFlags = pflag.NewFlagSet(Name, pflag.ExitOnError)
	runFlags.AddGoFlag(flag.CommandLine.Lookup("system.hostfs"))

	RootCmd = cmd.GenRootCmdWithRunFlags(Name, "", beater.New, runFlags)
}
