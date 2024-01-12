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

package tlscommon

import (
	"fmt"
	"testing"

	"github.com/elastic/elastic-agent-libs/config"
	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"
)

func TestMarshallText(t *testing.T) {
	var verification TLSVerificationMode
	bytes, err := verification.MarshalText()
	require.NoError(t, err)
	require.NotNil(t, bytes)
	require.Equal(t, "full", string(bytes))

	verification = VerifyNone
	bytes, err = verification.MarshalText()
	require.NoError(t, err)
	require.NotNil(t, bytes)
	require.Equal(t, "none", string(bytes))
}

func TestLoadWithEmptyStringVerificationMode(t *testing.T) {
	cfg, err := load(`
    enabled: true
    certificate: mycert.pem
    key: mycert.key
    verification_mode: ""
    supported_protocols: [TLSv1.1, TLSv1.2]
    renegotiation: freely
  `)

	assert.NoError(t, err)
	assert.Equal(t, cfg.VerificationMode, VerifyFull)
}

func TestLoadWithEmptyVerificationMode(t *testing.T) {
	cfg, err := load(`
    enabled: true
    verification_mode:
    supported_protocols: [TLSv1.1, TLSv1.2]
    curve_types:
      - P-521
    renegotiation: freely
  `)

	assert.NoError(t, err)
	assert.Equal(t, cfg.VerificationMode, VerifyFull)
}

func TestTLSClientAuthUnpack(t *testing.T) {
	tests := []struct {
		val    string
		expect TLSClientAuth
		err    error
	}{{
		val:    "",
		expect: TLSClientAuthNone,
		err:    nil,
	}, {
		val:    "none",
		expect: TLSClientAuthNone,
		err:    nil,
	}, {
		val:    "optional",
		expect: TLSClientAuthOptional,
		err:    nil,
	}, {
		val:    "required",
		expect: TLSClientAuthRequired,
		err:    nil,
	}, {
		val: "invalid",
		err: fmt.Errorf("unknown client authentication mode 'invalid'"),
	}}
	for _, tc := range tests {
		t.Run(tc.val, func(t *testing.T) {
			var auth TLSClientAuth
			err := auth.Unpack(tc.val)
			assert.Equal(t, tc.expect, auth)
			if tc.err != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTLSClientAuthMarshalText(t *testing.T) {
	tests := []struct {
		name   string
		val    TLSClientAuth
		expect []byte
	}{{
		name:   "no value",
		expect: []byte("none"),
	}, {
		name:   "none",
		val:    TLSClientAuthNone,
		expect: []byte("none"),
	}, {
		name:   "optional",
		val:    TLSClientAuthOptional,
		expect: []byte("optional"),
	}, {
		name:   "required",
		val:    TLSClientAuthRequired,
		expect: []byte("required"),
	}}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p, err := tc.val.MarshalText()
			assert.Equal(t, tc.expect, p)
			assert.NoError(t, err)
		})
	}
}

func TestLoadTLSClientAuth(t *testing.T) {
	tests := []struct {
		name   string
		yaml   string
		expect *TLSClientAuth
	}{{
		name: "no client auth value",
		yaml: `
    certificate: mycert.pem
    key: mycert.key`,
		expect: nil,
	}, {
		name: "client auth empty",
		yaml: `
    certificate: mycert.pem
    key: mycert.key
    client_authentication: `,
		expect: nil,
	}, {
		name: "client auth none",
		yaml: `
    certificate: mycert.pem
    key: mycert.key
    client_authentication: none`,
		expect: &none,
	}, {
		name: "client auth optional",
		yaml: `
    certificate: mycert.pem
    key: mycert.key
    client_authentication: optional`,
		expect: &optional,
	}, {
		name: "client auth required",
		yaml: `
    certificate: mycert.pem
    key: mycert.key
    client_authentication: required`,
		expect: &required,
	}, {
		name: "certificate_authorities is not null, no client_authentication",
		yaml: `
    certificate: mycert.pem
    key: mycert.key
    certificate_authorities: [ca.crt]`,
		expect: &required, // NOTE Unpack will insert required if cas are present and no client_authentication is passed
	}, {
		name: "certificate_authorities is not null, client_authentication is none",
		yaml: `
    certificate: mycert.pem
    key: mycert.key
    client_authentication: none
    certificate_authorities: [ca.crt]`,
		expect: &none,
	}}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := mustLoadServerConfig(t, tc.yaml)
			if tc.expect == nil {
				assert.Nil(t, cfg.ClientAuth)
			} else {
				assert.Equal(t, *tc.expect, *cfg.ClientAuth)
			}
		})
	}

	t.Run("invalid", func(t *testing.T) {
		_, err := loadServerConfig(`client_authentication: invalid`)
		assert.Error(t, err)
	})
}

func loadServerConfig(yamlStr string) (*ServerConfig, error) {
	var cfg ServerConfig
	config, err := config.NewConfigWithYAML([]byte(yamlStr), "")
	if err != nil {
		return nil, err
	}

	if err := config.Unpack(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func mustLoadServerConfig(t *testing.T, yamlStr string) *ServerConfig {
	t.Helper()
	cfg, err := loadServerConfig(yamlStr)
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}
