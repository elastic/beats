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
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/elastic/beats/v7/libbeat/cmd/instance"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/dashboards"
	"github.com/elastic/beats/v7/libbeat/kibana"
)

// GenDashboardCmd is the command used to export a dashboard.
func GenDashboardCmd(settings instance.Settings) *cobra.Command {
	genTemplateConfigCmd := &cobra.Command{
		Use:   "dashboard",
		Short: "Export defined dashboard to stdout",
		Run: func(cmd *cobra.Command, args []string) {
			dashboard, _ := cmd.Flags().GetString("id")
			yml, _ := cmd.Flags().GetString("yml")
			folder, _ := cmd.Flags().GetString("folder")

			if len(folder) == 0 {
				fatalf("-folder must be specified")
			}

			b, err := instance.NewInitializedBeat(settings)
			if err != nil {
				fatalfInitCmd(err)
			}

			// Use empty config to use default configs if not set
			if b.Config.Kibana == nil {
				b.Config.Kibana = common.NewConfig()
			}

			// Initialize kibana config. If username and password is set in
			// elasticsearch output config but not in kibana, initKibanaConfig
			// will attach the username and password into kibana config as a
			// part of the initialization.
			initConfig := instance.InitKibanaConfig(b.Config)

			client, err := kibana.NewKibanaClient(initConfig, b.Info.Beat)
			if err != nil {
				fatalf("Error creating Kibana client: %+v.\n", err)
			}

			// Export dashboards from yml file
			if yml != "" {
				results, info, err := dashboards.ExportAllFromYml(client, yml)
				if err != nil {
					fatalf("Error exporting dashboards from yml: %+v.\n", err)
				}
				for i, r := range results {
					r = dashboards.DecodeExported(r)

					err = dashboards.SaveToFile(r, info.Dashboards[i].File, filepath.Dir(yml), client.GetVersion())
					if err != nil {
						fatalf("Error saving dashboard '%s' to file '%s' : %+v.\n",
							info.Dashboards[i].ID, info.Dashboards[i].File, err)
					}
				}
				return
			}

			// Export single dashboard
			if dashboard != "" {
				result, err := dashboards.Export(client, dashboard)
				if err != nil {
					fatalf("Error exporting dashboard: %+v.\n", err)
				}

				result = dashboards.DecodeExported(result)

				err = dashboards.SaveToFolder(result, folder, client.GetVersion())
				if err != nil {
					fatalf("Error saving assets to folder '%s' : %+v.\n", folder, err)
				}

			}
		},
	}

	genTemplateConfigCmd.Flags().String("id", "", "Dashboard id")
	genTemplateConfigCmd.Flags().String("yml", "", "Yaml file containing list of dashboard ID and filename pairs")
	genTemplateConfigCmd.Flags().String("folder", "", "Target folder to save exported assets")

	return genTemplateConfigCmd
}
