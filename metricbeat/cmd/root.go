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

	"github.com/elastic/beats/v7/libbeat/cmd"
	"github.com/elastic/beats/v7/libbeat/cmd/instance"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/publisher/processing"

	"github.com/elastic/beats/v7/metricbeat/beater"
	"github.com/elastic/beats/v7/metricbeat/cmd/test"

	// import modules
	_ "github.com/elastic/beats/v7/metricbeat/include"
	_ "github.com/elastic/beats/v7/metricbeat/include/fields"

	// Import processors.
	_ "github.com/elastic/beats/v7/libbeat/processors/script"
)

const (
	// Name of the beat
	Name = "metricbeat"

	// ecsVersion specifies the version of ECS that this beat is implementing.
	ecsVersion = "1.9.0"
)

// RootCmd to handle beats cli
var RootCmd *cmd.BeatsRootCmd

// withECSVersion is a modifier that adds ecs.version to events.
var withECSVersion = processing.WithFields(common.MapStr{
	"ecs": common.MapStr{
		"version": ecsVersion,
	},
})

// MetricbeatSettings contains the default settings for metricbeat
func MetricbeatSettings() instance.Settings {
	var runFlags = pflag.NewFlagSet(Name, pflag.ExitOnError)
	runFlags.AddGoFlag(flag.CommandLine.Lookup("system.hostfs"))
	return instance.Settings{
		RunFlags:      runFlags,
		Name:          Name,
		HasDashboards: true,
		Processing:    processing.MakeDefaultSupport(true, withECSVersion, processing.WithHost, processing.WithAgentMeta()),
	}
}

// Initialize initializes the entrypoint commands for metricbeat
func Initialize(settings instance.Settings) *cmd.BeatsRootCmd {
	rootCmd := cmd.GenRootCmdWithSettings(beater.DefaultCreator(), settings)
	rootCmd.AddCommand(cmd.GenModulesCmd(Name, "", BuildModulesManager))
	rootCmd.TestCmd.AddCommand(test.GenTestModulesCmd(Name, "", beater.DefaultTestModulesCreator()))
	return rootCmd
}

func init() {
	RootCmd = Initialize(MetricbeatSettings())
}
