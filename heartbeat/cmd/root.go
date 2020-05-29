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

	"github.com/elastic/beats/v7/heartbeat/beater"

	// include all heartbeat specific autodiscovery builders
	_ "github.com/elastic/beats/v7/heartbeat/autodiscover/builder/hints"

	// register default heartbeat monitors
	_ "github.com/elastic/beats/v7/heartbeat/monitors/defaults"
	cmd "github.com/elastic/beats/v7/libbeat/cmd"
	"github.com/elastic/beats/v7/libbeat/cmd/instance"
	"github.com/elastic/beats/v7/libbeat/publisher/processing"
)

// Name of this beat
var Name = "heartbeat"

// RootCmd to handle beats cli
var RootCmd *cmd.BeatsRootCmd

func init() {
	settings := instance.Settings{
		Name:          Name,
		Processing:    processing.MakeDefaultSupport(true, processing.WithECS, processing.WithAgentMeta()),
		HasDashboards: false,
	}
	RootCmd = cmd.GenRootCmdWithSettings(beater.New, settings)

	// remove dashboard from export commands
	for _, cmd := range RootCmd.ExportCmd.Commands() {
		if cmd.Name() == "dashboard" {
			RootCmd.ExportCmd.RemoveCommand(cmd)
		}
	}

	// only add defined flags to setup command
	setup := RootCmd.SetupCmd
	setup.Short = "Setup Elasticsearch index template and pipelines"
	setup.Long = `This command does initial setup of the environment:
 * Index mapping template in Elasticsearch to ensure fields are mapped.
 * ILM Policy
`
	setup.ResetFlags()
	setup.Flags().Bool(cmd.IndexManagementKey, false, "Setup all components related to Elasticsearch index management, including template, ilm policy and rollover alias")
	setup.Flags().MarkDeprecated(cmd.TemplateKey, fmt.Sprintf("use --%s instead", cmd.IndexManagementKey))
	setup.Flags().MarkDeprecated(cmd.ILMPolicyKey, fmt.Sprintf("use --%s instead", cmd.IndexManagementKey))
	setup.Flags().Bool(cmd.TemplateKey, false, "Setup index template")
	setup.Flags().Bool(cmd.ILMPolicyKey, false, "Setup ILM policy")
}
