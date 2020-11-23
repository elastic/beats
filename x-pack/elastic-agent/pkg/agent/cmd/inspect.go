// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/cli"
)

func newInspectCommandWithArgs(flags *globalFlags, s []string, streams *cli.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "inspect",
		Short: "Shows configuration of the agent",
		Long:  "Shows current configuration of the agent",
		Args:  cobra.ExactArgs(0),
		Run: func(c *cobra.Command, args []string) {
			command, err := application.NewInspectConfigCmd(flags.Config())
			if err != nil {
				fmt.Fprintf(streams.Err, "%v\n", err)
				os.Exit(1)
			}

			if err := command.Execute(); err != nil {
				fmt.Fprintf(streams.Err, "%v\n", err)
				os.Exit(1)
			}
		},
	}

	cmd.AddCommand(newInspectOutputCommandWithArgs(flags, s, streams))

	return cmd
}

func newInspectOutputCommandWithArgs(flags *globalFlags, _ []string, streams *cli.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "output",
		Short: "Displays configuration generated for output",
		Long:  "Displays configuration generated for output.\nIf no output is specified list of output is displayed",
		Args:  cobra.MaximumNArgs(2),
		Run: func(c *cobra.Command, args []string) {
			outName, _ := c.Flags().GetString("output")
			program, _ := c.Flags().GetString("program")

			command, err := application.NewInspectOutputCmd(flags.Config(), outName, program)
			if err != nil {
				fmt.Fprintf(streams.Err, "%v\n", err)
				os.Exit(1)
			}

			if err := command.Execute(); err != nil {
				fmt.Fprintf(streams.Err, "%v\n", err)
				os.Exit(1)
			}
		},
	}

	cmd.Flags().StringP("output", "o", "", "name of the output to be inspected")
	cmd.Flags().StringP("program", "p", "", "type of program to inspect, needs to be combined with output. e.g filebeat")

	return cmd
}
