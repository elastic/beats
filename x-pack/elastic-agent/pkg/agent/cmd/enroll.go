// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cmd

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/control/client"

	"github.com/spf13/cobra"

	"github.com/elastic/beats/v7/libbeat/common/backoff"
	c "github.com/elastic/beats/v7/libbeat/common/cli"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/configuration"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/warn"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/cli"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/fleetapi"
)

var defaultDelay = 1 * time.Second

func newEnrollCommandWithArgs(flags *globalFlags, _ []string, streams *cli.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "enroll <kibana_url> <enrollment_token>",
		Short: "Enroll the Agent into Fleet",
		Long:  "This will enroll the Agent into Fleet.",
		Args:  cobra.ExactArgs(2),
		Run: func(c *cobra.Command, args []string) {
			if err := enroll(streams, c, flags, args); err != nil {
				fmt.Fprintf(streams.Err, "Error: %v\n", err)
				os.Exit(1)
			}
		},
	}

	addEnrollFlags(cmd)
	cmd.Flags().BoolP("force", "f", false, "Force overwrite the current and do not prompt for confirmation")
	cmd.Flags().Bool("no-restart", false, "Skip restarting the currently running daemon")

	// used by install command
	cmd.Flags().BoolP("from-install", "", false, "Set by install command to signal this was executed from install")
	cmd.Flags().MarkHidden("from-install")

	return cmd
}

func addEnrollFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("certificate-authorities", "a", "", "Comma separated list of root certificate for server verifications")
	cmd.Flags().StringP("ca-sha256", "p", "", "Comma separated list of certificate authorities hash pins used for certificate verifications")
	cmd.Flags().BoolP("insecure", "i", false, "Allow insecure connection to Kibana")
	cmd.Flags().StringP("staging", "", "", "Configures agent to download artifacts from a staging build")
}

func buildEnrollmentFlags(cmd *cobra.Command) []string {
	ca, _ := cmd.Flags().GetString("certificate-authorities")
	sha256, _ := cmd.Flags().GetString("ca-sha256")
	insecure, _ := cmd.Flags().GetBool("insecure")
	staging, _ := cmd.Flags().GetString("staging")

	args := []string{}
	if ca != "" {
		args = append(args, "--certificate-authorities")
		args = append(args, ca)
	}
	if sha256 != "" {
		args = append(args, "--ca-sha256")
		args = append(args, sha256)
	}
	if insecure {
		args = append(args, "--insecure")
	}
	if staging != "" {
		args = append(args, "--staging")
		args = append(args, staging)
	}
	return args
}

func enroll(streams *cli.IOStreams, cmd *cobra.Command, flags *globalFlags, args []string) error {
	fromInstall, _ := cmd.Flags().GetBool("from-install")
	if !fromInstall {
		warn.PrintNotGA(streams.Out)
	}

	pathConfigFile := flags.Config()
	rawConfig, err := application.LoadConfigFromFile(pathConfigFile)
	if err != nil {
		return errors.New(err,
			fmt.Sprintf("could not read configuration file %s", pathConfigFile),
			errors.TypeFilesystem,
			errors.M(errors.MetaKeyPath, pathConfigFile))
	}

	cfg, err := configuration.NewFromConfig(rawConfig)
	if err != nil {
		return errors.New(err,
			fmt.Sprintf("could not parse configuration file %s", pathConfigFile),
			errors.TypeFilesystem,
			errors.M(errors.MetaKeyPath, pathConfigFile))
	}

	staging, _ := cmd.Flags().GetString("staging")
	if staging != "" {
		if len(staging) < 8 {
			return errors.New(fmt.Errorf("invalid staging build hash; must be at least 8 characters"), "Error")
		}
	}

	force, _ := cmd.Flags().GetBool("force")
	if fromInstall {
		force = true
	}

	// prompt only when it is not forced and is already enrolled
	if !force && (cfg.Fleet != nil && cfg.Fleet.Enabled == true) {
		confirm, err := c.Confirm("This will replace your current settings. Do you want to continue?", true)
		if err != nil {
			return errors.New(err, "problem reading prompt response")
		}
		if !confirm {
			fmt.Fprintln(streams.Out, "Enrollment was cancelled by the user")
			return nil
		}
	}

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
	if noRestart || fromInstall {
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
}

func delay(t time.Duration) {
	<-time.After(time.Duration(rand.Int63n(int64(t))))
}
