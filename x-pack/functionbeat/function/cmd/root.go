// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/elastic/beats/v8/libbeat/beat"
	"github.com/elastic/beats/v8/libbeat/cfgfile"
	"github.com/elastic/beats/v8/libbeat/cmd"
	"github.com/elastic/beats/v8/libbeat/cmd/instance"
	"github.com/elastic/beats/v8/x-pack/functionbeat/config"
)

// FunctionCmd is the command of the function.
type FunctionCmd struct {
	*cobra.Command
	VersionCmd *cobra.Command
}

// NewFunctionCmd return a new initialized function command.
func NewFunctionCmd(name string, beatCreator beat.Creator) *FunctionCmd {
	settings := instance.Settings{
		Name:            name,
		IndexPrefix:     name,
		ConfigOverrides: config.FunctionOverrides,
	}

	err := cfgfile.ChangeDefaultCfgfileFlag(settings.Name)
	if err != nil {
		panic(fmt.Errorf("failed to set default config file path: %v", err))
	}

	rootCmd := &FunctionCmd{
		&cobra.Command{
			Run: func(cmd *cobra.Command, args []string) {
				err := instance.Run(settings, beatCreator)
				if err != nil {
					os.Exit(1)
				}
			},
		},
		cmd.GenVersionCmd(settings),
	}

	rootCmd.AddCommand(rootCmd.VersionCmd)

	return rootCmd
}
