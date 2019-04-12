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
	"log"
	"os"

	"github.com/spf13/cobra"

	"github.com/elastic/beats/libbeat/cmd/instance"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/kibana"
)

// GenIndexPatternConfigCmd generates an index pattern for Kibana
func GenIndexPatternConfigCmd(settings instance.Settings) *cobra.Command {
	genTemplateConfigCmd := &cobra.Command{
		Use:   "index-pattern",
		Short: "Export kibana index pattern to stdout",
		Run: func(cmd *cobra.Command, args []string) {
			version, _ := cmd.Flags().GetString("es.version")

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

			// Index pattern generation
			v, err := common.NewVersion(version)
			if err != nil {
				fatalf("Error creating version: %+v", err)
			}
			indexPattern, err := kibana.NewGenerator(b.Info.IndexPrefix, b.Info.Beat, b.Fields, settings.Version, *v, b.Config.Migration.Enabled())
			if err != nil {
				log.Fatal(err)
			}

			pattern, err := indexPattern.Generate()
			if err != nil {
				log.Fatalf("ERROR: %s", err)
			}

			_, err = os.Stdout.WriteString(pattern.StringToPrint() + "\n")
			if err != nil {
				fatalf("Error writing index pattern: %+v", err)
			}
		},
	}

	genTemplateConfigCmd.Flags().String("es.version", settings.Version, "Elasticsearch version")

	return genTemplateConfigCmd
}
