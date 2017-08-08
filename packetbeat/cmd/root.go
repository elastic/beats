package cmd

import (
	"flag"

	"github.com/spf13/pflag"

	// import protocol modules
	_ "github.com/elastic/beats/packetbeat/include"

	cmd "github.com/elastic/beats/libbeat/cmd"
	"github.com/elastic/beats/packetbeat/beater"
)

// Name of this beat
var Name = "packetbeat"

// RootCmd to handle beats cli
var RootCmd *cmd.BeatsRootCmd

func init() {
	var runFlags = pflag.NewFlagSet(Name, pflag.ExitOnError)
	runFlags.AddGoFlag(flag.CommandLine.Lookup("I"))
	runFlags.AddGoFlag(flag.CommandLine.Lookup("t"))
	runFlags.AddGoFlag(flag.CommandLine.Lookup("O"))
	runFlags.AddGoFlag(flag.CommandLine.Lookup("l"))
	runFlags.AddGoFlag(flag.CommandLine.Lookup("dump"))

	RootCmd = cmd.GenRootCmdWithRunFlags(Name, "", beater.New, runFlags)
	RootCmd.AddCommand(genDevicesCommand())
}
