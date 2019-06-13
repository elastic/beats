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

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/cmd/instance"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/idxmgmt"
	"github.com/elastic/beats/libbeat/idxmgmt/ilm"
	"github.com/elastic/beats/libbeat/logp"
)

// GenTemplateConfigCmd is the command used to export the elasticsearch template.
func GenTemplateConfigCmd(settings instance.Settings) *cobra.Command {
	genTemplateConfigCmd := &cobra.Command{
		Use:   "template",
		Short: "Export index template to stdout",
		Run: func(cmd *cobra.Command, args []string) {
			version, _ := cmd.Flags().GetString("es.version")
			dir, _ := cmd.Flags().GetString("dir")
			noILM, _ := cmd.Flags().GetBool("noilm")

			if noILM {
				settings.ILM = ilmNoopSupport
			}

			b, err := instance.NewInitializedBeat(settings)
			if err != nil {
				fatalfInitCmd(err)
			}

			clientHandler := idxmgmt.NewFileClientHandler(newIdxmgmtClient(dir, version))
			idxManager := b.IdxSupporter.Manager(clientHandler, idxmgmt.BeatsAssets(b.Fields))
			if err := idxManager.Setup(idxmgmt.LoadModeEnabled, idxmgmt.LoadModeDisabled); err != nil {
				fatalf("Error exporting template: %+v.", err)
			}
		},
	}

	genTemplateConfigCmd.Flags().String("es.version", settings.Version, "Elasticsearch version")
	genTemplateConfigCmd.Flags().Bool("noilm", false, "Generate template with ILM disabled")
	genTemplateConfigCmd.Flags().String("dir", "", "Specify directory for printing template files. By default templates are printed to stdout.")

	return genTemplateConfigCmd
}

func ilmNoopSupport(_ *logp.Logger, info beat.Info, config *common.Config) (ilm.Supporter, error) {
	return ilm.NoopSupport(info, config)
}
