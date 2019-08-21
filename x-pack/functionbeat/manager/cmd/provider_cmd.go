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
	"github.com/elastic/beats/x-pack/functionbeat/function/provider"
)

var output string

func initProviders() ([]provider.Provider, error) {
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

	var providers []provider.Provider
	for _, p := range cfg.Provider.GetFields() {
		isAvailable, err := provider.IsAvailable(p)
		if err != nil {
			return nil, err
		}
		if !isAvailable {
			continue
		}

		providerCfg, err := cfg.Provider.Child(p, -1)
		if err != nil {
			return nil, err
		}
		provider, err := provider.NewProvider(p, providerCfg)
		if err != nil {
			return nil, err
		}
		providers = append(providers, provider)
	}

	return providers, nil
}

func handler() (*cliHandler, error) {
	providers, err := initProviders()
	if err != nil {
		return nil, err
	}

	clis := make(map[string]provider.CLIManager)
	functionsByProvider := make(map[string]string)
	for _, provider := range providers {
		cli, err := provider.CLIManager()
		if err != nil {
			return nil, err
		}
		clis[provider.Name()] = cli

		enabledFunctions, err := provider.EnabledFunctions()
		if err != nil {
			return nil, err
		}

		for _, f := range enabledFunctions {
			functionsByProvider[f] = provider.Name()
		}
	}
	return newCLIHandler(clis, functionsByProvider, os.Stdout, os.Stderr), nil
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

			return h.BuildPackage(output)
		}),
	}

	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	defaultOutput := filepath.Join(dir, "package-{{.Provider}}.zip")
	cmd.Flags().StringVarP(&output, "output", "o", defaultOutput, "full path pattern to the package")
	return cmd
}

func genExportFunctionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "function",
		Short: "Export function template",
		Run: cli.RunWith(func(_ *cobra.Command, args []string) error {
			providers, err := initProviders()
			if err != nil {
				return err
			}

			for _, p := range providers {
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
			}
			return nil
		}),
	}
}
