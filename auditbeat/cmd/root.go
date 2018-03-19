package cmd

import (
	"github.com/spf13/pflag"

	"github.com/elastic/beats/metricbeat/beater"

	"github.com/elastic/beats/auditbeat/core"
	cmd "github.com/elastic/beats/libbeat/cmd"
	"github.com/elastic/beats/metricbeat/mb/module"
)

// Name of the beat (auditbeat).
const Name = "auditbeat"

// RootCmd for running auditbeat.
var RootCmd *cmd.BeatsRootCmd

func init() {
	create := beater.Creator(
		beater.WithModuleOptions(
			module.WithEventModifier(core.AddDatasetToEvent),
		),
	)
	var runFlags = pflag.NewFlagSet(Name, pflag.ExitOnError)
	RootCmd = cmd.GenRootCmdWithRunFlags(Name, "", create, runFlags)
}
