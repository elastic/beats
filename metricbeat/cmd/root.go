// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package cmd

import (
	"flag"

	"github.com/spf13/pflag"

	cmd "github.com/elastic/beats/libbeat/cmd"
	"github.com/elastic/beats/libbeat/cmd/instance"
	"github.com/elastic/beats/metricbeat/beater"
	"github.com/elastic/beats/metricbeat/cmd/test"
	"github.com/elastic/beats/metricbeat/mb/module"

	// import modules
	_ "github.com/elastic/beats/metricbeat/include"
	_ "github.com/elastic/beats/metricbeat/include/fields"
)

// Name of this beat
var Name = "metricbeat"

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
	var runFlags = pflag.NewFlagSet(Name, pflag.ExitOnError)
	runFlags.AddGoFlag(flag.CommandLine.Lookup("system.hostfs"))
	RootCmd = cmd.GenRootCmdWithSettings(beater.DefaultCreator(), instance.Settings{RunFlags: runFlags, Name: Name})
	RootCmd.AddCommand(cmd.GenModulesCmd(Name, "", BuildModulesManager))
	RootCmd.TestCmd.AddCommand(test.GenTestModulesCmd(Name, "", testModulesCreator))
}
