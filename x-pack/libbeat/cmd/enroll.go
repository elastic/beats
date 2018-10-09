// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cmd

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/elastic/beats/libbeat/cmd/instance"
	"github.com/elastic/beats/libbeat/common/cli"
	"github.com/elastic/beats/x-pack/libbeat/management"
	"github.com/elastic/beats/x-pack/libbeat/management/api"
)

func getBeat(name, version string) (*instance.Beat, error) {
	b, err := instance.NewBeat(name, "", version)

	if err != nil {
		return nil, fmt.Errorf("error creating beat: %s", err)
	}

	if err = b.Init(); err != nil {
		return nil, fmt.Errorf("error initializing beat: %s", err)
	}

	return b, nil
}

func genEnrollCmd(name, version string) *cobra.Command {
	var username, password string

	enrollCmd := cobra.Command{
		Use:   "enroll <kibana_url> [<enrollment_token>]",
		Short: "Enroll in Kibana for Central Management",
		Long: `This will enroll in  Kibana Beats Central Management. If you pass an enrollment token
		it will be used. You can also enroll using a username and password combination.`,
		Args: cobra.RangeArgs(1, 2),
		Run: cli.RunWith(func(cmd *cobra.Command, args []string) error {
			beat, err := getBeat(name, version)
			kibanaURL := args[0]

			if username == "" && len(args) == 1 {
				return errors.New("You should pass either an enrollment token or use --username flag")
			}

			var enrollmentToken string
			if len(args) == 2 {
				// use given enrollment token
				enrollmentToken = args[1]
				if err != nil {
					return err
				}
			} else {
				// retrieve an enrollment token using username/password
				config, err := api.ConfigFromURL(kibanaURL)
				if err != nil {
					return err
				}

				// pass username/password
				config.IgnoreVersion = true
				config.Username = username
				config.Password, err = cli.ReadPassword(password)
				if err != nil {
					return err
				}

				client, err := api.NewClient(config)
				if err != nil {
					return err
				}
				enrollmentToken, err = client.CreateEnrollmentToken()
				if err != nil {
					return errors.Wrap(err, "Creating a new enrollment token")
				}
			}

			if err = management.Enroll(beat, kibanaURL, enrollmentToken); err != nil {
				return errors.Wrap(err, "Error while enrolling")
			}

			fmt.Println("Enrolled and ready to retrieve settings from Kibana")
			return nil
		}),
	}

	enrollCmd.Flags().StringVar(&username, "username", "elastic", "Username to use when enrolling without token")
	enrollCmd.Flags().StringVar(&password, "password", "stdin", "Method to read the password to use when enrolling without token (stdin or env:VAR_NAME)")

	return &enrollCmd
}
