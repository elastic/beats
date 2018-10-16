// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cmd

import (
	"github.com/spf13/cobra"

	cmd "github.com/elastic/beats/libbeat/cmd"
	"github.com/elastic/beats/libbeat/cmd/instance"
	"github.com/elastic/beats/x-pack/beatless/beater"
	"github.com/elastic/beats/x-pack/beatless/config"
)

// Name of this beat
var Name = "beatless"

// RootCmd to handle beatless
var RootCmd *cmd.BeatsRootCmd

func init() {
	RootCmd = cmd.GenRootCmdWithSettings(beater.New, instance.Settings{
		Name:            Name,
		ConfigOverrides: config.ConfigOverrides,
	})

	functionCmd := &cobra.Command{
		Use:   "function",
		Short: "Manage functions",
	}

	functionCmd.AddCommand(genDeployCmd())
	functionCmd.AddCommand(genUpdateCmd())
	functionCmd.AddCommand(genRemoveCmd())
	functionCmd.AddCommand(genPackageCmd())

	RootCmd.AddCommand(functionCmd)

}
