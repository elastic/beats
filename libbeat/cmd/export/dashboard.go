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
	"github.com/elastic/beats/libbeat/kibana"
)

// GenDashboardCmd is the command used to export a dashboard.
func GenDashboardCmd(name, idxPrefix, beatVersion string) *cobra.Command {
	genTemplateConfigCmd := &cobra.Command{
		Use:   "dashboard",
		Short: "Export defined dashboard to stdout",
		Run: func(cmd *cobra.Command, args []string) {
			dashboard, _ := cmd.Flags().GetString("id")

			b, err := instance.NewBeat(name, idxPrefix, beatVersion)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error creating beat: %s\n", err)
				os.Exit(1)
			}
			err = b.Init()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error initializing beat: %s\n", err)
				os.Exit(1)
			}

			// Use empty config to use default configs if not set
			if b.Config.Kibana == nil {
				b.Config.Kibana = common.NewConfig()
			}

			client, err := kibana.NewKibanaClient(b.Config.Kibana)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error creating Kibana client: %+v\n", err)
				os.Exit(1)
			}

			result, err := client.GetDashboard(dashboard)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error getting dashboard: %+v\n", err)
				os.Exit(1)
			}
			fmt.Println(result.StringToPrint())
		},
	}

	genTemplateConfigCmd.Flags().String("id", "", "Dashboard id")

	return genTemplateConfigCmd
}
