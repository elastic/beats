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

	"github.com/elastic/beats/v7/libbeat/tests/integration"
	"github.com/elastic/beats/v7/libbeat/version"
	"github.com/elastic/beats/v7/x-pack/filebeat/fbreceiver"
	"github.com/elastic/beats/v7/x-pack/metricbeat/mbreceiver"
	"github.com/elastic/beats/v7/x-pack/otel/exporter/logstashexporter"
	"github.com/elastic/beats/v7/x-pack/otel/extension/beatsauthextension"
	"github.com/elastic/beats/v7/x-pack/otel/processor/beatprocessor"

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
	collector    *otelcol.Collector
	observer     *observer.ObservedLogs
	done         chan struct{}
	shutdownOnce sync.Once
}

// New creates and starts a new OTel collector for testing.
//
// The collector's own telemetry metrics are disabled (see metricsOffConfig) so
// multiple collectors can run in parallel, e.g. under script/stresstest.sh,
// without colliding on the fixed Prometheus port.
func New(tb testing.TB, configYAML string) *Collector {
	tb.Helper()

	configDir := tb.TempDir()
	configFile := filepath.Join(configDir, "otel.yaml")
<<<<<<< HEAD
	err := os.WriteFile(configFile, []byte(configYAML), 0644)
	require.NoError(tb, err)
=======
	require.NoError(tb, os.WriteFile(configFile, []byte(configYAML), 0o644))
>>>>>>> 6497c632a (libbeat/testing: use ephemeral ports to avoid TOCTOU collisions (#51617))

	// Merged after the test config to disable the collector's own telemetry
	// metrics; kept in a separate file so we don't have to parse/rewrite the
	// test's YAML.
	metricsOffFile := filepath.Join(configDir, "metrics-off.yaml")
	require.NoError(tb, os.WriteFile(metricsOffFile, []byte(metricsOffConfig), 0o644))

	var zapBuf zaptest.Buffer
	zapCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.Lock(zapcore.AddSync(&zapBuf)),
		zapcore.DebugLevel,
	)
	observed, observer := observer.New(zapcore.DebugLevel)
	core := zapcore.NewTee(zapCore, observed)

	settings := newCollectorSettings([]string{"file:" + configFile, "file:" + metricsOffFile}, core)
	col, err := otelcol.NewCollector(settings)
	require.NoError(tb, err)

	c := &Collector{collector: col, observer: observer, done: make(chan struct{})}

	tb.Cleanup(func() {
		c.Shutdown()

		if tb.Failed() {
			tb.Log("OTel Collector logs:\n" + zapBuf.String())
		}
	})

<<<<<<< HEAD
	wg.Add(1)
	go func() {
		defer wg.Done()
=======
	go func() {
		defer close(c.done)
>>>>>>> 6497c632a (libbeat/testing: use ephemeral ports to avoid TOCTOU collisions (#51617))
		ctx, cancel := signal.NotifyContext(tb.Context(), os.Interrupt)
		defer cancel()
		assert.NoError(tb, col.Run(ctx))
	}()

	require.Eventually(tb, func() bool {
		return col.GetState() == otelcol.StateRunning
	}, 10*time.Second, 10*time.Millisecond, "Collector did not start in time")

	return c
}

func (c *Collector) ObservedLogs() *observer.ObservedLogs {
	return c.observer
}

// Shutdown stops the collector and blocks until it has fully exited. Blocking
// lets callers restart a collector that reuses the same path.home (or other
// resources) without racing the previous instance's teardown. It is safe to
// call multiple times.
func (c *Collector) Shutdown() {
	c.shutdownOnce.Do(c.collector.Shutdown)
	<-c.done
}

// metricsOffConfig is merged on top of the test config to turn off the
// collector's own telemetry metrics. The collector otherwise starts a
// Prometheus reader on a fixed localhost:8888, which collides when collectors
// run in parallel (e.g. under stresstest.sh). Tests assert on Elasticsearch
// documents and the per-receiver HTTP monitoring endpoint rather than the
// collector's own metrics, so disabling them is safe.
const metricsOffConfig = `service:
  telemetry:
    metrics:
      level: none
`

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
		for _, entry := range c.observer.FilterMessageSnippet(integration.MonitoringEndpointSnippet).All() {
			port, err := integration.ParseMonitoringPort(entry.Message)
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

// SocketListeningPort waits for a tcp or udp input running inside the collector
// to log its listening address and returns the ephemeral port it bound to.
//
// Configure the input with host: <ip>:0 so the OS assigns a free port at bind
// time. Reading the port back from the logs avoids the time-of-check/time-of-use
// race of pre-allocating a port.
func (c *Collector) SocketListeningPort(tb testing.TB) int {
	tb.Helper()
	var port int
	require.EventuallyWithT(tb, func(ct *assert.CollectT) {
		for _, entry := range c.observer.FilterMessageSnippet(integration.SocketListeningSnippet).All() {
			p, err := integration.ParseSocketListeningPort(entry.Message)
			if !assert.NoError(ct, err) {
				continue
			}
			port = p
			return
		}
		assert.Fail(ct, "input listening address not logged yet")
	}, 30*time.Second, 100*time.Millisecond, "collector input did not start listening")
	return port
}

func getComponent() (otelcol.Factories, error) {
	receivers, err := otelcol.MakeFactoryMap(
		fbreceiver.NewFactory(),
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

// newCollectorSettings builds the collector settings from the given config
// URIs, which are resolved and deep-merged in order (later URIs win).
func newCollectorSettings(uris []string, core zapcore.Core) otelcol.CollectorSettings {
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
				URIs: uris,
				ProviderFactories: []confmap.ProviderFactory{
					fileprovider.NewFactory(),
				},
			},
		},
	}
}
