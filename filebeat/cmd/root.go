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

	"github.com/elastic/beats/filebeat/beater"

	cmd "github.com/elastic/beats/libbeat/cmd"
	"github.com/elastic/beats/libbeat/cmd/instance"

	// Import the script processor.
	_ "github.com/elastic/beats/libbeat/processors/script"
)

// Name of this beat
var Name = "filebeat"

// RootCmd to handle beats cli
var RootCmd *cmd.BeatsRootCmd

func init() {
	var runFlags = pflag.NewFlagSet(Name, pflag.ExitOnError)
	runFlags.AddGoFlag(flag.CommandLine.Lookup("once"))
	runFlags.AddGoFlag(flag.CommandLine.Lookup("modules"))
	RootCmd = cmd.GenRootCmdWithSettings(beater.New, instance.Settings{RunFlags: runFlags, Name: Name})
	RootCmd.PersistentFlags().AddGoFlag(flag.CommandLine.Lookup("M"))
	RootCmd.TestCmd.Flags().AddGoFlag(flag.CommandLine.Lookup("modules"))
	RootCmd.SetupCmd.Flags().AddGoFlag(flag.CommandLine.Lookup("modules"))
	RootCmd.AddCommand(cmd.GenModulesCmd(Name, "", buildModulesManager))
	RootCmd.AddCommand(genGenerateCmd())
}
