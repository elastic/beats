// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/otelcol"

	"github.com/elastic/beats/v7/libbeat/otelcommon"
	"github.com/elastic/beats/v7/libbeat/otelcommon/converters"
	"github.com/elastic/beats/v7/libbeat/otelcommon/providers/fbprovider"
)

func OtelCmd() *cobra.Command {
	command := &cobra.Command{
		Short: "Run this to start filebeat with otel collector",
		Use:   "otel",
		RunE: func(cmd *cobra.Command, args []string) error {
			info := component.BuildInfo{
				Command:     "otel",
				Description: "Beats OTel",
				Version:     "9.0.0",
			}

			// get filebeat configuration file
			filebeatCfg, _ := cmd.Flags().GetString("config")
			// adds scheme name as prefix
			filebeatCfg = "fb:" + filebeatCfg

			// initialize collector settings
			set := otelcol.CollectorSettings{
				BuildInfo: info,
				Factories: otelcommon.Component,
				ConfigProviderSettings: otelcol.ConfigProviderSettings{
					ResolverSettings: confmap.ResolverSettings{
						URIs:          []string{filebeatCfg},
						DefaultScheme: "fb",
						ProviderFactories: []confmap.ProviderFactory{
							fbprovider.NewFactory(),
						},
						ConverterFactories: []confmap.ConverterFactory{
							converters.NewFactory(),
						},
					},
				},
			}

			col, err := otelcol.NewCollector(set)
			if err != nil {
				panic(fmt.Errorf("error initializing collector process: %w", err))
			}
			return col.Run(context.Background())
		},
	}

	command.Flags().String("config", "filebeat-otel.yml", "path to filebeat config file")
	return command
}
