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
	"github.com/elastic/beats/v7/libbeat/cmd"
	"github.com/elastic/beats/v7/libbeat/cmd/instance"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/ecs"
	"github.com/elastic/beats/v7/libbeat/publisher/processing"
	"github.com/elastic/beats/v7/winlogbeat/beater"

	// Register fields.
	_ "github.com/elastic/beats/v7/winlogbeat/include"

	// Import processors and supporting modules.
	_ "github.com/elastic/beats/v7/libbeat/processors/script"
	_ "github.com/elastic/beats/v7/libbeat/processors/timestamp"
)

const (
	// Name of this beat.
	Name = "winlogbeat"
)

// withECSVersion is a modifier that adds ecs.version to events.
var withECSVersion = processing.WithFields(common.MapStr{
	"ecs": common.MapStr{
		"version": ecs.Version,
	},
})

// RootCmd to handle beats cli
var RootCmd *cmd.BeatsRootCmd

// WinlogbeatSettings contains the default settings for winlogbeat
func WinlogbeatSettings() instance.Settings {
	return instance.Settings{
		Name:          Name,
		HasDashboards: true,
		Processing:    processing.MakeDefaultSupport(true, withECSVersion, processing.WithAgentMeta()),
	}
}

// Initialize initializes the entrypoint commands for packetbeat
func Initialize(settings instance.Settings) *cmd.BeatsRootCmd {
	return cmd.GenRootCmdWithSettings(beater.New, settings)
}

func init() {
	RootCmd = Initialize(WinlogbeatSettings())
}
