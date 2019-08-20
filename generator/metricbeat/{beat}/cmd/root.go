package cmd

import (
	cmd "github.com/elastic/beats/libbeat/cmd"
	"github.com/elastic/beats/libbeat/cmd/instance"
	"github.com/elastic/beats/metricbeat/beater"
	"github.com/elastic/beats/metricbeat/cmd/test"
	"github.com/elastic/beats/metricbeat/mb/module"
)

// Name of this beat
var Name = "examplebeat"

// RootCmd to handle beats cli
var RootCmd *cmd.BeatsRootCmd

var (
	// Use a customized instance of Metricbeat where startup delay has
	// been disabled to workaround the fact that Modules() will return
	// the static modules (not the dynamic ones) with a start delay.
	testModulesCreator = beater.Creator(
		beater.WithModuleOptions(
			module.WithMetricSetInfo(),
			module.WithMaxStartDelay(0),
		),
	)
)

func init() {

	RootCmd = cmd.GenRootCmdWithSettings(beater.DefaultCreator(), instance.Settings{Name: Name})
	RootCmd.AddCommand(cmd.GenModulesCmd(Name, "", BuildModulesManager))
	RootCmd.TestCmd.AddCommand(test.GenTestModulesCmd(Name, "", testModulesCreator))
}
