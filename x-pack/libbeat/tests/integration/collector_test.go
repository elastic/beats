// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration && !agentbeat

package integration

import (
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/elasticsearchexporter"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/exporter/debugexporter"
	"go.opentelemetry.io/collector/otelcol"

	"github.com/elastic/beats/v7/libbeat/otelbeat/beatconverter"
	"github.com/elastic/beats/v7/libbeat/otelbeat/providers/fbprovider"
	"github.com/elastic/beats/v7/x-pack/filebeat/fbreceiver"
)

var schemeMap = map[string]string{
	"filebeat": "fb",
}

// NewTestCollector configures and returns an otel collector intended for testing only
func NewTestCollector(beatname string, configPath string) (*otelcol.Collector, error) {
	// adds scheme name as prefix
	beatCfg := schemeMap[beatname] + ":" + configPath

	set := getCollectorSettings(beatCfg)
	return otelcol.NewCollector(set)
}

// Component initializes collector components
func getComponent() (otelcol.Factories, error) {
	receivers, err := otelcol.MakeFactoryMap(
		fbreceiver.NewFactory(),
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
		Receivers: receivers,
		Exporters: exporters,
	}, nil

}

func getCollectorSettings(filename string) otelcol.CollectorSettings {
	// initialize collector settings
	info := component.BuildInfo{
		Command:     "otel-test",
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
