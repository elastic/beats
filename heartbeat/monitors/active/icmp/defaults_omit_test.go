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

package icmp

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/heartbeat/monitors"
	conf "github.com/elastic/elastic-agent-libs/config"
)

// See TestOmittedDefaultsFallBackToHeartbeatDefaults in the http package for
// context: proves omitting/nulling fields that equal the agent default is safe
// (elastic/kibana#241818).
func TestOmittedDefaultsFallBackToHeartbeatDefaults(t *testing.T) {
	assertDefaults := func(t *testing.T, c Config) {
		t.Helper()
		require.Equal(t, 16*time.Second, c.Timeout, "timeout")
		require.Equal(t, 1*time.Second, c.Wait, "wait")
		require.Equal(t, true, c.Mode.IPv4, "ipv4")
		require.Equal(t, true, c.Mode.IPv6, "ipv6")
		require.Equal(t, monitors.PingAny, c.Mode.Mode, "mode")
	}

	t.Run("all default fields absent", func(t *testing.T) {
		cfg, err := conf.NewConfigFrom(map[string]interface{}{"hosts": "localhost"})
		require.NoError(t, err)

		c := DefaultConfig
		require.NoError(t, cfg.Unpack(&c))
		assertDefaults(t, c)
	})

	t.Run("all default fields explicitly null", func(t *testing.T) {
		cfg, err := conf.NewConfigFrom(map[string]interface{}{
			"hosts":   "localhost",
			"timeout": nil,
			"wait":    nil,
			"mode":    nil,
			"ipv4":    nil,
			"ipv6":    nil,
		})
		require.NoError(t, err)

		c := DefaultConfig
		require.NoError(t, cfg.Unpack(&c))
		assertDefaults(t, c)
	})

	t.Run("explicit non-default values override", func(t *testing.T) {
		cfg, err := conf.NewConfigFrom(map[string]interface{}{
			"hosts":   "localhost",
			"timeout": "30s",
			"wait":    "5s",
			"ipv6":    false,
		})
		require.NoError(t, err)

		c := DefaultConfig
		require.NoError(t, cfg.Unpack(&c))
		require.Equal(t, 30*time.Second, c.Timeout)
		require.Equal(t, 5*time.Second, c.Wait)
		require.Equal(t, false, c.Mode.IPv6)
	})
}
