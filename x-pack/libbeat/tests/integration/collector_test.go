// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration && !agentbeat

package integration

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/elasticsearchexporter"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/exporter/debugexporter"
	"go.opentelemetry.io/collector/otelcol"

	"github.com/elastic/beats/v7/libbeat/otelbeat/beatconverter"
	"github.com/elastic/beats/v7/libbeat/otelbeat/providers/fbprovider"
	"github.com/elastic/beats/v7/libbeat/tests/integration"
	"github.com/elastic/beats/v7/x-pack/filebeat/fbreceiver"
)

var schemeMap = map[string]string{
	"filebeat": "fb",
}

type TestCollector struct {
	t       *testing.T
	tempDir string
	otelcol *otelcol.Collector
	wg      sync.WaitGroup
}

// NewTestCollector configures and returns an otel collector intended for testing only
// It accepts beatname and configuration
func NewTestCollector(t *testing.T, beatname string, config string) (*TestCollector, error) {

	// create a temp dir
	tempDir := integration.CreateTempDir(t)
	// stdoutFile, err := os.Create(filepath.Join(tempDir, "stdout"))
	// require.NoError(t, err, "error creating stdout file")
	// stderrFile, err := os.Create(filepath.Join(tempDir, "stderr"))
	// require.NoError(t, err, "error creating stderr file")

	// create a config file
	configFile := filepath.Join(tempDir, beatname+".yml")
	// write configuration to a file
	if err := os.WriteFile(configFile, []byte(config), 0o644); err != nil {
		t.Fatalf("cannot create config file '%s': %s", configFile, err)
	}
	// adds scheme name as prefix to the configfile
	beatCfg := schemeMap[beatname] + ":" + configFile
	// get collector settings
	set := getCollectorSettings(beatCfg)
	// get new collector instance
	otelcol, err := otelcol.NewCollector(set)

	return &TestCollector{
		t:       t,
		tempDir: tempDir,
		otelcol: otelcol,
		wg:      sync.WaitGroup{},
	}, err
}

// NewTestCollector configures and returns an otel collector intended for testing only
// It accepts beatname and configuration
func NewTestStartCollector(t *testing.T, beatname string, config string) (*TestCollector, error) {

	otelcol, err := NewTestCollector(t, beatname, config)
	if err != nil {
		return nil, err
	}
	err = otelcol.Run()
	if err != nil {
		return nil, err
	}

	t.Cleanup(func() {
		otelcol.Shutdown()
		if !t.Failed() {
			return
		}
	})

	return otelcol, err
}

func (c *TestCollector) GetTempDir() string {
	return c.tempDir
}

func (c *TestCollector) Run() error {
	wg := sync.WaitGroup{}
	var err error
	go func() {
		wg.Add(1)
		defer wg.Done()
		err = c.otelcol.Run(c.t.Context())

	}()
	return err
}
func (c *TestCollector) Shutdown() {
	c.otelcol.Shutdown()
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
