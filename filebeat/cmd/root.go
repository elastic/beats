package cmd

import (
	"flag"

	"github.com/spf13/pflag"

	cmd "github.com/elastic/beats/libbeat/cmd"

	"github.com/elastic/beats/filebeat/beater"
	"github.com/elastic/beats/filebeat/cmd/multiline"
)

// Name of this beat
var Name = "filebeat"

// RootCmd to handle beats cli
var RootCmd *cmd.BeatsRootCmd

func init() {
	var runFlags = pflag.NewFlagSet(Name, pflag.ExitOnError)
	runFlags.AddGoFlag(flag.CommandLine.Lookup("once"))
	runFlags.AddGoFlag(flag.CommandLine.Lookup("modules"))

	RootCmd = cmd.GenRootCmdWithRunFlags(Name, "", beater.New, runFlags)
	RootCmd.PersistentFlags().AddGoFlag(flag.CommandLine.Lookup("M"))
	RootCmd.TestCmd.Flags().AddGoFlag(flag.CommandLine.Lookup("modules"))
	RootCmd.TestCmd.AddCommand(multiline.Command)
	RootCmd.SetupCmd.Flags().AddGoFlag(flag.CommandLine.Lookup("modules"))
	RootCmd.AddCommand(cmd.GenModulesCmd(Name, "", buildModulesManager))
}
