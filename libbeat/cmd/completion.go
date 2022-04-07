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

	"github.com/elastic/beats/v8/libbeat/cmd/instance"

	"github.com/spf13/cobra"
)

func genCompletionCmd(_ instance.Settings, rootCmd *BeatsRootCmd) *cobra.Command {
	completionCmd := cobra.Command{
		Use:   "completion SHELL",
		Short: "Output shell completion code for the specified shell (bash and zsh only by the moment)",
		// We don't want to expose this one in help:
		Hidden: true,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 1 {
				fmt.Println("Expected one argument with the desired shell")
				os.Exit(1)
			}

			switch args[0] {
			case "bash":
				rootCmd.GenBashCompletion(os.Stdout)
			case "zsh":
				rootCmd.GenZshCompletion(os.Stdout)
			default:
				fmt.Printf("Unknown shell %s, only bash and zsh are available\n", args[0])
				os.Exit(1)
			}
		},
	}

	return &completionCmd
}
