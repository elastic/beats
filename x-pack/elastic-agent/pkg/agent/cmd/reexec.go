// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build !windows

package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/cli"
)

func newReExecWindowsCommand(flags *globalFlags, _ []string, streams *cli.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Hidden: true,
		Use:    "reexec_windows <service_name> <pid>",
		Short:  "ReExec the windows service",
		Long:   "This waits for the windows service to stop then restarts it to allow self-upgrading.",
		Args:   cobra.ExactArgs(2),
		Run: func(c *cobra.Command, args []string) {
			fmt.Fprint(streams.Err, "Error: not windows; cannot be used!\n")
			os.Exit(1)
		},
	}

	return cmd
}
