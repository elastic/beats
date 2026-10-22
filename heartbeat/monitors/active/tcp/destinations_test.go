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

package tcp

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
		{name: "host and port", cfg: map[string]any{"hosts": []any{"api.example.com:443", "[::1]:22"}}, want: []string{"api.example.com", "::1"}},
		{name: "hosts and ports", cfg: map[string]any{"hosts": []any{"api.example.com"}, "ports": []any{80, 443}}, want: []string{"api.example.com"}},
		{name: "URL", cfg: map[string]any{"hosts": []any{"tls://api.example.com:443"}}, want: []string{"api.example.com"}},
		{name: "SOCKS5 proxy", cfg: map[string]any{"hosts": []any{"api.example.com:443"}, "proxy_url": "socks5://proxy.example.com:1080", "proxy_use_local_resolver": true}, want: []string{"api.example.com", "proxy.example.com"}},
		{name: "missing port", cfg: map[string]any{"hosts": []any{"api.example.com"}}, wantErr: "cannot parse host 0"},
		{name: "unsupported scheme", cfg: map[string]any{"hosts": []any{"http://api.example.com:80"}}, wantErr: "cannot parse host 0"},
		// References are text here. Whatever they resolve to, they can't
		// change a host parsed before them.
		{name: "reference as the host", cfg: map[string]any{"hosts": []any{"vault://mine@app#host"}, "ports": []any{443}}, wantErr: "cannot parse host 0"},
		{name: "reference after the host", cfg: map[string]any{"hosts": []any{"api.example.com:vault://mine@app#port"}, "ports": []any{443}}, want: []string{"api.example.com:vault://mine@app#port"}},
		{name: "reference in the path", cfg: map[string]any{"hosts": []any{"tcp://api.example.com:443/vault://mine@app#path"}}, want: []string{"api.example.com"}},
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
