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

package cmd

import (
	"fmt"

	"os"

	"github.com/spf13/cobra"

	"github.com/elastic/beats/libbeat/cmd/instance"
	"github.com/elastic/beats/libbeat/fields/rollup"
	"github.com/elastic/beats/libbeat/template"
)

func GenRollupConfigCmd(name string) *cobra.Command {
	genRollupConfigCmd := &cobra.Command{
		Use:   "rollup",
		Short: "Export rollup config for module/metricset to stdout",
		Run: func(cmd *cobra.Command, args []string) {

			// Skipping index prefix and beat version as not needed
			b, err := instance.NewBeat(name, "", "")
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error creating beat: %s\n", err)
				os.Exit(1)
			}

			module, _ := cmd.Flags().GetString("module")
			metricSet, _ := cmd.Flags().GetString("metricset")
			if module == "" || metricSet == "" {
				fmt.Fprintf(os.Stderr, "Module and metricset params have to be set.")
				os.Exit(1)
			}

			// Load fields from Beat
			f, err := template.LoadYamlByte(b.Fields)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error loading fields: %s\n", err)
				os.Exit(1)
			}
			nodePath := fmt.Sprintf("%s.%s", module, metricSet)
			fields := f.GetNode(nodePath)

			if fields == nil {
				fmt.Fprint(os.Stderr, "No fields found for module %s and metricset %s\n", module, metricSet)
				os.Exit(1)
			}

			// Load all the additional params
			indexPattern, _ := cmd.Flags().GetString("index_pattern")
			rollupIndex, _ := cmd.Flags().GetString("rollup_index")
			cron, _ := cmd.Flags().GetString("cron")
			pageSize, _ := cmd.Flags().GetString("page_size")
			interval, _ := cmd.Flags().GetString("interval")
			delay, _ := cmd.Flags().GetString("delay")

			processor := rollup.NewProcessor()
			err = processor.Process(fields, nodePath)
			if err != nil {
				fmt.Fprint(os.Stderr, "Error processing fields: %s", err)
				os.Exit(1)
			}

			// Note: No validation happens if the strings used are actually valid
			// TODO: What if no terms are specified? Should we return error?
			_, err = os.Stdout.WriteString(processor.Generate(indexPattern, rollupIndex, cron, pageSize, interval, delay).StringToPrint() + "\n")
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error writing rollup job: %+v", err)
				os.Exit(1)
			}
		},
	}

	genRollupConfigCmd.Flags().String("module", "", "Module to create rollup job for")
	genRollupConfigCmd.Flags().String("metricset", "", "Metricset to create rollup job for")
	genRollupConfigCmd.Flags().String("index_pattern", "metricbeat-*", "Index pattern to roll up on")
	genRollupConfigCmd.Flags().String("rollup_index", "rollup-metricbeat", "Rollup index")
	genRollupConfigCmd.Flags().String("cron", "*/30 * * * * ?s", "Rollup cron")
	genRollupConfigCmd.Flags().Int("page_size", 10, "Rollup page size")
	genRollupConfigCmd.Flags().String("interval", "1h", "Rollup interval")
	genRollupConfigCmd.Flags().String("delay", "7d", "Rollup delay")

	return genRollupConfigCmd
}
