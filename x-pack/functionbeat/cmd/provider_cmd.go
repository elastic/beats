// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/elastic/beats/libbeat/cmd/instance"
	"github.com/elastic/beats/libbeat/common/cli"
	"github.com/elastic/beats/x-pack/functionbeat/config"
	"github.com/elastic/beats/x-pack/functionbeat/provider"
)

var output string

func initProvider() (provider.Provider, error) {
	b, err := instance.NewInitializedBeat(instance.Settings{Name: Name})
	if err != nil {
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

	return provider.NewProvider(cfg)
}

// TODO: Add List() subcommand.
func handler() (*cliHandler, error) {
	provider, err := initProvider()
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

func genCLICmd(use, short string, fn func(*cliHandler, []string) error) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		Run: cli.RunWith(func(_ *cobra.Command, args []string) error {
			h, err := handler()
			if err != nil {
				return err
			}
			return fn(h, args)
		}),
	}
}

func genDeployCmd() *cobra.Command {
	return genCLICmd("deploy", "Deploy a function", (*cliHandler).Deploy)
}

func genUpdateCmd() *cobra.Command {
	return genCLICmd("update", "Update a function", (*cliHandler).Update)
}

func genRemoveCmd() *cobra.Command {
	return genCLICmd("remove", "Remove a function", (*cliHandler).Remove)
}

func genPackageCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "package",
		Short: "Package the configuration and the executable in a zip",
		Run: cli.RunWith(func(cmd *cobra.Command, args []string) error {
			h, err := handler()
			if err != nil {
				return err
			}

			if len(output) == 0 {
				dir, err := os.Getwd()
				if err != nil {
					return err
				}

				output = filepath.Join(dir, "package.zip")
			}

			return h.BuildPackage(output)
		}),
	}
	cmd.Flags().StringVarP(&output, "output", "o", "", "full path to the package")
	return cmd
}

func genExportFunctionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "function",
		Short: "Export function template",
		Run: cli.RunWith(func(_ *cobra.Command, args []string) error {
			p, err := initProvider()
			if err != nil {
				return err
			}
			builder, err := p.TemplateBuilder()
			if err != nil {
				return err
			}
			for _, name := range args {
				template, err := builder.RawTemplate(name)
				if err != nil {
					return fmt.Errorf("error generating raw template for %s: %+v", name, err)
				}
				fmt.Println(template)
			}
			return nil
		}),
	}
}
