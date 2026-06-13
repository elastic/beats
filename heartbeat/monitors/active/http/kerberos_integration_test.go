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

	"github.com/elastic/beats/v7/libbeat/beat"
)

// TestHTTPMonitorKerberosHandshake drives the HTTP monitor against a real
// SPNEGO/Negotiate-protected endpoint backed by an MIT KDC. Both the KDC and
// the SPNEGO server are provided by the heartbeat_kerberos docker fixture
// (see heartbeat/docker-compose.yml); the client authenticates with password
// auth using the committed testdata/krb5.conf, so no keytab is needed host-side.
//
// Run locally with:
//
//	cd testing/environments/docker/heartbeat_kerberos && docker build -t hb-kdc:latest .
//	docker run -d --name hb-kdc -p 1088:88 -p 1088:88/udp -p 18080:8080 hb-kdc:latest
//	HB_KRB5_TARGET=http://localhost:18080/ go test -tags integration \
//	    -run TestHTTPMonitorKerberosHandshake ./heartbeat/monitors/active/http/...
func TestHTTPMonitorKerberosHandshake(t *testing.T) {
	krb5Conf := envOr("HB_KRB5_CONF", "testdata/krb5.conf")
	target := envOr("HB_KRB5_TARGET", "http://localhost:8080/")
	realm := envOr("HB_KRB5_REALM", "EXAMPLE.COM")
	user := envOr("HB_KRB5_USER", "testuser")
	pass := envOr("HB_KRB5_PASS", "testpass")

	if _, err := os.Stat(krb5Conf); err != nil {
		t.Skipf("krb5 config %q not found; start the heartbeat_kerberos fixture first: %v", krb5Conf, err)
	}
	requireReachable(t, target)

	cfgSrc := map[string]interface{}{
		"hosts":   target,
		"timeout": "15s",
		"kerberos": map[string]interface{}{
			"enabled":     true,
			"auth_type":   "password",
			"username":    user,
			"password":    pass,
			"realm":       realm,
			"config_path": krb5Conf,
		},
	}
	cfg, err := conf.NewConfigFrom(cfgSrc)
	require.NoError(t, err)

	p, err := create("kerberos", cfg)
	require.NoError(t, err)
	require.Equal(t, 1, p.Endpoints)

	event := &beat.Event{}
	_, err = p.Jobs[0](event)
	require.NoError(t, err, "kerberos-authenticated ping should succeed")

	statusCode, err := event.GetValue("http.response.status_code")
	require.NoError(t, err, "event must carry the response status code")
	assert.Equal(t, 200, statusCode, "monitor should report the SPNEGO-authenticated response")
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
		t.Skipf("SPNEGO target %q not reachable; start the heartbeat_kerberos fixture first: %v", rawURL, err)
	}
	_ = conn.Close()
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
