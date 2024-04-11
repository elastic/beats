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

	"github.com/elastic/beats/v7/filebeat/beater"
	"github.com/elastic/beats/v7/filebeat/fileset"
	"github.com/elastic/beats/v7/filebeat/include"
	"github.com/elastic/beats/v7/filebeat/input"
	"github.com/elastic/beats/v7/libbeat/cmd"
	"github.com/elastic/beats/v7/libbeat/cmd/instance"

	// Import processors.
	_ "github.com/elastic/beats/v7/libbeat/processors/cache"
	_ "github.com/elastic/beats/v7/libbeat/processors/timestamp"
)

// Name of this beat
const Name = "filebeat"

// RootCmd to handle beats cli
var RootCmd *cmd.BeatsRootCmd

// FilebeatSettings contains the default settings for filebeat
func FilebeatSettings() instance.Settings {
	runFlags := pflag.NewFlagSet(Name, pflag.ExitOnError)
	runFlags.AddGoFlag(flag.CommandLine.Lookup("once"))
	runFlags.AddGoFlag(flag.CommandLine.Lookup("modules"))
	return instance.Settings{
		RunFlags:      runFlags,
		Name:          Name,
		HasDashboards: true,
		Initialize: []func(){
			include.InitializeModule,
			fileset.RegisterMonitoringModules,
			input.RegisterMonitoringInputs,
		},
	}
}

// Filebeat build the beat root command for executing filebeat and it's subcommands.
func Filebeat(inputs beater.PluginFactory, settings instance.Settings) *cmd.BeatsRootCmd {
	command := cmd.GenRootCmdWithSettings(beater.New(inputs), settings)
	command.PersistentFlags().AddGoFlag(flag.CommandLine.Lookup("M"))
	command.TestCmd.Flags().AddGoFlag(flag.CommandLine.Lookup("modules"))
	command.SetupCmd.Flags().AddGoFlag(flag.CommandLine.Lookup("modules"))
	command.AddCommand(cmd.GenModulesCmd(Name, "", buildModulesManager))
	command.AddCommand(genGenerateCmd())
	return command
}
