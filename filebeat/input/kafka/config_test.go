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

package kafka

import (
	"sync"
	"testing"

	metrics "github.com/rcrowley/go-metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

func TestNewSaramaConfigDoesNotRegisterOnDefault(t *testing.T) {
	_, err := newSaramaConfig(defaultConfig(), logp.NewNopLogger())
	require.NoError(t, err, "default config should produce a valid sarama config")
	assert.Nil(t, monitoring.Default.Get("filebeat.inputs.kafka"),
		"config construction must not register Sarama metrics on monitoring.Default")
	assert.Nil(t, monitoring.Default.Get("kafka"),
		"config construction must not register Sarama metrics on monitoring.Default")
}

func TestAttachSaramaMetricsUsesParentRegistry(t *testing.T) {
	parent := monitoring.NewRegistry()
	cfg, err := newSaramaConfig(defaultConfig(), logp.NewNopLogger())
	require.NoError(t, err, "default config should produce a valid sarama config")

	attachSaramaMetrics(cfg, parent, logp.NewNopLogger())
	require.NotNil(t, cfg.MetricRegistry, "Sarama should have a metric registry after attach")

	metrics.GetOrRegisterMeter("incoming-byte-rate", cfg.MetricRegistry)
	assert.NotNil(t, parent.Get("kafka.bytes_read"),
		"incoming-byte-rate should be renamed to bytes_read on the provided parent")
	assert.Nil(t, monitoring.Default.Get("filebeat.inputs.kafka.bytes_read"),
		"must not register Kafka input metrics on the process-global registry")
	assert.Nil(t, monitoring.Default.Get("kafka.bytes_read"),
		"must not register Kafka input metrics on the process-global registry")
}

func TestAttachSaramaMetricsNilParentDoesNotUseDefault(t *testing.T) {
	cfg, err := newSaramaConfig(defaultConfig(), logp.NewNopLogger())
	require.NoError(t, err, "default config should produce a valid sarama config")

	attachSaramaMetrics(cfg, nil, logp.NewNopLogger())
	require.NotNil(t, cfg.MetricRegistry, "nil parent should still get a private registry")

	metrics.GetOrRegisterMeter("incoming-byte-rate", cfg.MetricRegistry)
	assert.Nil(t, monitoring.Default.Get("filebeat.inputs.kafka.bytes_read"),
		"nil parent must not fall back to monitoring.Default")
	assert.Nil(t, monitoring.Default.Get("kafka.bytes_read"),
		"nil parent must not fall back to monitoring.Default")
}

func TestAttachSaramaMetricsIsolatedPerParent(t *testing.T) {
	const n = 32
	var wg sync.WaitGroup
	wg.Add(n)
	for range n {
		go func() {
			defer wg.Done()
			parent := monitoring.NewRegistry()
			cfg, err := newSaramaConfig(defaultConfig(), logp.NewNopLogger())
			assert.NoError(t, err, "default config should produce a valid sarama config")
			if err != nil {
				return
			}
			attachSaramaMetrics(cfg, parent, logp.NewNopLogger())
			metrics.GetOrRegisterMeter("incoming-byte-rate", cfg.MetricRegistry)
			metrics.GetOrRegisterMeter("outgoing-byte-rate", cfg.MetricRegistry)
			assert.NotNil(t, parent.Get("kafka.bytes_read"),
				"each parent should own its own bytes_read metric")
			assert.NotNil(t, parent.Get("kafka.bytes_write"),
				"each parent should own its own bytes_write metric")
		}()
	}
	wg.Wait()
}
