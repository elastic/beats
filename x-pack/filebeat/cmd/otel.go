// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cmd

import (
	"context"
	"fmt"

	"github.com/elastic/beats/v7/libbeat/otelcommon/providers/fbprovider"
	"github.com/elastic/beats/v7/x-pack/filebeat/fbreceiver"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/elasticsearchexporter"
	"github.com/spf13/cobra"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/debugexporter"
	"go.opentelemetry.io/collector/otelcol"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/batchprocessor"
	"go.opentelemetry.io/collector/processor/memorylimiterprocessor"
	"go.opentelemetry.io/collector/receiver"
)

// initialize collector components
func components() (otelcol.Factories, error) {
	receivers, err := receiver.MakeFactoryMap(
		fbreceiver.NewFactory(),
	)
	if err != nil {
		return otelcol.Factories{}, nil
	}

	exporters, err := exporter.MakeFactoryMap(
		debugexporter.NewFactory(),
		elasticsearchexporter.NewFactory(),
	)
	if err != nil {
		return otelcol.Factories{}, nil
	}

	processors, err := processor.MakeFactoryMap(
		batchprocessor.NewFactory(),
		memorylimiterprocessor.NewFactory(),
	)
	if err != nil {
		return otelcol.Factories{}, nil
	}

	return otelcol.Factories{
		Receivers:  receivers,
		Exporters:  exporters,
		Processors: processors,
	}, nil

}

func OtelCmd() *cobra.Command {
	command := &cobra.Command{
		Short: "Run this to start filebeat as a otel",
		Use:   "otel",
		RunE: func(cmd *cobra.Command, args []string) error {
			info := component.BuildInfo{
				Command:     "otel",
				Description: "Beats OTel",
				Version:     "9.0.0",
			}

			// get filebeat configuration file
			filebeatCfg, _ := cmd.Flags().GetString("config")

			// initialize collector settings
			set := otelcol.CollectorSettings{
				BuildInfo: info,
				Factories: components,
				ConfigProviderSettings: otelcol.ConfigProviderSettings{
					ResolverSettings: confmap.ResolverSettings{
						URIs:          []string{filebeatCfg},
						DefaultScheme: "file",
						ProviderFactories: []confmap.ProviderFactory{
							fbprovider.NewFactory(),
						},
					},
				},
			}

			col, err := otelcol.NewCollector(set)
			if err != nil {
				panic(fmt.Errorf("error initializting collector process: %w", err))
			}
			return col.Run(context.Background())
		},
	}

	command.Flags().String("config", "filebeat.yml", "pass filebeat config")
	return command
}
