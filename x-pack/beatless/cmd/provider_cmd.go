// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cmd

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/elastic/beats/libbeat/cmd/instance"
	"github.com/elastic/beats/libbeat/common/cli"
	"github.com/elastic/beats/x-pack/beatless/config"
	"github.com/elastic/beats/x-pack/beatless/provider"
)

// TODO: Add List() subcommand.
func handler() (*cliHandler, error) {
	b, err := instance.NewBeat(Name, "", "")
	if err != nil {
		return nil, err
	}

	if err = b.InitWithSettings(instance.Settings{
		ConfigOverrides: config.ConfigOverrides,
	}); err != nil {
		return nil, err
	}

	c, err := b.BeatConfig()
	if err != nil {
		return nil, err
	}

	cfg := &config.DefaultConfig
	if err := c.Unpack(cfg); err != nil {
		return nil, err
	}

	provider, err := provider.NewProvider(cfg)
	if err != nil {
		return nil, err
	}

	cli, err := provider.CLIManager()
	if err != nil {
		return nil, err
	}
	handler := newCLIHandler(cli, os.Stdout, os.Stderr)
	return handler, nil
}

func genDeployCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deploy",
		Short: "Deploy a function",
		Run: cli.RunWith(func(cmd *cobra.Command, args []string) error {
			h, err := handler()
			if err != nil {
				return err
			}
			return h.Deploy(args)
		}),
	}
	return cmd
}

func genUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update a function",
		Run: cli.RunWith(func(cmd *cobra.Command, args []string) error {
			h, err := handler()
			if err != nil {
				return err
			}
			return h.Update(args)
		}),
	}
	return cmd
}

func genRemoveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove",
		Short: "Remove a function",
		Run: cli.RunWith(func(cmd *cobra.Command, args []string) error {
			h, err := handler()
			if err != nil {
				return err
			}
			return h.Remove(args)
		}),
	}
	return cmd
}
