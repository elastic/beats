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

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/cmd/instance"
)

const (
	//TemplateKey used for defining template in setup cmd
	TemplateKey = "template"
	//DashboardKey used for registering dashboards in setup cmd
	DashboardKey = "dashboards"
	//MachineLearningKey used for registering ml jobs in setup cmd
	MachineLearningKey = "machine-learning"
	//PipelineKey used for registering pipelines in setup cmd
	PipelineKey = "pipelines"
	//ILMPolicyKey used for registering ilm in setup cmd
	ILMPolicyKey = "ilm-policy"
)

func genSetupCmd(settings instance.Settings, beatCreator beat.Creator) *cobra.Command {
	setup := cobra.Command{
		Use:   "setup",
		Short: "Setup index template, dashboards and ML jobs",
		Long: `This command does initial setup of the environment:

 * Index mapping template in Elasticsearch to ensure fields are mapped.
 * Kibana dashboards (where available).
 * ML jobs (where available).
 * Ingest pipelines (where available).
 * ILM policy (for Elasticsearch 6.5 and newer).
`,
		Run: func(cmd *cobra.Command, args []string) {
			beat, err := instance.NewBeat(settings.Name, settings.IndexPrefix, settings.Version)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error initializing beat: %s\n", err)
				os.Exit(1)
			}

			var registeredFlags = map[string]bool{
				TemplateKey:        false,
				DashboardKey:       false,
				MachineLearningKey: false,
				PipelineKey:        false,
				ILMPolicyKey:       false,
			}
			var setupAll = true

			// create collection with registered flags and their values
			for k := range registeredFlags {
				val, err := cmd.Flags().GetBool(k)
				//if flag is not registered, an error is thrown
				if err != nil {
					delete(registeredFlags, k)
					continue
				}
				registeredFlags[k] = val

				//if any flag is set via cmd line then only this flag should be run
				if val {
					setupAll = false
				}
			}

			//create the struct to pass on
			var s = instance.SetupSettings{}
			for k, v := range registeredFlags {
				if setupAll || v {
					switch k {
					case TemplateKey:
						s.Template = true
					case DashboardKey:
						s.Dashboard = true
					case MachineLearningKey:
						s.MachineLearning = true
					case PipelineKey:
						s.Pipeline = true
					case ILMPolicyKey:
						s.ILMPolicy = true
					}
				}
			}

			if err = beat.Setup(settings, beatCreator, s); err != nil {
				os.Exit(1)
			}
		},
	}

	setup.Flags().Bool(TemplateKey, false, "Setup index template")
	setup.Flags().Bool(DashboardKey, false, "Setup dashboards")
	setup.Flags().Bool(MachineLearningKey, false, "Setup machine learning job configurations")
	setup.Flags().Bool(PipelineKey, false, "Setup Ingest pipelines")
	setup.Flags().Bool(ILMPolicyKey, false, "Setup ILM policy")

	return &setup
}
