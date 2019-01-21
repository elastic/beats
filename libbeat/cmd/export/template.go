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

	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/template"

	"github.com/spf13/cobra"

	"github.com/elastic/beats/libbeat/cmd/instance"
	"github.com/elastic/beats/libbeat/index"
)

//GenTemplateConfigCmd implements generating a template for the given settings and
//prints it to stdout
func GenTemplateConfigCmd(settings instance.Settings, name, idxPrefix, beatVersion string) *cobra.Command {
	genTemplateConfigCmd := &cobra.Command{
		Use:   "template",
		Short: "Export index templates to stdout",
		Run: func(cmd *cobra.Command, args []string) {
			version, _ := cmd.Flags().GetString("es.version")
			idx, _ := cmd.Flags().GetString("index")

			b, err := instance.NewBeat(name, idx, version)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error initializing beat: %s\n", err)
				os.Exit(1)
			}
			err = b.InitWithSettings(settings)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error initializing beat: %s\n", err)
				os.Exit(1)
			}

			if err != nil {
				fmt.Fprintf(os.Stderr, "Invalid Elasticsearch version: %s\n", err)

				os.Exit(1)
			}

			cfg := b.Config.Indices
			if len(cfg) == 0 {
				cfg, err = index.DeprecatedTemplateConfigs(b.Config.Template)
				if err != nil {
					fmt.Fprintf(os.Stderr, "unpacking template config fails: %v", err)
					os.Exit(1)
				}
			}

			loader, err := template.NewStdoutLoader(b.Info, b.Config.Migration.Enabled())
			if err != nil {
				fmt.Fprintf(os.Stderr, "error initializing ilm loader: %s\n", err)
				os.Exit(1)
			}
			if _, _, _, err = index.LoadTemplates(loader, cfg); err != nil {
				fmt.Fprintf(os.Stderr, err.Error())
				os.Exit(1)
			}
			logp.Info("Printed Elasticsearch templates.")
		},
	}

	genTemplateConfigCmd.Flags().String("es.version", beatVersion, "Elasticsearch version")
	genTemplateConfigCmd.Flags().String("index", idxPrefix, "Base index name")

	return genTemplateConfigCmd
}
