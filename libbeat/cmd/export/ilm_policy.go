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

	"github.com/elastic/beats/libbeat/cmd/instance"
	"github.com/elastic/beats/libbeat/idxmgmt"
)

// GenGetILMPolicyCmd is the command used to export the ilm policy.
func GenGetILMPolicyCmd(settings instance.Settings) *cobra.Command {
	genTemplateConfigCmd := &cobra.Command{
		Use:   "ilm-policy",
		Short: "Export ILM policy",
		Run: func(cmd *cobra.Command, args []string) {
			b, err := instance.NewInitializedBeat(settings)
			if err != nil {
				fatalf("error initializing beat: %+v", err)
			}

			idxManager := b.IdxMgmtSupporter().Manager(nil, idxmgmt.BeatsAssets(b.Fields))
			templateLoadCfg := idxmgmt.SetupConfig{Load: new(bool)}
			ilmLoadCfg := idxmgmt.DefaultSetupConfig()
			if err := idxManager.Setup(templateLoadCfg, ilmLoadCfg); err != nil {
				fatalf("exporting ilm-policy failed: %+v", err)
			}
		},
	}

	return genTemplateConfigCmd
}
