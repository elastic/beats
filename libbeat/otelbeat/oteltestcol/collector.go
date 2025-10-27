// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package oteltestcol

import (
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/elastic/beats/v7/libbeat/version"
	"github.com/elastic/beats/v7/x-pack/filebeat/fbreceiver"
	"github.com/elastic/beats/v7/x-pack/metricbeat/mbreceiver"
	"github.com/elastic/beats/v7/x-pack/otel/exporter/logstashexporter"
	"github.com/elastic/beats/v7/x-pack/otel/processor/beatprocessor"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/opentelemetry-collector-components/extension/beatsauthextension"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/elasticsearchexporter"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/provider/fileprovider"
	"go.opentelemetry.io/collector/exporter/debugexporter"
	"go.opentelemetry.io/collector/otelcol"
)

type Collector struct {
	collector *otelcol.Collector
	logger    *logp.Logger
	observer  *observer.ObservedLogs
}

// New creates and starts a new OTel collector for testing.
func New(tb testing.TB, configYAML string) *Collector {
	tb.Helper()

	configDir := tb.TempDir()
	configFile := filepath.Join(configDir, "otel.yaml")
	err := os.WriteFile(configFile, []byte(configYAML), 0644)
	require.NoError(tb, err)

	if err != nil {
		tb.Fatalf("failed to create collector: %v", err)
	}

	// TODO(mauri870): this logger is too verbose, change it so it does not log everything to stderr.
	logger, observer := logptest.NewTestingLoggerWithObserver(tb, "")
	settings := newCollectorSettings("file:"+configFile, logger)
	col, err := otelcol.NewCollector(settings)
	require.NoError(tb, err)

	var wg sync.WaitGroup
	tb.Cleanup(func() {
		col.Shutdown()
		wg.Wait()
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

	return &Collector{collector: col, logger: logger, observer: observer}
}

func (c *Collector) ObservedLogs() *observer.ObservedLogs {
	return c.observer
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
	)
	if err != nil {
		return otelcol.Factories{}, nil //nolint:nilerr //ignoring this error
	}

	return otelcol.Factories{
		Receivers:  receivers,
		Processors: processors,
		Exporters:  exporters,
		Extensions: extensions,
	}, nil

}

func newCollectorSettings(filename string, logger *logp.Logger) otelcol.CollectorSettings {
	return otelcol.CollectorSettings{
		BuildInfo: component.BuildInfo{
			Command:     "otel",
			Description: "Test OTel Collector",
			Version:     version.GetDefaultVersion(),
		},
		Factories: getComponent,
		LoggingOptions: []zap.Option{
			zap.WrapCore(func(c zapcore.Core) zapcore.Core {
				return logger.Core()
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
