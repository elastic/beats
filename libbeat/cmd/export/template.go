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
	"github.com/elastic/beats/libbeat/paths"
	"github.com/elastic/beats/libbeat/template"
)

func GenTemplateConfigCmd(settings instance.Settings, name, idxPrefix, beatVersion string) *cobra.Command {
	genTemplateConfigCmd := &cobra.Command{
		Use:   "template",
		Short: "Export index template to stdout",
		Run: func(cmd *cobra.Command, args []string) {
			version, _ := cmd.Flags().GetString("es.version")
			index, _ := cmd.Flags().GetString("index")

			b, err := instance.NewBeat(name, idxPrefix, beatVersion)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error initializing beat: %s\n", err)
				os.Exit(1)
			}
			err = b.InitWithSettings(settings)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error initializing beat: %s\n", err)
				os.Exit(1)
			}

			cfg := template.DefaultConfig
			if b.Config.Template.Enabled() {
				err = b.Config.Template.Unpack(&cfg)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error getting template settings: %+v", err)
					os.Exit(1)
				}
			}

			if version == "" {
				version = b.Info.Version
			}

			esVersion, err := common.NewVersion(version)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Invalid Elasticsearch version: %s\n", err)
			}

			tmpl, err := template.New(b.Info.Version, index, *esVersion, cfg)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error generating template: %+v", err)
				os.Exit(1)
			}

			var templateString common.MapStr
			if cfg.Fields != "" {
				fieldsPath := paths.Resolve(paths.Config, cfg.Fields)
				templateString, err = tmpl.LoadFile(fieldsPath)
			} else {
				templateString, err = tmpl.LoadBytes(b.Fields)
			}

			if err != nil {
				fmt.Fprintf(os.Stderr, "Error generating template: %+v", err)
				os.Exit(1)
			}

			_, err = os.Stdout.WriteString(templateString.StringToPrint() + "\n")
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error writing template: %+v", err)
				os.Exit(1)
			}
		},
	}

	genTemplateConfigCmd.Flags().String("es.version", beatVersion, "Elasticsearch version")
	genTemplateConfigCmd.Flags().String("index", idxPrefix, "Base index name")

	return genTemplateConfigCmd
}
