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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/logp"
)

// TestNewSaramaConfigDefaults verifies that the default input config maps the
// consumer-group and network timeouts onto sarama's own defaults, so that
// existing configurations are unaffected by these options being added.
func TestNewSaramaConfigDefaults(t *testing.T) {
	saramaConfig, err := newSaramaConfig(defaultConfig(), logp.NewNopLogger())
	require.NoError(t, err)

	assert.Equal(t, 10*time.Second, saramaConfig.Consumer.Group.Session.Timeout)
	assert.Equal(t, 3*time.Second, saramaConfig.Consumer.Group.Heartbeat.Interval)
	assert.Equal(t, 30*time.Second, saramaConfig.Net.DialTimeout)
	assert.Equal(t, 30*time.Second, saramaConfig.Net.ReadTimeout)
	assert.Equal(t, 30*time.Second, saramaConfig.Net.WriteTimeout)
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
