// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/install"

	c "github.com/elastic/beats/v7/libbeat/common/cli"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/warn"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/cli"
)

func newInstallCommandWithArgs(flags *globalFlags, _ []string, streams *cli.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install Agent permanently on this system",
		Long: `
This will install Agent permanently on this system and will become managed by the systems service manager.

Unless all the require command-line parameters are provided or -f is used this command will ask questions on how you
would like the Agent to operate.
`,
		Run: func(c *cobra.Command, args []string) {
			if err := installCmd(streams, c, flags, args); err != nil {
				fmt.Fprintf(streams.Err, "%v\n", err)
				os.Exit(1)
			}
		},
	}

	cmd.Flags().StringP("kibana-url", "", "", "URL of Kibana to enroll Agent into Fleet")
	cmd.Flags().StringP("enrollment-token", "", "", "Enrollment token to use to enroll Agent into Fleet")
	addEnrollFlags(cmd)

	return cmd
}

func installCmd(streams *cli.IOStreams, cmd *cobra.Command, flags *globalFlags, args []string) error {
	if !install.HasRoot() {
		return fmt.Errorf("Error: unable to start install command, not executed with %s permissions.", install.PermissionUser)
	}
	installPath := install.Installed()
	if installPath != "" {
		return fmt.Errorf("Error: Elastic Agent is already installed at: %s", installPath)
	}

	warn.PrintNotGA(streams.Out)
	force, _ := cmd.Flags().GetBool("force")
	if !force {
		confirm, err := c.Confirm("Elastic Agent will be installed onto your system and will run as a service. Do you want to continue?", true)
		if err != nil {
			return fmt.Errorf("Error: problem reading prompt response")
		}
		if !confirm {
			return fmt.Errorf("Warn: Installation was cancelled by the user")
		}
	}

	err := install.Install()
	if err != nil {
		return fmt.Errorf("Error: %s", err)
	}
	err = install.StartService()
	if err != nil {
		fmt.Fprintf(streams.Out, "Installation of required system files was successful, but starting of the service failed.")
		return fmt.Errorf("Error: %s", err)
	}

	/*
		insecure, _ := cmd.Flags().GetBool("insecure")

		logger, err := logger.NewFromConfig("", cfg.Settings.LoggingConfig)
		if err != nil {
			return err
		}

		url := args[0]
		enrollmentToken := args[1]

		caStr, _ := cmd.Flags().GetString("certificate-authorities")
		CAs := cli.StringToSlice(caStr)

		caSHA256str, _ := cmd.Flags().GetString("ca-sha256")
		caSHA256 := cli.StringToSlice(caSHA256str)

		delay(defaultDelay)

		options := application.EnrollCmdOption{
			ID:                   "", // TODO(ph), This should not be an empty string, will clarify in a new PR.
			EnrollAPIKey:         enrollmentToken,
			URL:                  url,
			CAs:                  CAs,
			CASha256:             caSHA256,
			Insecure:             insecure,
			UserProvidedMetadata: make(map[string]interface{}),
			Staging:              staging,
		}

		c, err := application.NewEnrollCmd(
			logger,
			&options,
			pathConfigFile,
		)

		if err != nil {
			return err
		}

		err = c.Execute()
		signal := make(chan struct{})

		backExp := backoff.NewExpBackoff(signal, 60*time.Second, 10*time.Minute)

		for errors.Is(err, fleetapi.ErrTooManyRequests) {
			fmt.Fprintln(streams.Out, "Too many requests on the remote server, will retry in a moment.")
			backExp.Wait()
			fmt.Fprintln(streams.Out, "Retrying to enroll...")
			err = c.Execute()
		}

		close(signal)

		if err != nil {
			return errors.New(err, "fail to enroll")
		}

		fmt.Fprintln(streams.Out, "Successfully enrolled the Elastic Agent.")

		// skip restarting
		noRestart, _ := cmd.Flags().GetBool("no-restart")
		if noRestart {
			return nil
		}

		daemon := client.New()
		err = daemon.Connect(context.Background())
		if err == nil {
			defer daemon.Disconnect()
			err = daemon.Restart(context.Background())
			if err == nil {
				fmt.Fprintln(streams.Out, "Successfully triggered restart on running Elastic Agent.")
				return nil
			}
		}
		fmt.Fprintln(streams.Out, "Elastic Agent might not be running; unable to trigger restart")
		return nil
	*/
	return nil
}
