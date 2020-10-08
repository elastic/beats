// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build !windows

package cmd

import (
	"github.com/spf13/cobra"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/cli"
)

func newReExecWindowsCommand(flags *globalFlags, _ []string, streams *cli.IOStreams) *cobra.Command {
	return nil
}
