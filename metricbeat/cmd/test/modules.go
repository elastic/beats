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

package test

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/elastic/beats/libbeat/cmd/instance"
	"github.com/elastic/beats/libbeat/testing"
	"github.com/elastic/beats/metricbeat/beater"
	"github.com/elastic/beats/metricbeat/mb/module"
)

func GenTestModulesCmd(name, beatVersion string) *cobra.Command {
	return &cobra.Command{
		Use:   "modules [module] [metricset]",
		Short: "Test modules settings",
		Run: func(cmd *cobra.Command, args []string) {
			var filter_module, filter_metricset string
			if len(args) > 0 {
				filter_module = args[0]
			}

			if len(args) > 1 {
				filter_metricset = args[1]
			}

			b, err := instance.NewBeat(name, "", beatVersion)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error initializing beat: %s\n", err)
				os.Exit(1)
			}

			err = b.Init()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error initializing beat: %s\n", err)
				os.Exit(1)
			}

			// Use a customized instance of Metricbeat where startup delay has
			// been disabled to workaround the fact that Modules() will return
			// the static modules (not the dynamic ones) with a start delay.
			create := beater.Creator(
				beater.WithModuleOptions(
					module.WithMetricSetInfo(),
					module.WithMaxStartDelay(0),
				),
			)
			mb, err := create(&b.Beat, b.Beat.BeatConfig)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error initializing metricbeat: %s\n", err)
				os.Exit(1)
			}

			modules, err := mb.(*beater.Metricbeat).Modules()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error getting metricbeat modules: %s\n", err)
				os.Exit(1)
			}

			driver := testing.NewConsoleDriver(os.Stdout)
			for _, module := range modules {
				if filter_module != "" && module.Name() != filter_module {
					continue
				}
				driver.Run(module.Name(), func(driver testing.Driver) {
					for _, set := range module.MetricSets() {
						if filter_metricset != "" && set.Name() != filter_metricset {
							continue
						}
						set.Test(driver)
					}
				})
			}
		},
	}
}
