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
	"flag"
	"os"

	"github.com/spf13/cobra"

	"github.com/menderesk/beats/v7/libbeat/beat"
	"github.com/menderesk/beats/v7/libbeat/cmd/instance"
)

func genRunCmd(settings instance.Settings, beatCreator beat.Creator) *cobra.Command {
	name := settings.Name
	runCmd := cobra.Command{
		Use:   "run",
		Short: "Run " + name,
		Run: func(cmd *cobra.Command, args []string) {
			err := instance.Run(settings, beatCreator)
			if err != nil {
				os.Exit(1)
			}
		},
	}

	// Run subcommand flags, only available to *beat run
	runCmd.Flags().AddGoFlag(flag.CommandLine.Lookup("N"))
	runCmd.Flags().AddGoFlag(flag.CommandLine.Lookup("httpprof"))
	runCmd.Flags().AddGoFlag(flag.CommandLine.Lookup("cpuprofile"))
	runCmd.Flags().AddGoFlag(flag.CommandLine.Lookup("memprofile"))

	if settings.RunFlags != nil {
		runCmd.Flags().AddFlagSet(settings.RunFlags)
	}

	return &runCmd
}
