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

	"github.com/elastic/beats/v8/filebeat/generator/fields"
	"github.com/elastic/beats/v8/filebeat/generator/fileset"
	"github.com/elastic/beats/v8/filebeat/generator/module"
	"github.com/elastic/beats/v8/libbeat/common/cli"
	"github.com/elastic/beats/v8/libbeat/paths"
)

var defaultHomePath = paths.Resolve(paths.Home, "")

func genGenerateCmd() *cobra.Command {
	generateCmd := cobra.Command{
		Use:   "generate",
		Short: "Generate Filebeat modules, filesets and fields.yml",
	}
	generateCmd.AddCommand(genGenerateModuleCmd())
	generateCmd.AddCommand(genGenerateFilesetCmd())
	generateCmd.AddCommand(genGenerateFieldsCmd())

	return &generateCmd
}

func genGenerateModuleCmd() *cobra.Command {
	genModuleCmd := &cobra.Command{
		Use:   "module [module]",
		Short: "Generates a new module",
		Run: cli.RunWith(func(cmd *cobra.Command, args []string) error {
			modulesPath, _ := cmd.Flags().GetString("modules-path")
			esBeatsPath, _ := cmd.Flags().GetString("es-beats")

			if len(args) != 1 {
				fmt.Fprintf(os.Stderr, "Exactly one parameter is required: module name\n")
				os.Exit(1)
			}
			name := args[0]

			return module.Generate(name, modulesPath, esBeatsPath)
		}),
	}

	genModuleCmd.Flags().String("modules-path", defaultHomePath, "Path to modules directory")
	genModuleCmd.Flags().String("es-beats", defaultHomePath, "Path to Elastic Beats")

	return genModuleCmd
}

func genGenerateFilesetCmd() *cobra.Command {
	genFilesetCmd := &cobra.Command{
		Use:   "fileset [module] [fileset]",
		Short: "Generates a new fileset",
		Run: cli.RunWith(func(cmd *cobra.Command, args []string) error {
			modulesPath, _ := cmd.Flags().GetString("modules-path")
			esBeatsPath, _ := cmd.Flags().GetString("es-beats")

			if len(args) != 2 {
				fmt.Fprintf(os.Stderr, "Two parameters are required: module name, fileset name\n")
				os.Exit(1)
			}
			moduleName := args[0]
			filesetName := args[1]

			return fileset.Generate(moduleName, filesetName, modulesPath, esBeatsPath)
		}),
	}

	genFilesetCmd.Flags().String("modules-path", defaultHomePath, "Path to modules directory")
	genFilesetCmd.Flags().String("es-beats", defaultHomePath, "Path to Elastic Beats")

	return genFilesetCmd
}

func genGenerateFieldsCmd() *cobra.Command {
	genFieldsCmd := &cobra.Command{
		Use:   "fields [module] [fileset]",
		Short: "Generates a new fields.yml file for fileset",
		Run: cli.RunWith(func(cmd *cobra.Command, args []string) error {
			esBeatsPath, _ := cmd.Flags().GetString("es-beats")
			noDoc, _ := cmd.Flags().GetBool("without-documentation")

			if len(args) != 2 {
				fmt.Fprintf(os.Stderr, "Two parameters are required: module name, fileset name\n")
				os.Exit(1)
			}
			moduleName := args[0]
			filesetName := args[1]

			return fields.Generate(esBeatsPath, moduleName, filesetName, noDoc)
		}),
	}

	genFieldsCmd.Flags().String("es-beats", defaultHomePath, "Path to Elastic Beats")
	genFieldsCmd.Flags().Bool("without-documentation", false, "Do not add description fields")

	return genFieldsCmd
}
