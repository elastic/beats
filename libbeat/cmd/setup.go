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

func genSetupCmd(name, idxPrefix, version string, beatCreator beat.Creator) *cobra.Command {
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
			beat, err := instance.NewBeat(name, idxPrefix, version)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error initializing beat: %s\n", err)
				os.Exit(1)
			}

			template, _ := cmd.Flags().GetBool("template")
			dashboards, _ := cmd.Flags().GetBool("dashboards")
			machineLearning, _ := cmd.Flags().GetBool("machine-learning")
			pipelines, _ := cmd.Flags().GetBool("pipelines")
			policy, _ := cmd.Flags().GetBool("ilm-policy")

			// No flags: setup all
			if !template && !dashboards && !machineLearning && !pipelines && !policy {
				template = true
				dashboards = true
				machineLearning = true
				pipelines = true
				policy = false
			}

			if err = beat.Setup(beatCreator, template, dashboards, machineLearning, pipelines, policy); err != nil {
				os.Exit(1)
			}
		},
	}

	setup.Flags().Bool("template", false, "Setup index template")
	setup.Flags().Bool("dashboards", false, "Setup dashboards")
	setup.Flags().Bool("machine-learning", false, "Setup machine learning job configurations")
	setup.Flags().Bool("pipelines", false, "Setup Ingest pipelines")
	setup.Flags().Bool("ilm-policy", false, "Setup ILM policy")

	return &setup
}
