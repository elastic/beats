// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cmd

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	c "github.com/elastic/beats/libbeat/common/cli"
	"github.com/elastic/beats/x-pack/agent/pkg/agent/application"
	"github.com/elastic/beats/x-pack/agent/pkg/cli"
	"github.com/elastic/beats/x-pack/agent/pkg/config"
	"github.com/elastic/beats/x-pack/agent/pkg/core/logger"
)

func newEnrollCommandWithArgs(flags *globalFlags, _ []string, streams *cli.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "enroll <kibana_url> <enrollment_token>",
		Short: "Enroll the Agent into Fleet",
		Long:  "This will enroll the Agent into Fleet.",
		Args:  cobra.ExactArgs(2),
		Run: func(c *cobra.Command, args []string) {
			if err := enroll(streams, c, flags, args); err != nil {
				fmt.Fprintf(streams.Err, "%v\n", err)
				os.Exit(1)
			}
		},
	}

	cmd.Flags().StringP("certificate-authorities", "a", "", "Comma separated list of root certificate for server verifications")
	cmd.Flags().BoolP("force", "f", false, "Force overwrite the current and do not prompt for confirmation")

	return cmd
}

func enroll(streams *cli.IOStreams, cmd *cobra.Command, flags *globalFlags, args []string) error {
	config, err := config.LoadYAML(flags.PathConfigFile)
	if err != nil {
		return errors.Wrapf(err, "could not read configuration file %s", flags.PathConfigFile)
	}

	force, _ := cmd.Flags().GetBool("force")
	if !force {
		confirm, err := c.Confirm("This will replace your current settings. Do you want to continue?", true)
		if err != nil {
			return errors.Wrap(err, "problem reading prompt response")
		}
		if !confirm {
			fmt.Fprintln(streams.Out, "Enrollment was canceled by the user")
			return nil
		}
	}

	logger, err := logger.NewFromConfig(config)
	if err != nil {
		return err
	}

	url := args[0]
	enrollmentToken := args[1]

	caStr, _ := cmd.Flags().GetString("certificate-authorities")
	CAs := cli.StringToSlice(caStr)

	c, err := application.NewEnrollCmd(
		logger,
		url,
		CAs,
		enrollmentToken,
		"",
		nil,
		flags.PathConfigFile,
	)
	if err != nil {
		return err
	}

	err = c.Execute()
	if err != nil {
		return errors.Wrap(err, "fail to enroll")
	}

	fmt.Fprintln(streams.Out, "Successfully enrolled the Agent.")
	return nil
}
