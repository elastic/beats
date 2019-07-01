// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cmd

import (
	cmd "github.com/elastic/beats/libbeat/cmd"
	"github.com/elastic/beats/libbeat/cmd/instance"
	"github.com/elastic/beats/x-pack/functionbeat/function/beater"
	"github.com/elastic/beats/x-pack/functionbeat/function/config"
)

// CfgNamespace is the namespace under configuration options are located.
var CfgNamespace = "functionbeat"

// GenRootCmdWithBeatName generates common command for Functionbeat and sets the name containing the provider.
func GenRootCmdWithBeatName(name string) *cmd.BeatsRootCmd {
	rootCmd := cmd.GenRootCmdWithSettings(beater.New, instance.Settings{
		Name:            name,
		ConfigNamespace: CfgNamespace,
		ConfigOverrides: config.ConfigOverrides,
	})

	rootCmd.AddCommand(genDeployCmd(name))
	rootCmd.AddCommand(genUpdateCmd(name))
	rootCmd.AddCommand(genRemoveCmd(name))
	rootCmd.AddCommand(genPackageCmd(name))

	rootCmd.ExportCmd.Short = "Export current config, index template or function"
	rootCmd.ExportCmd.AddCommand(genExportFunctionCmd(name))

	return rootCmd
}
