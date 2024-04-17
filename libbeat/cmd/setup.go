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
	"os"

	"github.com/spf13/cobra"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/cmd/instance"
)

const (
	// DashboardKey used for registering dashboards in setup cmd
	DashboardKey = "dashboards"
	// PipelineKey used for registering pipelines in setup cmd
	PipelineKey = "pipelines"
	// IndexManagementKey used for loading all components related to ES index management in setup cmd
	IndexManagementKey = "index-management"
	// EnableAllFilesetsKey enables all modules and filesets regardless of config
	EnableAllFilesetsKey = "enable-all-filesets"
	// ForceEnableModuleFilesets enables all the filesets contained
	// in the modules that have been explicitly enabled.  The
	// requires "modules" to be used.
	ForceEnableModuleFilesets = "force-enable-module-filesets"
)

func genSetupCmd(settings instance.Settings, beatCreator beat.Creator) *cobra.Command {
	setup := cobra.Command{
		Use:   "setup",
		Short: "Setup index template, dashboards and ML jobs",
		Long: `This command does initial setup of the environment:

 * Index mapping template in Elasticsearch to ensure fields are mapped.
 * Kibana dashboards (where available).
 * Ingest pipelines (where available).
 * ILM policy (for Elasticsearch 6.5 and newer).
`,
		Run: func(cmd *cobra.Command, args []string) {
			beat, err := instance.NewBeat(settings.Name, settings.IndexPrefix, settings.Version, settings.ElasticLicensed, settings.Initialize)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error initializing beat: %s\n", err)
				os.Exit(1)
			}

			registeredFlags := map[string]bool{
				DashboardKey:              false,
				PipelineKey:               false,
				IndexManagementKey:        false,
				EnableAllFilesetsKey:      false,
				ForceEnableModuleFilesets: false,
			}
			setupAll := true

			// create collection with registered flags and their values
			for k := range registeredFlags {
				val, err := cmd.Flags().GetBool(k)
				// if flag is not registered, an error is thrown
				if err != nil {
					delete(registeredFlags, k)
					continue
				}
				registeredFlags[k] = val

				// if any flag is set via cmd line then only this flag should be run
				if val {
					setupAll = false
				}
			}

			// create the struct to pass on
			s := instance.SetupSettings{}
			for k, v := range registeredFlags {
				if setupAll || v {
					switch k {
					case DashboardKey:
						s.Dashboard = true
					case PipelineKey:
						s.Pipeline = true
					case IndexManagementKey:
						s.IndexManagement = true
					case EnableAllFilesetsKey:
						s.EnableAllFilesets = true
					case ForceEnableModuleFilesets:
						s.ForceEnableModuleFilesets = true
					}
				}
			}

			if err = beat.Setup(settings, beatCreator, s); err != nil {
				os.Exit(1)
			}
		},
	}

	setup.Flags().Bool(DashboardKey, false, "Setup dashboards")
	setup.Flags().Bool(PipelineKey, false, "Setup Ingest pipelines")
	setup.Flags().Bool(IndexManagementKey, false,
		"Setup all components related to Elasticsearch index management, including template, ilm policy and rollover alias")
	setup.Flags().Bool("enable-all-filesets", false, "Behave as if all modules and filesets had been enabled")
	setup.Flags().Bool("force-enable-module-filesets", false, "Behave as if all filesets, within enabled modules, are enabled")

	return &setup
}
