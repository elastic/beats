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

package http

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/heartbeat/monitors"
	conf "github.com/elastic/elastic-agent-libs/config"
)

// TestOmittedDefaultsFallBackToHeartbeatDefaults proves that when Kibana stops
// emitting fields that already equal the agent default, Heartbeat still applies
// the exact same values. It exercises both scenarios that can reach the agent:
//   - the field is fully absent (new guarded integration template), and
//   - the field is present but null (old un-guarded template rendering an
//     omitted Kibana value as `key: null`).
//
// This is the safety proof for elastic/kibana issue #241818.
func TestOmittedDefaultsFallBackToHeartbeatDefaults(t *testing.T) {
	assertDefaults := func(t *testing.T, c Config) {
		t.Helper()
		require.Equal(t, "GET", c.Check.Request.Method, "check.request.method")
		require.Equal(t, 0, c.MaxRedirects, "max_redirects")
		require.Equal(t, "on_error", c.Response.IncludeBody, "response.include_body")
		require.Equal(t, true, c.Response.IncludeHeaders, "response.include_headers")
		require.Equal(t, true, c.Mode.IPv4, "ipv4")
		require.Equal(t, true, c.Mode.IPv6, "ipv6")
		require.Equal(t, monitors.PingAny, c.Mode.Mode, "mode")
		require.Equal(t, 16*time.Second, c.Transport.Timeout, "timeout")
	}

	t.Run("all default fields absent", func(t *testing.T) {
		cfg, err := conf.NewConfigFrom(map[string]interface{}{
			"urls": "http://localhost:8080",
		})
		require.NoError(t, err)

		c := defaultConfig()
		require.NoError(t, cfg.Unpack(&c))
		assertDefaults(t, c)
	})

	t.Run("all default fields explicitly null", func(t *testing.T) {
		cfg, err := conf.NewConfigFrom(map[string]interface{}{
			"urls":                     "http://localhost:8080",
			"timeout":                  nil,
			"max_redirects":            nil,
			"response.include_headers": nil,
			"response.include_body":    nil,
			"check.request.method":     nil,
			"mode":                     nil,
			"ipv4":                     nil,
			"ipv6":                     nil,
		})
		require.NoError(t, err)

		c := defaultConfig()
		require.NoError(t, cfg.Unpack(&c))
		assertDefaults(t, c)
	})

	// Control: confirms the dotted keys actually bind (path separator works),
	// so the assertions above are meaningful and not passing by accident.
	t.Run("explicit non-default values override", func(t *testing.T) {
		cfg, err := conf.NewConfigFrom(map[string]interface{}{
			"urls":                     "http://localhost:8080",
			"max_redirects":            5,
			"response.include_headers": false,
			"response.include_body":    "always",
			"check.request.method":     "POST",
			"ipv4":                     false,
		})
		require.NoError(t, err)

		c := defaultConfig()
		require.NoError(t, cfg.Unpack(&c))
		require.Equal(t, "POST", c.Check.Request.Method)
		require.Equal(t, 5, c.MaxRedirects)
		require.Equal(t, "always", c.Response.IncludeBody)
		require.Equal(t, false, c.Response.IncludeHeaders)
		require.Equal(t, false, c.Mode.IPv4)
	})
}
