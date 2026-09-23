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

//go:build integration && !requirefips

package http

import (
	"context"
	"net"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp/logptest"

	"github.com/elastic/beats/v7/libbeat/beat"
)

// TestHTTPMonitorKerberosHandshake checks password auth against the heartbeat_kerberos fixture.
func TestHTTPMonitorKerberosHandshake(t *testing.T) {
	const (
		target   = "http://localhost:8080/"
		confPath = "testdata/krb5.conf"
	)
	body, err := os.ReadFile(confPath)
	require.NoError(t, err, "reading krb5 config")
	requireReachable(t, target)

	// service_name is pinned to the fixture principal. DNS search domains can
	// otherwise rewrite localhost and ask the KDC for a different SPN.
	tests := []struct {
		name string
		key  string
		val  any
	}{
		{name: "config file", key: "config_path", val: confPath},
		{name: "inline krb5_conf", key: "krb5_conf", val: string(body)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := conf.NewConfigFrom(map[string]any{
				"hosts":   target,
				"timeout": "15s",
				"kerberos": map[string]any{
					"enabled":      true,
					"auth_type":    "password",
					"username":     "testuser",
					"password":     "testpass",
					"realm":        "EXAMPLE.COM",
					"service_name": "HTTP/localhost",
					tc.key:         tc.val,
				},
			})
			require.NoError(t, err)

			p, err := create("kerberos", cfg, beat.Info{Logger: logptest.NewTestingLogger(t, "")})
			require.NoError(t, err)
			require.Equal(t, 1, p.Endpoints)

			event := &beat.Event{}
			_, err = p.Jobs[0](event)
			require.NoError(t, err, "kerberos-authenticated ping should succeed")

			statusCode, err := event.GetValue("http.response.status_code")
			require.NoError(t, err, "event must carry the response status code")
			assert.Equal(t, 200, statusCode, "monitor should report the SPNEGO-authenticated response")
		})
	}
}

// requireReachable skips the test when the SPNEGO target cannot be dialed, so
// local runs without the docker fixture skip cleanly instead of failing.
func requireReachable(t *testing.T, rawURL string) {
	t.Helper()
	u, err := url.Parse(rawURL)
	require.NoError(t, err, "parsing target URL")
	host := u.Host
	if u.Port() == "" {
		host = net.JoinHostPort(u.Hostname(), "80")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", host)
	if err != nil {
		if os.Getenv("CI") != "" {
			t.Fatalf("SPNEGO target %q not reachable; heartbeat_kerberos should be up via docker-compose in CI: %v", rawURL, err)
		}
		t.Skipf("SPNEGO target %q not reachable; start the heartbeat_kerberos fixture first: %v", rawURL, err)
	}
	_ = conn.Close()
}
