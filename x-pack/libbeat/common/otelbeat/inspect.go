// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package otelbeat

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	"go.opentelemetry.io/collector/confmap"
	"gopkg.in/yaml.v3"

	"github.com/elastic/beats/v7/libbeat/cfgfile"
	"github.com/elastic/beats/v7/libbeat/otelbeat"
	"github.com/elastic/beats/v7/libbeat/otelbeat/beatconverter"
)

func OTelInspectComand(beatname string) *cobra.Command {
	command := &cobra.Command{
		Short: "Run this command to inspect the OTel configuration translated from the Beats config",
		Use:   "inspect",
		RunE: func(cmd *cobra.Command, args []string) error {
			// get beat configuration file
			beatCfg, _ := cmd.Flags().GetString("config")
			beatCfgFile := filepath.Join(cfgfile.GetPathConfig(), beatCfg)

			isOtelConfig, err := isOtelConfigFile(beatCfgFile)
			if err != nil {
				return fmt.Errorf("error reading config file: %w", err)
			}
			if isOtelConfig {
				// the user has defined a custom OTel config. Skip the rest of the logic.
				fmt.Fprintln(cmd.OutOrStdout(), "Custom OTel config detected. Skipping translation")
				return nil
			}

			provider, err := otelbeat.NewFactory(beatname)
			if err != nil {
				return fmt.Errorf("error creating %s factory: %w", beatname, err)
			}

			retrieved, err := provider.Create(confmap.ProviderSettings{}).Retrieve(cmd.Context(), schemeMap[beatname]+":"+beatCfg, nil)
			if err != nil {
				return fmt.Errorf("error getting the config from provider: %w", err)
			}

			conf, err := retrieved.AsConf()
			if err != nil {
				return fmt.Errorf("error retrieving confmap: %w", err)
			}

			converter := beatconverter.NewFactory().Create(confmap.ConverterSettings{})

			if err = converter.Convert(cmd.Context(), conf); err != nil {
				return fmt.Errorf("error converting config: %w", err)
			}

			b, err := yaml.Marshal(conf.ToStringMap())
			if err != nil {
				return fmt.Errorf("error marshalling yaml: %w", err)
			}

			fmt.Fprintln(cmd.OutOrStdout(), string(b))
			return nil
		},
	}

	command.Flags().String("config", beatname+"-otel.yml", "path to "+beatname+" config file")
	return command
}
