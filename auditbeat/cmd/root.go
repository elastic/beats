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
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/elastic/beats/v7/auditbeat/core"
	"github.com/elastic/beats/v7/libbeat/cmd"
	"github.com/elastic/beats/v7/libbeat/cmd/instance"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/publisher/processing"
	"github.com/elastic/beats/v7/metricbeat/beater"
	"github.com/elastic/beats/v7/metricbeat/mb/module"
)

const (
	// Name of the beat (auditbeat).
	Name = "auditbeat"

	// ecsVersion specifies the version of ECS that Auditbeat is implementing.
	ecsVersion = "1.10.0"
)

// RootCmd for running auditbeat.
var RootCmd *cmd.BeatsRootCmd

// ShowCmd to display extra information.
var ShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show modules information",
}

// withECSVersion is a modifier that adds ecs.version to events.
var withECSVersion = processing.WithFields(common.MapStr{
	"ecs": common.MapStr{
		"version": ecsVersion,
	},
})

// AuditbeatSettings contains the default settings for auditbeat
func AuditbeatSettings() instance.Settings {
	var runFlags = pflag.NewFlagSet(Name, pflag.ExitOnError)
	return instance.Settings{
		RunFlags:      runFlags,
		Name:          Name,
		HasDashboards: true,
		Processing:    processing.MakeDefaultSupport(true, withECSVersion, processing.WithHost, processing.WithAgentMeta()),
	}
}

// Initialize initializes the entrypoint commands for journalbeat
func Initialize(settings instance.Settings) *cmd.BeatsRootCmd {
	create := beater.Creator(
		beater.WithModuleOptions(
			module.WithEventModifier(core.AddDatasetToEvent),
		),
	)
	rootCmd := cmd.GenRootCmdWithSettings(create, settings)
	rootCmd.AddCommand(ShowCmd)
	return rootCmd
}

func init() {
	RootCmd = Initialize(AuditbeatSettings())
}
