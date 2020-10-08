// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/control"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/control/client"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/cli"
)

func newUpgradeCommandWithArgs(flags *globalFlags, _ []string, streams *cli.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "upgrade <version>",
		Short: "Upgrade the currently running Elastic Agent to the specified version",
		Args:  cobra.ExactArgs(1),
		Run: func(c *cobra.Command, args []string) {
			if err := upgradeCmd(streams, c, flags, args); err != nil {
				fmt.Fprintf(streams.Err, "%v\n", err)
				os.Exit(1)
			}
		},
	}

	cmd.Flags().StringP("source-uri", "s", "", "Source URI to download the new version from")

	return cmd
}

func upgradeCmd(streams *cli.IOStreams, cmd *cobra.Command, flags *globalFlags, args []string) error {
	fmt.Fprintln(streams.Out, "The upgrade process of Elastic Agent is currently EXPERIMENTAL and should not be used in production")

	version := args[0]
	sourceURI, _ := cmd.Flags().GetString("source-uri")

	c := client.New()
	err := c.Connect(context.Background())
	if err != nil {
		return errors.New(err, "Failed communicating to running daemon", errors.TypeNetwork, errors.M("socket", control.Address()))
	}
	defer c.Disconnect()
	version, err = c.Upgrade(context.Background(), version, sourceURI)
	if err != nil {
		return errors.New(err, "Failed trigger upgrade of daemon")
	}
	fmt.Fprintf(streams.Out, "Upgrade triggered to version %s, Elastic Agent is currently restarting\n", version)
	return nil
}
