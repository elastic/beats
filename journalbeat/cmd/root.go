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
	"github.com/elastic/beats/v7/journalbeat/beater"

	"github.com/elastic/beats/v7/libbeat/cmd"
	"github.com/elastic/beats/v7/libbeat/cmd/instance"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/publisher/processing"

	// Import processors.
	_ "github.com/elastic/beats/v7/libbeat/processors/script"
	_ "github.com/elastic/beats/v7/libbeat/processors/timestamp"
)

const (
	// Name of this beat.
	Name = "journalbeat"

	// ecsVersion specifies the version of ECS that Winlogbeat is implementing.
	ecsVersion = "1.10.0"
)

// withECSVersion is a modifier that adds ecs.version to events.
var withECSVersion = processing.WithFields(common.MapStr{
	"ecs": common.MapStr{
		"version": ecsVersion,
	},
})

// RootCmd to handle beats cli
var RootCmd *cmd.BeatsRootCmd

// JournalbeatSettings contains the default settings for journalbeat
func JournalbeatSettings() instance.Settings {
	return instance.Settings{
		Name:          Name,
		HasDashboards: false,
		Processing:    processing.MakeDefaultSupport(true, withECSVersion, processing.WithHost, processing.WithAgentMeta()),
	}
}

// Initialize initializes the entrypoint commands for journalbeat
func Initialize(settings instance.Settings) *cmd.BeatsRootCmd {
	return cmd.GenRootCmdWithSettings(beater.New, settings)
}

func init() {
	RootCmd = Initialize(JournalbeatSettings())
}
