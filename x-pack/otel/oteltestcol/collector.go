// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package oteltestcol

import (
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"testing"
	"time"

	libbeattesting "github.com/elastic/beats/v7/libbeat/testing"
	"github.com/elastic/beats/v7/libbeat/version"
	"github.com/elastic/beats/v7/x-pack/auditbeat/abreceiver"
	"github.com/elastic/beats/v7/x-pack/filebeat/fbreceiver"
	"github.com/elastic/beats/v7/x-pack/heartbeat/hbreceiver"
	"github.com/elastic/beats/v7/x-pack/metricbeat/mbreceiver"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/osqreceiver"
	"github.com/elastic/beats/v7/x-pack/otel/exporter/logstashexporter"
	"github.com/elastic/beats/v7/x-pack/otel/extension/beatsauthextension"
	"github.com/elastic/beats/v7/x-pack/otel/extension/elasticsearchstorage"
	"github.com/elastic/beats/v7/x-pack/otel/processor/beatprocessor"
	"github.com/elastic/beats/v7/x-pack/packetbeat/pbreceiver"

	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/elasticsearchexporter"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/kafkaexporter"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest"
	"go.uber.org/zap/zaptest/observer"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/provider/fileprovider"
	"go.opentelemetry.io/collector/exporter/debugexporter"
	"go.opentelemetry.io/collector/otelcol"
	"go.opentelemetry.io/collector/service/telemetry/otelconftelemetry"
)

type Collector struct {
	collector *otelcol.Collector
	observer  *observer.ObservedLogs
}

// New creates and starts a new OTel collector for testing.
func New(tb testing.TB, configYAML string) *Collector {
	tb.Helper()

	configDir := tb.TempDir()
	configFile := filepath.Join(configDir, "otel.yaml")
	err := os.WriteFile(configFile, []byte(configYAML), 0o644)
	require.NoError(tb, err)

	if err != nil {
		tb.Fatalf("failed to create collector: %v", err)
	}

	var zapBuf zaptest.Buffer
	zapCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.Lock(zapcore.AddSync(&zapBuf)),
		zapcore.DebugLevel,
	)
	observed, observer := observer.New(zapcore.DebugLevel)
	core := zapcore.NewTee(zapCore, observed)

	settings := newCollectorSettings("file:"+configFile, core)
	col, err := otelcol.NewCollector(settings)
	require.NoError(tb, err)

	var wg sync.WaitGroup
	tb.Cleanup(func() {
		col.Shutdown()
		wg.Wait()

		if tb.Failed() {
			tb.Log("OTel Collector logs:\n" + zapBuf.String())
		}
	})

	wg.Add(1)
	go func() {
		defer wg.Done()
		ctx, cancel := signal.NotifyContext(tb.Context(), os.Interrupt)
		defer cancel()
		assert.NoError(tb, col.Run(ctx))
	}()

	require.Eventually(tb, func() bool {
		return col.GetState() == otelcol.StateRunning
	}, 10*time.Second, 10*time.Millisecond, "Collector did not start in time")

	return &Collector{collector: col, observer: observer}
}

func (c *Collector) ObservedLogs() *observer.ObservedLogs {
	return c.observer
}

func (c *Collector) Shutdown() {
	c.collector.Shutdown()
}

// MonitoringPort waits for a single Beat receiver HTTP monitoring server to log
// its listening address and returns the ephemeral port it bound to.
//
// Configure the receiver with `http.host: localhost` and `http.port: 0` so the
// OS assigns a free port at bind time. Reading the port back from the logs
// avoids the time-of-check/time-of-use race of pre-allocating a port.
func (c *Collector) MonitoringPort(tb testing.TB) int {
	tb.Helper()
	return c.MonitoringPorts(tb, 1)[0]
}

// MonitoringPorts waits until at least n Beat receiver HTTP monitoring servers
// have logged their listening addresses and returns the ephemeral ports they
// bound to. The ports are returned in the order they were logged; callers that
// only assert each endpoint works do not need a per-receiver mapping.
func (c *Collector) MonitoringPorts(tb testing.TB, n int) []int {
	tb.Helper()
	var ports []int
	require.EventuallyWithT(tb, func(ct *assert.CollectT) {
		seen := make(map[int]struct{})
		ports = ports[:0]
		for _, entry := range c.observer.FilterMessageSnippet(libbeattesting.MonitoringEndpointSnippet).All() {
			port, err := libbeattesting.ParseMonitoringPort(entry.Message)
			if !assert.NoError(ct, err) {
				continue
			}
			if _, ok := seen[port]; ok {
				continue
			}
			seen[port] = struct{}{}
			ports = append(ports, port)
		}
		assert.GreaterOrEqualf(ct, len(ports), n, "waiting for %d monitoring endpoints to start", n)
	}, 30*time.Second, 100*time.Millisecond, "collector monitoring endpoints did not start")
	return ports[:n]
}

func getComponent() (otelcol.Factories, error) {
	receivers, err := otelcol.MakeFactoryMap(
		abreceiver.NewFactory(),
		fbreceiver.NewFactory(),
		hbreceiver.NewFactory(),
		mbreceiver.NewFactory(),
		osqreceiver.NewFactory(),
		pbreceiver.NewFactory(),
	)
	if err != nil {
		return otelcol.Factories{}, nil //nolint:nilerr //ignoring this error
	}

	extensions, err := otelcol.MakeFactoryMap(
		beatsauthextension.NewFactory(),
		elasticsearchstorage.NewFactory(),
	)
	if err != nil {
		return otelcol.Factories{}, nil //nolint:nilerr //ignoring this error
	}

	processors, err := otelcol.MakeFactoryMap(
		beatprocessor.NewFactory(),
	)
	if err != nil {
		return otelcol.Factories{}, nil //nolint:nilerr //ignoring this error
	}

	exporters, err := otelcol.MakeFactoryMap(
		debugexporter.NewFactory(),
		elasticsearchexporter.NewFactory(),
		logstashexporter.NewFactory(),
		kafkaexporter.NewFactory(),
	)
	if err != nil {
		return otelcol.Factories{}, nil //nolint:nilerr //ignoring this error
	}

	return otelcol.Factories{
		Receivers:  receivers,
		Processors: processors,
		Exporters:  exporters,
		Extensions: extensions,
		Telemetry:  otelconftelemetry.NewFactory(),
	}, nil
}

func newCollectorSettings(filename string, core zapcore.Core) otelcol.CollectorSettings {
	return otelcol.CollectorSettings{
		BuildInfo: component.BuildInfo{
			Command:     "otel",
			Description: "Test OTel Collector",
			Version:     version.GetDefaultVersion(),
		},
		Factories: getComponent,
		LoggingOptions: []zap.Option{
			zap.WrapCore(func(c zapcore.Core) zapcore.Core {
				return core
			}),
		},
		ConfigProviderSettings: otelcol.ConfigProviderSettings{
			ResolverSettings: confmap.ResolverSettings{
				URIs: []string{filename},
				ProviderFactories: []confmap.ProviderFactory{
					fileprovider.NewFactory(),
				},
			},
		},
	}
}
