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

	"github.com/spf13/cobra"

	"github.com/elastic/beats/libbeat/cmd/instance"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/idxmgmt"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/paths"
	"github.com/elastic/beats/libbeat/template"
)

func GenTemplateConfigCmd(settings instance.Settings) *cobra.Command {
	genTemplateConfigCmd := &cobra.Command{
		Use:   "template",
		Short: "Export index template to stdout",
		Run: func(cmd *cobra.Command, args []string) {
			version, _ := cmd.Flags().GetString("es.version")
			index, _ := cmd.Flags().GetString("index")
			noILM, _ := cmd.Flags().GetBool("noilm")

			b, err := instance.NewBeat(settings.Name, settings.IndexPrefix, settings.Version)
			if err != nil {
				fatalf("Error initializing beat: %+v", err)
			}
			err = b.InitWithSettings(settings)
			if err != nil {
				fatalf("Error initializing beat: %+v", err)
			}

			if version == "" {
				version = b.Info.Version
			}
			esVersion, err := common.NewVersion(version)
			if err != nil {
				fatalf("Invalid Elasticsearch version: %+v", err)
			}

			imFactory := settings.IndexManagement
			if imFactory == nil {
				imFactory = idxmgmt.MakeDefaultSupport(settings.ILM)
			}
			indexManager, err := imFactory(logp.NewLogger("index-management"), b.Info, b.RawConfig)
			if err != nil {
				fatalf("Error initializing the index manager: %+v", err)
			}

			tmplCfg, err := indexManager.TemplateConfig(!noILM)
			if err != nil {
				fatalf("Template error detected: %+v", err)
			}
			if tmplCfg.Enabled == false {
				tmplCfg = template.DefaultConfig()
			}

			tmpl, err := template.New(b.Info.Version, index, *esVersion, tmplCfg, b.Config.Migration.Enabled())
			if err != nil {
				fatalf("Error generating template: %+v", err)
			}

			var templateString common.MapStr
			if tmplCfg.Fields != "" {
				fieldsPath := paths.Resolve(paths.Config, tmplCfg.Fields)
				templateString, err = tmpl.LoadFile(fieldsPath)
			} else {
				templateString, err = tmpl.LoadBytes(b.Fields)
			}
			if err != nil {
				fatalf("Error generating template: %+v", err)
			}

			_, err = os.Stdout.WriteString(templateString.StringToPrint() + "\n")
			if err != nil {
				fatalf("Error writing template: %+v", err)
			}
		},
	}

	genTemplateConfigCmd.Flags().String("es.version", settings.Version, "Elasticsearch version")
	genTemplateConfigCmd.Flags().String("index", settings.IndexPrefix, "Base index name")
	genTemplateConfigCmd.Flags().Bool("noilm", false, "Generate template with ILM disabled")

	return genTemplateConfigCmd
}

func fatalf(msg string, vs ...interface{}) {
	fmt.Fprintf(os.Stderr, msg, vs...)
	fmt.Fprintln(os.Stderr)
	os.Exit(1)
}
