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
	"fmt"
	"os"

	"github.com/elastic/beats/libbeat/idxmgmt/ilm"

	"github.com/spf13/cobra"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/cmd/instance"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/idxmgmt"
	"github.com/elastic/beats/libbeat/logp"
)

func GenTemplateConfigCmd(settings instance.Settings) *cobra.Command {
	genTemplateConfigCmd := &cobra.Command{
		Use:   "template",
		Short: "Export index template to stdout",
		Run: func(cmd *cobra.Command, args []string) {
			noILM, _ := cmd.Flags().GetBool("noilm")
			if noILM {
				settings.ILM = ilmNoopSupport
			}

			b, err := instance.NewInitializedBeat(settings)
			if err != nil {
				fatalf("error initializing beat: %+v", err)
			}

			idxManager := b.IdxMgmtSupporter().Manager(nil, idxmgmt.BeatsAssets(b.Fields))
			ilmLoadCfg := idxmgmt.SetupConfig{Load: new(bool)}
			templateLoadCfg := idxmgmt.DefaultSetupConfig()
			if err := idxManager.Setup(templateLoadCfg, ilmLoadCfg); err != nil {
				fatalf("exporting template failed: %+v", err)
			}
		},
	}

	genTemplateConfigCmd.Flags().Bool("noilm", false, "Generate template with ILM disabled")

	return genTemplateConfigCmd
}

func fatalf(msg string, vs ...interface{}) {
	fmt.Fprintf(os.Stderr, msg, vs...)
	fmt.Fprintln(os.Stderr)
	os.Exit(1)
}

func ilmNoopSupport(log *logp.Logger, info beat.Info, config *common.Config) (ilm.Supporter, error) {
	if log == nil {
		log = logp.NewLogger("export template")
	} else {
		log = log.Named("export template")
	}

	return ilm.NoopSupport(info, config)
}
