// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	cmd "github.com/elastic/beats/v7/libbeat/cmd"
	"github.com/elastic/beats/v7/libbeat/cmd/instance"
	"github.com/elastic/beats/v7/x-pack/functionbeat/config"
	"github.com/elastic/beats/v7/x-pack/functionbeat/manager/beater"
)

// Name of this beat
var Name = "functionbeat"

// RootCmd to handle functionbeat
var RootCmd *cmd.BeatsRootCmd

func init() {
	RootCmd = cmd.GenRootCmdWithSettings(beater.New, instance.Settings{
		Name:            Name,
		HasDashboards:   false,
		ConfigOverrides: config.Overrides,
		ElasticLicensed: true,
	})

	RootCmd.RemoveCommand(RootCmd.RunCmd)
	RootCmd.Run = func(_ *cobra.Command, _ []string) {
		fmt.Println("Functionbeat is going to be removed in 8.1")

		RootCmd.Usage()
		os.Exit(1)
	}

	RootCmd.AddCommand(genDeployCmd())
	RootCmd.AddCommand(genUpdateCmd())
	RootCmd.AddCommand(genRemoveCmd())
	RootCmd.AddCommand(genPackageCmd())

	addBeatSpecificSubcommands()
}

func addBeatSpecificSubcommands() {
	RootCmd.ExportCmd.Short = "Export current config, index template or function"
	RootCmd.ExportCmd.AddCommand(genExportFunctionCmd())
}
