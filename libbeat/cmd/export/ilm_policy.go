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

package export

import (
	"github.com/spf13/cobra"

	"github.com/elastic/beats/v7/libbeat/cmd/instance"
	"github.com/elastic/beats/v7/libbeat/idxmgmt"
	"github.com/elastic/beats/v7/libbeat/idxmgmt/lifecycle"
)

// GenGetILMPolicyCmd is the command used to export the ilm policy.
func GenGetILMPolicyCmd(settings instance.Settings) *cobra.Command {
	genTemplateConfigCmd := &cobra.Command{
		Use:   "ilm-policy",
		Short: "Export ILM policy",
		Run: func(cmd *cobra.Command, args []string) {
			version, _ := cmd.Flags().GetString("es.version")
			dir, _ := cmd.Flags().GetString("dir")

			if settings.ILM == nil {
				settings.ILM = lifecycle.StdSupport
			}
			b, err := instance.NewInitializedBeat(settings)
			if err != nil {
				fatalfInitCmd(err)
			}

			// the way this works, we decide to export ILM or DSL based on the user's config.
			// This means that if a user has no index management config, we'll default to ILM, regardless of what the user
			// is connected to. Might not be a problem since a user who doesn't have any custom lifecycle config has nothing to export?
			clientHandler, err := idxmgmt.NewFileClientHandler(newIdxmgmtClient(dir, version), b.Info, b.Config.LifecycleConfig)
			if err != nil {
				fatalf("error creating file handler: %s", err)
			}
			idxManager := b.IdxSupporter.Manager(clientHandler, idxmgmt.BeatsAssets(b.Fields))
			if err := idxManager.Setup(idxmgmt.LoadModeDisabled, idxmgmt.LoadModeForce); err != nil {
				fatalf("Error exporting ilm-policy: %+v.", err)
			}
		},
	}

	genTemplateConfigCmd.Flags().String("es.version", settings.Version, "Elasticsearch version")
	genTemplateConfigCmd.Flags().String("dir", "", "Specify directory for printing policy files. By default policies are printed to stdout.")

	return genTemplateConfigCmd
}
