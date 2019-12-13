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
	"github.com/elastic/beats/libbeat/idxmgmt"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/testing"
)

func GenTestOutputCmd(settings instance.Settings) *cobra.Command {
	return &cobra.Command{
		Use:   "output",
		Short: "Test " + settings.Name + " can connect to the output by using the current settings",
		Run: func(cmd *cobra.Command, args []string) {
			b, err := instance.NewInitializedBeat(settings)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error initializing beat: %s\n", err)
				os.Exit(1)
			}

			im, _ := idxmgmt.DefaultSupport(nil, b.Info, nil)
			output, err := outputs.Load(im, b.Info, nil, b.Config.Output.Name(), b.Config.Output.Config())
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error initializing output: %s\n", err)
				os.Exit(1)
			}

			for _, client := range output.Clients {
				tClient, ok := client.(testing.Testable)
				if !ok {
					fmt.Printf("%s output doesn't support testing\n", b.Config.Output.Name())
					os.Exit(1)
				}

				// Perform test:
				tClient.Test(testing.NewConsoleDriver(os.Stdout))
			}
		},
	}
}
