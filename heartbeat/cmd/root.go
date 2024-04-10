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
	"fmt"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/elastic/beats/v7/heartbeat/autodiscover/builder/hints"
	"github.com/elastic/beats/v7/heartbeat/beater"
	"github.com/elastic/beats/v7/libbeat/autodiscover"
	cmd "github.com/elastic/beats/v7/libbeat/cmd"
	"github.com/elastic/beats/v7/libbeat/cmd/instance"
	"github.com/elastic/beats/v7/libbeat/ecs"
	"github.com/elastic/beats/v7/libbeat/publisher/processing"

	// Import packages that need to register themselves.
	_ "github.com/elastic/beats/v7/heartbeat/monitors/active/http"
	_ "github.com/elastic/beats/v7/heartbeat/monitors/active/icmp"
	_ "github.com/elastic/beats/v7/heartbeat/monitors/active/tcp"

	// include all heartbeat specific autodiscovery builders
	_ "github.com/elastic/beats/v7/heartbeat/autodiscover/builder/hints"
)

const (
	// Name of the beat
	Name = "heartbeat"
)

// RootCmd to handle beats cli
var RootCmd *cmd.BeatsRootCmd

// withECSVersion is a modifier that adds ecs.version to events.
var withECSVersion = processing.WithFields(mapstr.M{
	"ecs": mapstr.M{
		"version": ecs.Version,
	},
})

// HeartbeatSettings contains the default settings for heartbeat
func HeartbeatSettings() instance.Settings {
	return instance.Settings{
		Name:          Name,
		Processing:    processing.MakeDefaultSupport(true, nil, withECSVersion, processing.WithAgentMeta()),
		HasDashboards: false,
		InitFunc: func() {
			err := autodiscover.Registry.AddBuilder("hints", hints.NewHeartbeatHints)
			if err != nil {
				logp.Error(fmt.Errorf("could not add `hints` builder"))
			}
		},
	}
}

// Initialize initializes the entrypoint commands for heartbeat
func Initialize(settings instance.Settings) *cmd.BeatsRootCmd {
	rootCmd := cmd.GenRootCmdWithSettings(beater.New, settings)

	// remove dashboard from export commands
	for _, cmd := range rootCmd.ExportCmd.Commands() {
		if cmd.Name() == "dashboard" {
			rootCmd.ExportCmd.RemoveCommand(cmd)
		}
	}

	// only add defined flags to setup command
	setup := rootCmd.SetupCmd
	setup.Short = "Setup Elasticsearch index template and pipelines"
	setup.Long = `This command does initial setup of the environment:
 * Index mapping template in Elasticsearch to ensure fields are mapped.
 * ILM Policy
`
	setup.ResetFlags()
	setup.Flags().Bool(cmd.IndexManagementKey, false, "Setup all components related to Elasticsearch index management, including template, ilm policy and rollover alias")

	return rootCmd
}

func init() {
	RootCmd = Initialize(HeartbeatSettings())
}
