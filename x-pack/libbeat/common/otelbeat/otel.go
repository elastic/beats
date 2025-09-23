// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package otelbeat

import (
	"context"
	"fmt"

	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/elasticsearchexporter"
	"github.com/spf13/cobra"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/exporter/debugexporter"
	"go.opentelemetry.io/collector/otelcol"

	"github.com/elastic/beats/v7/libbeat/otelbeat/beatconverter"
	"github.com/elastic/beats/v7/libbeat/otelbeat/providers/fbprovider"
	"github.com/elastic/beats/v7/x-pack/filebeat/fbreceiver"
<<<<<<< HEAD
	"github.com/elastic/elastic-agent-libs/logp"
=======
	"github.com/elastic/beats/v7/x-pack/metricbeat/mbreceiver"
	"github.com/elastic/beats/v7/x-pack/otel/exporter/logstashexporter"
	"github.com/elastic/beats/v7/x-pack/otel/processor/beatprocessor"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/opentelemetry-collector-components/extension/beatsauthextension"
>>>>>>> d3be9bf15 (Remove settings on ES exporter config that no longer function (#46428))
)

var schemeMap = map[string]string{
	"filebeat": "fb",
}

func OTelCmd(beatname string) *cobra.Command {
	command := &cobra.Command{
		Short:  "Run this to start" + beatname + "with otel collector",
		Use:    "otel",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			logger := logp.NewLogger(beatname + "-otel-mode")
			logger.Info("This mode is experimental and unsupported")

			// get beat configuration file
			beatCfg, _ := cmd.Flags().GetString("config")
			// adds scheme name as prefix
			beatCfg = schemeMap[beatname] + ":" + beatCfg

			set := getCollectorSettings(beatCfg)
			col, err := otelcol.NewCollector(set)
			if err != nil {
				panic(fmt.Errorf("error initializing collector process: %w", err))
			}
			return col.Run(context.Background())
		},
	}

	command.Flags().String("config", beatname+"-otel.yml", "path to filebeat config file")
	return command
}

// Component initializes collector components
func getComponent() (otelcol.Factories, error) {
	receivers, err := otelcol.MakeFactoryMap(
		fbreceiver.NewFactory(),
<<<<<<< HEAD
=======
		mbreceiver.NewFactory(),
	)
	if err != nil {
		return otelcol.Factories{}, nil //nolint:nilerr //ignoring this error
	}

	extensions, err := otelcol.MakeFactoryMap(
		beatsauthextension.NewFactory(),
	)
	if err != nil {
		return otelcol.Factories{}, nil //nolint:nilerr //ignoring this error
	}

	processors, err := otelcol.MakeFactoryMap(
		beatprocessor.NewFactory(),
>>>>>>> d3be9bf15 (Remove settings on ES exporter config that no longer function (#46428))
	)
	if err != nil {
		return otelcol.Factories{}, nil //nolint:nilerr //ignoring this error
	}

	exporters, err := otelcol.MakeFactoryMap(
		debugexporter.NewFactory(),
		elasticsearchexporter.NewFactory(),
	)
	if err != nil {
		return otelcol.Factories{}, nil //nolint:nilerr //ignoring this error
	}

	return otelcol.Factories{
<<<<<<< HEAD
		Receivers: receivers,
		Exporters: exporters,
=======
		Receivers:  receivers,
		Processors: processors,
		Exporters:  exporters,
		Extensions: extensions,
>>>>>>> d3be9bf15 (Remove settings on ES exporter config that no longer function (#46428))
	}, nil

}

func getCollectorSettings(filename string) otelcol.CollectorSettings {
	// initialize collector settings
	info := component.BuildInfo{
		Command:     "otel",
		Description: "Beats OTel",
		Version:     "9.0.0",
	}

	return otelcol.CollectorSettings{
		BuildInfo: info,
		Factories: getComponent,
		ConfigProviderSettings: otelcol.ConfigProviderSettings{
			ResolverSettings: confmap.ResolverSettings{
				URIs: []string{filename},
				ProviderFactories: []confmap.ProviderFactory{
					fbprovider.NewFactory(),
				},
				ConverterFactories: []confmap.ConverterFactory{
					beatconverter.NewFactory(),
				},
			},
		},
	}
}
