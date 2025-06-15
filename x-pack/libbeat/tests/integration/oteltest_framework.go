// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration && !agentbeat

package integration

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/otelcol"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest"
	"go.uber.org/zap/zaptest/observer"

	"github.com/elastic/beats/v7/libbeat/otelbeat/beatconverter"
	"github.com/elastic/beats/v7/libbeat/otelbeat/providers/fbprovider"
	"github.com/elastic/beats/v7/libbeat/tests/integration"
	"github.com/elastic/beats/v7/x-pack/libbeat/common/otelbeat"
)

var schemeMap = map[string]string{
	"filebeat": "fb",
}

type TestCollector struct {
	t            *testing.T
	tempDir      string
	otelcol      *otelcol.Collector
	wg           sync.WaitGroup
	configFile   string
	beatname     string
	observedLogs *observer.ObservedLogs
}

// NewTestCollector configures and returns an instance of otel collector intended for testing only
func NewTestCollector(t *testing.T, beatname string, config string) (*TestCollector, error) {
	// create a temp dir at beat/build/integration-tests
	tempDir := integration.CreateTempDir(t)

	// create a configfile
	configFile := filepath.Join(tempDir, beatname+".yml")

	testCol := &TestCollector{
		t:          t,
		tempDir:    tempDir,
		wg:         sync.WaitGroup{},
		configFile: configFile,
		beatname:   beatname,
	}

	// write configuration to a file
	err := testCol.reloadConfig(config)
	return testCol, err
}

// NewTestStartCollector configures and starts an otel collector intended for testing only
func NewTestStartCollector(t *testing.T, beatname string, config string) (*TestCollector, error) {
	otelcol, err := NewTestCollector(t, beatname, config)
	if err != nil {
		return nil, err
	}
	err = otelcol.Run()
	if err != nil {
		return nil, err
	}

	return otelcol, err
}

// reloadConfig reloads configuration with which collector should be started.
// Note: A running collector will not pick the new config until it is stopped and started
func (c *TestCollector) reloadConfig(config string) error {
	// write configuration to a file
	if err := os.WriteFile(c.configFile, []byte(config), 0o644); err != nil {
		return fmt.Errorf("cannot create config file '%s': %s", c.configFile, err)
	}
	// adds scheme name as prefix to the configfile
	beatCfg := schemeMap[c.beatname] + ":" + c.configFile
	// get collector settings
	set, observedLogs := getCollectorSettings(beatCfg)

	c.observedLogs = observedLogs
	// get new collector instance
	otelcol, err := otelcol.NewCollector(set)
	c.otelcol = otelcol
	return err
}

// ReloadCollectorWithConfig reloads the collector with given config
func (c *TestCollector) ReloadCollectorWithConfig(config string) error {
	c.t.Helper()
	// shutdown collector if it is running
	c.Shutdown()
	err := c.reloadConfig(config)
	if err != nil {
		return err
	}
	return c.Run()
}

func (c *TestCollector) GetTempDir() string {
	return c.tempDir
}

// Run starts the otel collector
func (c *TestCollector) Run() error {
	c.t.Helper()

	wg := sync.WaitGroup{}
	var err error
	wg.Add(1)
	go func() {
		defer wg.Done()
		err = c.otelcol.Run(c.t.Context())
	}()

	c.t.Cleanup(func() {
		c.Shutdown()
		return
	})

	return err
}

func (c *TestCollector) Shutdown() {
	c.otelcol.Shutdown()
}

// IsCollectorHealthy returns true if collector is healthy
func (c *TestCollector) IsCollectorHealthy(config string) bool {
	// TODO
	return true
}

func getCollectorSettings(filename string) (otelcol.CollectorSettings, *observer.ObservedLogs) {
	// initialize collector settings
	info := component.BuildInfo{
		Command:     "otel-test",
		Description: "Beats OTel",
		Version:     "9.1.0",
	}

	zapCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		&zaptest.Discarder{},
		zapcore.DebugLevel,
	)
	observed, zapLogs := observer.New(zapcore.DebugLevel)

	return otelcol.CollectorSettings{
		BuildInfo: info,
		Factories: otelbeat.GetComponent,
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
		LoggingOptions: []zap.Option{zap.WrapCore(func(zapcore.Core) zapcore.Core {
			return zapcore.NewTee(zapCore, observed)
		})},
	}, zapLogs
}
