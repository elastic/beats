// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cmd

import (
	"github.com/spf13/cobra"

	cmd "github.com/elastic/beats/libbeat/cmd"
	"github.com/elastic/beats/x-pack/beatless/beater"
)

// Name of this beat
var Name = "beatless"

// RootCmd to handle beatless
var RootCmd *cmd.BeatsRootCmd

func init() {
	b := beater.New
	RootCmd = cmd.GenRootCmd(Name, "", b)

	functionCmd := &cobra.Command{
		Use:   "function",
		Short: "Manage functions",
	}

	functionCmd.AddCommand(genDeployCmd())
	functionCmd.AddCommand(genUpdateCmd())
	functionCmd.AddCommand(genRemoveCmd())

	RootCmd.AddCommand(functionCmd)
}
