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

// NTLM is unavailable in FIPS builds, so this end-to-end handshake test (which
// constructs the NTLM transport via create) only runs in non-FIPS builds.
//go:build !requirefips

package http

import (
	"bytes"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conf "github.com/elastic/elastic-agent-libs/config"

	"github.com/elastic/beats/v7/libbeat/beat"
)

// TestHTTPMonitorNTLMHandshake drives the full monitor against a server that
// emulates the NTLM challenge. It proves the Negotiator is wired into the
// transport: the server must observe a well-formed NTLM negotiate token and the
// monitor must report the final authenticated response.
func TestHTTPMonitorNTLMHandshake(t *testing.T) {
	var mu sync.Mutex
	var negotiateToken []byte
	sawAnonymous := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authz := r.Header.Get("Authorization")
		switch {
		case authz == "":
			// Anonymous attempt: ask the client to start NTLM.
			mu.Lock()
			sawAnonymous = true
			mu.Unlock()
			w.Header().Set("WWW-Authenticate", "NTLM")
			w.WriteHeader(http.StatusUnauthorized)
		case strings.HasPrefix(authz, "NTLM "):
			// The negotiate message arrived; capture it and accept. Returning a
			// non-401 here ends the handshake per RFC 4559, which is enough to
			// prove the negotiator engaged end-to-end.
			tok, _ := base64.StdEncoding.DecodeString(strings.TrimPrefix(authz, "NTLM "))
			mu.Lock()
			negotiateToken = tok
			mu.Unlock()
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("authenticated"))
		default:
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	defer server.Close()

	cfgSrc := map[string]interface{}{
		"hosts":   server.URL,
		"timeout": "5s",
		"ntlm": map[string]interface{}{
			"enabled":  true,
			"username": "user",
			"password": "pass",
			"domain":   "CORP",
		},
	}
	cfg, err := conf.NewConfigFrom(cfgSrc)
	require.NoError(t, err)

	p, err := create("ntlm", cfg)
	require.NoError(t, err)
	require.Equal(t, 1, p.Endpoints)

	event := &beat.Event{}
	_, err = p.Jobs[0](event)
	require.NoError(t, err, "ntlm-authenticated ping should succeed")

	statusCode, err := event.GetValue("http.response.status_code")
	require.NoError(t, err, "event must carry the response status code")
	assert.Equal(t, 200, statusCode, "monitor should report the final authenticated response")

	mu.Lock()
	defer mu.Unlock()
	assert.True(t, sawAnonymous, "negotiator should first probe anonymously")
	require.NotEmpty(t, negotiateToken, "server must receive an NTLM negotiate token")
	assert.True(t, bytes.HasPrefix(negotiateToken, []byte("NTLMSSP\x00")), "token must be a valid NTLMSSP message, got %q", negotiateToken)
}
