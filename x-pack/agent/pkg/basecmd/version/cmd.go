// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package version

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/elastic/beats/v7/x-pack/agent/pkg/cli"
	"github.com/elastic/beats/v7/x-pack/agent/pkg/release"
)

// NewCommandWithArgs returns a new version command.
func NewCommandWithArgs(streams *cli.IOStreams) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Display the version of the agent.",
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Fprintf(
				streams.Out,
				"Agent version is %s (build: %s at %s)\n",
				release.Version(),
				release.Commit(),
				release.BuildTime(),
			)
		},
	}
}
