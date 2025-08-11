// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package otelbeat

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/elasticsearchexporter"
	"github.com/spf13/cobra"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/provider/fileprovider"
	"go.opentelemetry.io/collector/exporter/debugexporter"
	"go.opentelemetry.io/collector/otelcol"
	"gopkg.in/yaml.v3"

	"github.com/elastic/beats/v7/libbeat/cfgfile"
	"github.com/elastic/beats/v7/libbeat/otelbeat/beatconverter"
	"github.com/elastic/beats/v7/libbeat/otelbeat/providers/fbprovider"
	"github.com/elastic/beats/v7/libbeat/otelbeat/providers/mbprovider"
	"github.com/elastic/beats/v7/libbeat/version"
	"github.com/elastic/beats/v7/x-pack/filebeat/fbreceiver"
	"github.com/elastic/beats/v7/x-pack/metricbeat/mbreceiver"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

var schemeMap = map[string]string{
	"filebeat":   "fb",
	"metricbeat": "mb",
}

func OTelCmd(beatname string) *cobra.Command {
	command := &cobra.Command{
		Short:  "Run this to start" + beatname + "with otel collector",
		Use:    "otel",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {

			// get beat configuration file
			beatCfg, _ := cmd.Flags().GetString("config")
			beatCfgFile := filepath.Join(cfgfile.GetPathConfig(), beatCfg)

			isOtelConfig, err := isOtelConfigFile(beatCfgFile)
			if err != nil {
				return err
			}

			// add scheme as prefix
			cfg := schemeMap[beatname] + ":" + beatCfg
			if isOtelConfig {
				cfg = "file:" + beatCfgFile
			}

			set := getCollectorSettings(cfg)
			col, err := otelcol.NewCollector(set)
			if err != nil {
				panic(fmt.Errorf("error initializing collector process: %w", err))
			}
			return col.Run(context.Background())
		},
	}

	command.Flags().String("config", beatname+"-otel.yml", "path to "+beatname+" config file")
	command.AddCommand(OTelInspectComand(beatname))
	return command
}

// Component initializes collector components
func getComponent() (otelcol.Factories, error) {
	receivers, err := otelcol.MakeFactoryMap(
		fbreceiver.NewFactory(),
		mbreceiver.NewFactory(),
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

func isOtelConfigFile(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, fmt.Errorf("error opening file %s: %w", path, err)
	}
	defer f.Close()

	var m mapstr.M
	if err = yaml.NewDecoder(f).Decode(&m); err != nil {
		return false, fmt.Errorf("error decoding file %s: %w", path, err)
	}

	for _, k := range []string{"receivers", "exporters", "service"} {
		if _, ok := m[k]; ok {
			return true, nil
		}
	}

	return false, nil
}

func getCollectorSettings(filename string) otelcol.CollectorSettings {
	// initialize collector settings
	info := component.BuildInfo{
		Command:     "otel",
		Description: "Beats OTel",
		Version:     version.GetDefaultVersion(),
	}

	return otelcol.CollectorSettings{
		BuildInfo: info,
		Factories: getComponent,
		ConfigProviderSettings: otelcol.ConfigProviderSettings{
			ResolverSettings: confmap.ResolverSettings{
				URIs: []string{filename},
				ProviderFactories: []confmap.ProviderFactory{
					fileprovider.NewFactory(),
					fbprovider.NewFactory(),
					mbprovider.NewFactory(),
				},
				ConverterFactories: []confmap.ConverterFactory{
					beatconverter.NewFactory(),
				},
			},
		},
	}
}
