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

	"github.com/elastic/beats/v7/libbeat/common/transport/kerberos"
)

// kerberosTestConfig returns a non-nil Kerberos config so the block is treated
// as "enabled". These tests never contact a KDC, so the auth_type/keytab fields
// (normally validated during config unpacking) are irrelevant here.
func kerberosTestConfig() *kerberos.Config {
	return &kerberos.Config{
		Realm:      "CORP.LOCAL",
		ConfigPath: "/etc/krb5.conf",
		Username:   "svc",
	}
}

func TestConfigAuthMutualExclusivity(t *testing.T) {
	enabled := true
	tests := map[string]struct {
		cfg     Config
		wantErr bool
	}{
		"basic only": {
			cfg: Config{Hosts: []string{"http://x"}, Username: "u", Password: "p"},
		},
		"kerberos only": {
			cfg: Config{Hosts: []string{"http://x"}, Kerberos: kerberosTestConfig()},
		},
		"ntlm only": {
			cfg: Config{Hosts: []string{"http://x"}, NTLM: &NTLMConfig{Enabled: &enabled, Username: `D\u`, Password: "p"}},
		},
		"basic + ntlm": {
			cfg:     Config{Hosts: []string{"http://x"}, Username: "u", Password: "p", NTLM: &NTLMConfig{Enabled: &enabled, Username: `D\u`, Password: "p"}},
			wantErr: true,
		},
		"kerberos + ntlm": {
			cfg:     Config{Hosts: []string{"http://x"}, Kerberos: kerberosTestConfig(), NTLM: &NTLMConfig{Enabled: &enabled, Username: `D\u`, Password: "p"}},
			wantErr: true,
		},
		"no auth": {
			cfg: Config{Hosts: []string{"http://x"}},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			err := tc.cfg.Validate()
			if tc.wantErr {
				require.Error(t, err, "expected a mutual-exclusivity error")
				assert.Contains(t, err.Error(), "only one authentication method", "error should explain the auth conflict")
			} else {
				require.NoError(t, err, "config should be valid")
			}
		})
	}
}

func TestNTLMConfigValidate(t *testing.T) {
	enabled := true
	disabled := false

	require.NoError(t, (&NTLMConfig{Enabled: &disabled}).Validate(), "disabled ntlm needs no credentials")
	require.NoError(t, (*NTLMConfig)(nil).Validate(), "nil ntlm config must be safe to validate")

	err := (&NTLMConfig{Enabled: &enabled, Password: "p"}).Validate()
	require.Error(t, err, "missing username must fail")
	assert.Contains(t, err.Error(), "username")

	err = (&NTLMConfig{Enabled: &enabled, Username: "u"}).Validate()
	require.Error(t, err, "missing password must fail")
	assert.Contains(t, err.Error(), "password")
}

func TestNTLMAuthUsername(t *testing.T) {
	tests := map[string]struct {
		cfg  NTLMConfig
		want string
	}{
		"separate domain":       {cfg: NTLMConfig{Username: "user", Domain: "CORP"}, want: `CORP\user`},
		"domain in username":    {cfg: NTLMConfig{Username: `CORP\user`, Domain: "IGNORED"}, want: `CORP\user`},
		"upn in username":       {cfg: NTLMConfig{Username: "user@corp.local", Domain: "IGNORED"}, want: "user@corp.local"},
		"no domain at all":      {cfg: NTLMConfig{Username: "user"}, want: "user"},
		"empty domain provided": {cfg: NTLMConfig{Username: "user", Domain: ""}, want: "user"},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.cfg.authUsername(), "domain/username should be combined for the negotiator")
		})
	}
}

func TestBuildRequestAuthSchemes(t *testing.T) {
	enabled := true

	t.Run("basic sets Authorization Basic and Close", func(t *testing.T) {
		cfg := defaultConfig()
		cfg.Username = "user"
		cfg.Password = "pass"
		req, err := buildRequest("http://example.com", &cfg, nil)
		require.NoError(t, err)
		user, pass, ok := req.BasicAuth()
		require.True(t, ok, "basic auth header must be present")
		assert.Equal(t, "user", user)
		assert.Equal(t, "pass", pass)
		assert.True(t, req.Close, "basic auth keeps the historical close-per-request behaviour")
	})

	t.Run("ntlm sets domain\\user creds and disables Close", func(t *testing.T) {
		cfg := defaultConfig()
		cfg.NTLM = &NTLMConfig{Enabled: &enabled, Username: "user", Password: "pass", Domain: "CORP"}
		req, err := buildRequest("http://example.com", &cfg, nil)
		require.NoError(t, err)
		user, pass, ok := req.BasicAuth()
		require.True(t, ok, "ntlm credentials are carried in the Basic auth header for the negotiator")
		assert.Equal(t, `CORP\user`, user, "domain must be prepended")
		assert.Equal(t, "pass", pass)
		assert.False(t, req.Close, "ntlm must keep the connection alive across the handshake")
	})

	t.Run("kerberos sets no Authorization header", func(t *testing.T) {
		cfg := defaultConfig()
		cfg.Kerberos = kerberosTestConfig()
		req, err := buildRequest("http://example.com", &cfg, nil)
		require.NoError(t, err)
		_, _, ok := req.BasicAuth()
		assert.False(t, ok, "kerberos must not set a Basic auth header; SPNEGO sets it during the handshake")
		assert.Empty(t, req.Header.Get("Authorization"), "no Authorization header before the handshake")
	})
}
