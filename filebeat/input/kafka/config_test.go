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
	"strings"
	"sync"
	"testing"
	"time"

	metrics "github.com/rcrowley/go-metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

// TestNewSaramaConfigDefaults verifies that the default input config maps the
// consumer-group and network timeouts onto sarama's own defaults, so that
// existing configurations are unaffected by these options being added.
func TestNewSaramaConfigDefaults(t *testing.T) {
	saramaConfig, err := newSaramaConfig(defaultConfig(), logp.NewNopLogger())
	require.NoError(t, err, "default config should produce a valid sarama config")
	assert.Nil(t, monitoring.Default.Get("filebeat.inputs.kafka"),
		"config construction must not register Sarama metrics on monitoring.Default")
	assert.Nil(t, monitoring.Default.Get("kafka"),
		"config construction must not register Sarama metrics on monitoring.Default")

	assert.Equal(t, 10*time.Second, saramaConfig.Consumer.Group.Session.Timeout)
	assert.Equal(t, 3*time.Second, saramaConfig.Consumer.Group.Heartbeat.Interval)
	assert.Equal(t, 30*time.Second, saramaConfig.Net.DialTimeout)
	assert.Equal(t, 30*time.Second, saramaConfig.Net.ReadTimeout)
	assert.Equal(t, 30*time.Second, saramaConfig.Net.WriteTimeout)
	assert.Empty(t, saramaConfig.Consumer.Group.InstanceId,
		"group_instance_id must be unset by default so consumers keep dynamic membership")
}

// TestNewSaramaConfigTimeoutOverrides verifies that the session_timeout,
// heartbeat_interval and timeout options are propagated to sarama. These are
// the knobs cross-region (high-latency WAN) consumers need to avoid spurious
// rebalances and fetch read timeouts.
func TestNewSaramaConfigTimeoutOverrides(t *testing.T) {
	config := defaultConfig()
	config.SessionTimeout = 30 * time.Second
	config.HeartbeatInterval = 10 * time.Second
	config.Timeout = 60 * time.Second
	config.KeepAlive = 15 * time.Second

	saramaConfig, err := newSaramaConfig(config, logp.NewNopLogger())
	require.NoError(t, err)

	assert.Equal(t, 30*time.Second, saramaConfig.Consumer.Group.Session.Timeout)
	assert.Equal(t, 10*time.Second, saramaConfig.Consumer.Group.Heartbeat.Interval)
	assert.Equal(t, 60*time.Second, saramaConfig.Net.DialTimeout)
	assert.Equal(t, 60*time.Second, saramaConfig.Net.ReadTimeout)
	assert.Equal(t, 60*time.Second, saramaConfig.Net.WriteTimeout)
	assert.Equal(t, 15*time.Second, saramaConfig.Net.KeepAlive)
}

// TestNewSaramaConfigGroupInstanceID verifies that group_instance_id is
// propagated to sarama's Consumer.Group.InstanceId, enabling Kafka static
// group membership (KIP-345), when a compatible protocol version is set.
func TestNewSaramaConfigGroupInstanceID(t *testing.T) {
	config := defaultConfig()
	config.Version = "2.3.0"
	config.GroupInstanceID = "filebeat-pod-1"

	saramaConfig, err := newSaramaConfig(config, logp.NewNopLogger())
	require.NoError(t, err)

	assert.Equal(t, "filebeat-pod-1", saramaConfig.Consumer.Group.InstanceId,
		"group_instance_id must be propagated to sarama's Consumer.Group.InstanceId")
}

// TestNewSaramaConfigGroupInstanceIDRequiresVersion verifies that setting
// group_instance_id with a protocol version below 2.3.0 (including the 2.1.0
// default) fails early with a clear, Filebeat-oriented error rather than
// sarama's opaque "need Version >= 2.3" message.
func TestNewSaramaConfigGroupInstanceIDRequiresVersion(t *testing.T) {
	config := defaultConfig() // Version defaults to 2.1.0
	config.GroupInstanceID = "filebeat-pod-1"

	_, err := newSaramaConfig(config, logp.NewNopLogger())
	require.Error(t, err, "group_instance_id below version 2.3.0 must be rejected")
	assert.ErrorContains(t, err, "group_instance_id requires 'version' >= 2.3.0",
		"error must carry the stable, searchable message naming the option and required version")
}

// TestNewSaramaConfigGroupInstanceIDInvalid verifies that malformed
// group_instance_id values are rejected by sarama's own validation (length,
// reserved names, and the allowed character set) even when the version is
// compatible.
func TestNewSaramaConfigGroupInstanceIDInvalid(t *testing.T) {
	tests := map[string]string{
		"dot":            ".",
		"dot-dot":        "..",
		"illegal char":   "has space",
		"too long (250)": strings.Repeat("a", 250),
	}

	for name, id := range tests {
		t.Run(name, func(t *testing.T) {
			config := defaultConfig()
			config.Version = "2.3.0"
			config.GroupInstanceID = id

			_, err := newSaramaConfig(config, logp.NewNopLogger())
			assert.Error(t, err,
				"invalid group_instance_id %q must be rejected", id)
		})
	}
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
	for i := 0; i < n; i++ {
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
