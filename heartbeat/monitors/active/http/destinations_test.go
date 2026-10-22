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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conf "github.com/elastic/elastic-agent-libs/config"
)

func TestDestinations(t *testing.T) {
	tests := []struct {
		name    string
		cfg     map[string]any
		want    []string
		wantErr string
	}{
		{name: "urls", cfg: map[string]any{"urls": []any{"https://api.example.com/v1", "http://[::1]:8080/"}}, want: []string{"api.example.com", "::1"}},
		{name: "hosts without scheme", cfg: map[string]any{"hosts": []any{"api.example.com:8080"}}, want: []string{"api.example.com"}},
		{name: "user info", cfg: map[string]any{"urls": []any{"https://user:pass@api.example.com"}}, want: []string{"api.example.com"}},
		{name: "proxy", cfg: map[string]any{"urls": []any{"https://api.example.com"}, "proxy_url": "http://proxy.example.com:3128"}, want: []string{"api.example.com", "proxy.example.com"}},
		{name: "disabled proxy", cfg: map[string]any{"urls": []any{"https://api.example.com"}, "proxy_url": "http://proxy.example.com:3128", "proxy_disable": true}, want: []string{"api.example.com"}},
		{name: "redirects", cfg: map[string]any{"urls": []any{"https://api.example.com"}, "max_redirects": 1}, wantErr: "max_redirects must be 0"},
		{name: "unsupported scheme", cfg: map[string]any{"urls": []any{"ftp://api.example.com"}}, wantErr: "cannot parse URL 0"},
		// References are text here. Whatever they resolve to, they can't
		// change a host parsed before them.
		{name: "reference in the query", cfg: map[string]any{"urls": []any{"https://api.example.com/?k=vault://prod@app#key"}}, want: []string{"api.example.com"}},
		{name: "reference in the path", cfg: map[string]any{"urls": []any{"https://api.example.com/vault://mine@app#path"}}, want: []string{"api.example.com"}},
		{name: "reference as the url", cfg: map[string]any{"urls": []any{"vault://mine@app#url"}}, want: []string{"vault"}},
		{name: "reference as the host", cfg: map[string]any{"urls": []any{"https://vault://mine@app#host/"}}, want: []string{"vault"}},
		{name: "reference as the port", cfg: map[string]any{"urls": []any{"https://api.example.com:vault://mine@app#port"}}, want: []string{"https"}},
		{name: "reference in user info", cfg: map[string]any{"urls": []any{"https://u:vault://mine@app#u@api.example.com"}}, want: []string{"https"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := destinations(conf.MustNewConfigFrom(tc.cfg))
			if tc.wantErr != "" {
				assert.ErrorContains(t, err, tc.wantErr, "destinations must fail")
				return
			}
			require.NoError(t, err, "destinations must not fail")
			assert.Equal(t, tc.want, got, "unexpected destinations")
		})
	}
}

func TestDestinationsAreTheHostsCreateUses(t *testing.T) {
	urls := []any{
		"https://api.example.com/v1",
		"api.example.com:8080",
		"http://[::1]:8080/",
		"https://u:p@api.example.com/",
		"https://api.example.com/?k=vault://prod@app#key",
	}
	cfg := conf.MustNewConfigFrom(map[string]any{"urls": urls})
	config := defaultConfig()
	require.NoError(t, cfg.Unpack(&config), "config must unpack")
	got, err := destinations(cfg)
	require.NoError(t, err, "destinations must not fail")
	require.Len(t, got, len(config.Hosts), "there must be one destination per URL")

	for i, u := range config.Hosts {
		// The jobs connect to the host splitHostnamePort returns.
		hostname, _, err := splitHostnamePort(u)
		require.NoError(t, err, "URL %d must parse as the jobs parse it", i)
		assert.Equal(t, hostname, got[i], "destination %d must be the host the monitor connects to", i)
	}
}
