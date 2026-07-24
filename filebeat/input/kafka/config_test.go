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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/common/kafka"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/sarama"
)

// TestNewSaramaConfigDefaults verifies that the default input config maps the
// consumer-group and network timeouts onto sarama's own defaults, so that
// existing configurations are unaffected by these options being added.
func TestNewSaramaConfigDefaults(t *testing.T) {
	saramaConfig, err := newSaramaConfig(defaultConfig(), logptest.NewTestingLogger(t, ""))
	require.NoError(t, err)

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

	saramaConfig, err := newSaramaConfig(config, logptest.NewTestingLogger(t, ""))
	require.NoError(t, err)

	assert.Equal(t, 30*time.Second, saramaConfig.Consumer.Group.Session.Timeout)
	assert.Equal(t, 10*time.Second, saramaConfig.Consumer.Group.Heartbeat.Interval)
	assert.Equal(t, 60*time.Second, saramaConfig.Net.DialTimeout)
	assert.Equal(t, 60*time.Second, saramaConfig.Net.ReadTimeout)
	assert.Equal(t, 60*time.Second, saramaConfig.Net.WriteTimeout)
	assert.Equal(t, 15*time.Second, saramaConfig.Net.KeepAlive)
}

func TestNewSaramaConfigOAUTHBEARER(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")

	baseConfig := func() kafkaInputConfig {
		cfg := defaultConfig()
		cfg.Hosts = []string{"localhost:9092"}
		cfg.Topics = []string{"foo"}
		cfg.GroupID = "filebeat"
		return cfg
	}

	t.Run("valid config enables SASL and sets a token provider", func(t *testing.T) {
		cfg := baseConfig()
		cfg.Sasl = kafka.SaslConfig{
			SaslMechanism:   "OAUTHBEARER",
			CredentialsPath: "/var/run/secrets/tokens/kafka.jwt",
			Extensions: map[string]string{
				"logicalCluster": "lkc-abc123",
				"identityPoolId": "pool-xyz789",
			},
		}

		sc, err := newSaramaConfig(cfg, logger)
		require.NoError(t, err)
		assert.True(t, sc.Net.SASL.Enable, "SASL should be enabled")
		assert.Equal(t, sarama.SASLMechanism(sarama.SASLTypeOAuth), sc.Net.SASL.Mechanism)
		assert.NotNil(t, sc.Net.SASL.TokenProvider, "token provider should be set for OAUTHBEARER")
	})

	t.Run("missing credentials_path is an error", func(t *testing.T) {
		cfg := baseConfig()
		cfg.Sasl = kafka.SaslConfig{SaslMechanism: "OAUTHBEARER"}

		_, err := newSaramaConfig(cfg, logger)
		require.Error(t, err, "expected an error when sasl.credentials_path is not set for OAUTHBEARER")
	})
}

// TestNewSaramaConfigGroupInstanceID verifies that group_instance_id is
// propagated to sarama's Consumer.Group.InstanceId, enabling Kafka static
// group membership (KIP-345), when a compatible protocol version is set.
func TestNewSaramaConfigGroupInstanceID(t *testing.T) {
	config := defaultConfig()
	config.Version = "2.3.0"
	config.GroupInstanceID = "filebeat-pod-1"

	saramaConfig, err := newSaramaConfig(config, logptest.NewTestingLogger(t, ""))
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

	_, err := newSaramaConfig(config, logptest.NewTestingLogger(t, ""))
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

			_, err := newSaramaConfig(config, logptest.NewTestingLogger(t, ""))
			assert.Error(t, err,
				"invalid group_instance_id %q must be rejected", id)
		})
	}
}
