// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/paths"

	"github.com/spf13/cobra"

	c "github.com/elastic/beats/v7/libbeat/common/cli"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/configuration"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/warn"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/cli"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/config"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
)

func newEnrollCommandWithArgs(_ []string, streams *cli.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "enroll",
		Short: "Enroll the Agent into Fleet",
		Long:  "This will enroll the Agent into Fleet.",
		Run: func(c *cobra.Command, args []string) {
			if err := enroll(streams, c, args); err != nil {
				fmt.Fprintf(streams.Err, "Error: %v\n", err)
				os.Exit(1)
			}
		},
	}

	addEnrollFlags(cmd)
	cmd.Flags().BoolP("force", "f", false, "Force overwrite the current and do not prompt for confirmation")

	// used by install command
	cmd.Flags().BoolP("from-install", "", false, "Set by install command to signal this was executed from install")
	cmd.Flags().MarkHidden("from-install")

	return cmd
}

func addEnrollFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("url", "", "", "URL to enroll Agent into Fleet")
	cmd.Flags().StringP("enrollment-token", "t", "", "Enrollment token to use to enroll Agent into Fleet")
	cmd.Flags().StringP("fleet-server-es", "", "", "Start and run a Fleet Server along side this Elastic Agent connecting to the provided elasticsearch")
	cmd.Flags().StringP("fleet-server-es-ca", "", "", "Path to certificate authority to use with communicate with elasticsearch")
	cmd.Flags().StringP("fleet-server-service-token", "", "", "Service token to use for communication with elasticsearch")
	cmd.Flags().StringP("fleet-server-policy", "", "", "Start and run a Fleet Server on this specific policy")
	cmd.Flags().StringP("fleet-server-host", "", "", "Fleet Server HTTP binding host (overrides the policy)")
	cmd.Flags().Uint16P("fleet-server-port", "", 0, "Fleet Server HTTP binding port (overrides the policy)")
	cmd.Flags().StringP("fleet-server-cert", "", "", "Certificate to use for exposed Fleet Server HTTPS endpoint")
	cmd.Flags().StringP("fleet-server-cert-key", "", "", "Private key to use for exposed Fleet Server HTTPS endpoint")
	cmd.Flags().BoolP("fleet-server-insecure-http", "", false, "Expose Fleet Server over HTTP (not recommended; insecure)")
	cmd.Flags().StringP("certificate-authorities", "a", "", "Comma separated list of root certificate for server verifications")
	cmd.Flags().StringP("ca-sha256", "p", "", "Comma separated list of certificate authorities hash pins used for certificate verifications")
	cmd.Flags().BoolP("insecure", "i", false, "Allow insecure connection to fleet-server")
	cmd.Flags().StringP("staging", "", "", "Configures agent to download artifacts from a staging build")
}

func buildEnrollmentFlags(cmd *cobra.Command, url string, token string) []string {
	if url == "" {
		url, _ = cmd.Flags().GetString("url")
	}
	if token == "" {
		token, _ = cmd.Flags().GetString("enrollment-token")
	}
	fServer, _ := cmd.Flags().GetString("fleet-server-es")
	fElasticSearchCA, _ := cmd.Flags().GetString("fleet-server-es-ca")
	fServiceToken, _ := cmd.Flags().GetString("fleet-server-service-token")
	fPolicy, _ := cmd.Flags().GetString("fleet-server-policy")
	fHost, _ := cmd.Flags().GetString("fleet-server-host")
	fPort, _ := cmd.Flags().GetUint16("fleet-server-port")
	fCert, _ := cmd.Flags().GetString("fleet-server-cert")
	fCertKey, _ := cmd.Flags().GetString("fleet-server-cert-key")
	fInsecure, _ := cmd.Flags().GetBool("fleet-server-insecure-http")
	ca, _ := cmd.Flags().GetString("certificate-authorities")
	sha256, _ := cmd.Flags().GetString("ca-sha256")
	insecure, _ := cmd.Flags().GetBool("insecure")
	staging, _ := cmd.Flags().GetString("staging")

	args := []string{}
	if url != "" {
		args = append(args, "--url")
		args = append(args, url)
	}
	if token != "" {
		args = append(args, "--enrollment-token")
		args = append(args, token)
	}
	if fServer != "" {
		args = append(args, "--fleet-server-es")
		args = append(args, fServer)
	}
	if fElasticSearchCA != "" {
		args = append(args, "--fleet-server-es-ca")
		args = append(args, fElasticSearchCA)
	}
	if fServiceToken != "" {
		args = append(args, "--fleet-server-service-token")
		args = append(args, fServiceToken)
	}
	if fPolicy != "" {
		args = append(args, "--fleet-server-policy")
		args = append(args, fPolicy)
	}
	if fHost != "" {
		args = append(args, "--fleet-server-host")
		args = append(args, fHost)
	}
	if fPort > 0 {
		args = append(args, "--fleet-server-port")
		args = append(args, strconv.Itoa(int(fPort)))
	}
	if fCert != "" {
		args = append(args, "--fleet-server-cert")
		args = append(args, fCert)
	}
	if fCertKey != "" {
		args = append(args, "--fleet-server-cert-key")
		args = append(args, fCertKey)
	}
	if fInsecure {
		args = append(args, "--fleet-server-insecure-http")
	}
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

func enroll(streams *cli.IOStreams, cmd *cobra.Command, args []string) error {
	fromInstall, _ := cmd.Flags().GetBool("from-install")
	if !fromInstall {
		warn.PrintNotGA(streams.Out)
	}

	pathConfigFile := paths.ConfigFile()
	rawConfig, err := config.LoadFile(pathConfigFile)
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
	if !force && (cfg.Fleet != nil && cfg.Fleet.Enabled) {
		confirm, err := c.Confirm("This will replace your current settings. Do you want to continue?", true)
		if err != nil {
			return errors.New(err, "problem reading prompt response")
		}
		if !confirm {
			fmt.Fprintln(streams.Out, "Enrollment was cancelled by the user")
			return nil
		}
	}

	// enroll is invoked either manually or from install with redirected IO
	// no need to log to file
	cfg.Settings.LoggingConfig.ToFiles = false
	cfg.Settings.LoggingConfig.ToStderr = true

	logger, err := logger.NewFromConfig("", cfg.Settings.LoggingConfig, false)
	if err != nil {
		return err
	}

	insecure, _ := cmd.Flags().GetBool("insecure")
	url, _ := cmd.Flags().GetString("url")
	enrollmentToken, _ := cmd.Flags().GetString("enrollment-token")
	fServer, _ := cmd.Flags().GetString("fleet-server-es")
	fElasticSearchCA, _ := cmd.Flags().GetString("fleet-server-es-ca")
	fServiceToken, _ := cmd.Flags().GetString("fleet-server-service-token")
	fPolicy, _ := cmd.Flags().GetString("fleet-server-policy")
	fHost, _ := cmd.Flags().GetString("fleet-server-host")
	fPort, _ := cmd.Flags().GetUint16("fleet-server-port")
	fCert, _ := cmd.Flags().GetString("fleet-server-cert")
	fCertKey, _ := cmd.Flags().GetString("fleet-server-cert-key")
	fInsecure, _ := cmd.Flags().GetBool("fleet-server-insecure-http")

	caStr, _ := cmd.Flags().GetString("certificate-authorities")
	CAs := cli.StringToSlice(caStr)
	caSHA256str, _ := cmd.Flags().GetString("ca-sha256")
	caSHA256 := cli.StringToSlice(caSHA256str)

	ctx := handleSignal(context.Background())

	options := enrollCmdOption{
		ID:                   "", // TODO(ph), This should not be an empty string, will clarify in a new PR.
		EnrollAPIKey:         enrollmentToken,
		URL:                  url,
		CAs:                  CAs,
		CASha256:             caSHA256,
		Insecure:             insecure,
		UserProvidedMetadata: make(map[string]interface{}),
		Staging:              staging,
		FleetServer: enrollCmdFleetServerOption{
			ConnStr:         fServer,
			ElasticsearchCA: fElasticSearchCA,
			ServiceToken:    fServiceToken,
			PolicyID:        fPolicy,
			Host:            fHost,
			Port:            fPort,
			Cert:            fCert,
			CertKey:         fCertKey,
			Insecure:        fInsecure,
			SpawnAgent:      !fromInstall,
		},
	}

	c, err := newEnrollCmd(
		logger,
		&options,
		pathConfigFile,
	)

	if err != nil {
		return err
	}

	err = c.Execute(ctx)
	if err == nil {
		fmt.Fprintln(streams.Out, "Successfully enrolled the Elastic Agent.")
	}
	return err
}

func handleSignal(ctx context.Context) context.Context {
	ctx, cfunc := context.WithCancel(ctx)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGKILL, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		select {
		case <-sigs:
			cfunc()
		case <-ctx.Done():
		}

		signal.Stop(sigs)
		close(sigs)
	}()

	return ctx
}
