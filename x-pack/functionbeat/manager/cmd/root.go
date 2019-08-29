// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cmd

import (
	cmd "github.com/elastic/beats/libbeat/cmd"
	"github.com/elastic/beats/libbeat/cmd/instance"
	"github.com/elastic/beats/x-pack/functionbeat/config"
	"github.com/elastic/beats/x-pack/functionbeat/manager/beater"
)

// Name of this beat
var Name = "functionbeat"

// RootCmd to handle functionbeat
var RootCmd *cmd.BeatsRootCmd

func init() {
	RootCmd = cmd.GenRootCmdWithSettings(beater.New, instance.Settings{
		Name:            Name,
		ConfigOverrides: config.Overrides,
	})

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
