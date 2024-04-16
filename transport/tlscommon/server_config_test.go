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
	"testing"

	"github.com/elastic/go-ucfg"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

// variables so we can use pointers in tests
var (
	required = TLSClientAuthRequired
	optional = TLSClientAuthOptional
	none     = TLSClientAuthNone
)

func Test_ServerConfig_Serialization_ClientAuth(t *testing.T) {
	tests := []struct {
		name       string
		cfg        ServerConfig
		clientAuth *TLSClientAuth
	}{{
		name: "with ca",
		cfg: ServerConfig{
			Certificate: CertificateConfig{
				Certificate: "/path/to/cert.crt",
				Key:         "/path/to/cert.key",
			},
			CAs: []string{"/path/to/ca.crt"},
		},
		clientAuth: &required,
	}, {
		name: "no ca",
		cfg: ServerConfig{
			Certificate: CertificateConfig{
				Certificate: "/path/to/cert.crt",
				Key:         "/path/to/cert.key",
			},
		},
		clientAuth: nil,
	}, {
		name: "with ca and client auth none",
		cfg: ServerConfig{
			Certificate: CertificateConfig{
				Certificate: "/path/to/cert.crt",
				Key:         "/path/to/cert.key",
			},
			CAs:        []string{"/path/to/ca.crt"},
			ClientAuth: &none,
		},
		clientAuth: &none,
	}, {
		name: "no ca and client auth none",
		cfg: ServerConfig{
			Certificate: CertificateConfig{
				Certificate: "/path/to/cert.crt",
				Key:         "/path/to/cert.key",
			},
			ClientAuth: &none,
		},
		clientAuth: &none,
	}}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			p, err := yaml.Marshal(&tc.cfg)
			require.NoError(t, err)
			t.Logf("YAML Config:\n%s", string(p))
			scfg := mustLoadServerConfig(t, string(p))
			if tc.clientAuth == nil {
				require.Nil(t, scfg.ClientAuth)
			} else {
				require.Equal(t, *tc.clientAuth, *scfg.ClientAuth)
			}
		})
	}
}

func Test_ServerConfig_Repack(t *testing.T) {
	tests := []struct {
		name string
		yaml string
		auth *TLSClientAuth
	}{{
		name: "with client auth",
		yaml: `
    enabled: true
    verification_mode: certificate
    supported_protocols: [TLSv1.1, TLSv1.2]
    cipher_suites:
      - RSA-AES-256-CBC-SHA
    certificate_authorities:
      - /path/to/ca.crt
    certificate: /path/to/cert.cry
    key: /path/to/key/crt
    curve_types:
      - P-521
    client_authentication: optional
    ca_sha256:
      - example`,
		auth: &optional,
	}, {
		name: "nil client auth",
		yaml: `
    enabled: true
    verification_mode: certificate
    supported_protocols: [TLSv1.1, TLSv1.2]
    cipher_suites:
      - RSA-AES-256-CBC-SHA
    certificate_authorities:
      - /path/to/ca.crt
    certificate: /path/to/cert.cry
    key: /path/to/key/crt
    curve_types:
      - P-521
    ca_sha256:
      - example`,
		auth: &required,
	}, {
		name: "nil client auth, no cas",
		yaml: `
    enabled: true
    verification_mode: certificate
    supported_protocols: [TLSv1.1, TLSv1.2]
    cipher_suites:
      - RSA-AES-256-CBC-SHA
    certificate: /path/to/cert.cry
    key: /path/to/key/crt
    curve_types:
      - P-521
    ca_sha256:
      - example`,
		auth: nil,
	}}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := mustLoadServerConfig(t, tc.yaml)
			if tc.auth != nil {
				require.Equal(t, *tc.auth, *cfg.ClientAuth)
			} else {
				require.Nil(t, cfg.ClientAuth)
			}

			tmp, err := ucfg.NewFrom(cfg)
			require.NoError(t, err)

			err = tmp.Unpack(&cfg)
			require.NoError(t, err)
			if tc.auth != nil {
				require.Equal(t, *tc.auth, *cfg.ClientAuth)
			} else {
				require.Nil(t, cfg.ClientAuth)
			}
		})
	}
}

func Test_ServerConfig_RepackJSON(t *testing.T) {
	tests := []struct {
		name string
		json string
		auth *TLSClientAuth
	}{{
		name: "with client auth",
		json: `{
    "enabled": true,
    "verification_mode": "certificate",
    "supported_protocols": ["TLSv1.1", "TLSv1.2"],
    "cipher_suites": ["RSA-AES-256-CBC-SHA"],
    "certificate_authorities": ["/path/to/ca.crt"],
    "certificate": "/path/to/cert.crt",
    "key": "/path/to/key.crt",
    "curve_types": "P-521",
    "renegotiation": "freely",
    "ca_sha256": ["example"],
    "ca_trusted_fingerprint": "fingerprint",
    "client_authentication": "optional"
    }`,
		auth: &optional,
	}, {
		name: "nil client auth",
		json: `{
    "enabled": true,
    "verification_mode": "certificate",
    "supported_protocols": ["TLSv1.1", "TLSv1.2"],
    "cipher_suites": ["RSA-AES-256-CBC-SHA"],
    "certificate_authorities": ["/path/to/ca.crt"],
    "certificate": "/path/to/cert.crt",
    "key": "/path/to/key.crt",
    "curve_types": "P-521",
    "renegotiation": "freely",
    "ca_sha256": ["example"],
    "ca_trusted_fingerprint": "fingerprint"
    }`,
		auth: &required,
	}, {
		name: "nil client auth, no cas",
		json: `{
    "enabled": true,
    "verification_mode": "certificate",
    "supported_protocols": ["TLSv1.1", "TLSv1.2"],
    "cipher_suites": ["RSA-AES-256-CBC-SHA"],
    "certificate": "/path/to/cert.crt",
    "key": "/path/to/key.crt",
    "curve_types": "P-521",
    "renegotiation": "freely",
    "ca_sha256": ["example"]
    }`,
		auth: nil,
	}}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := mustLoadServerConfigJSON(t, tc.json)
			if tc.auth != nil {
				require.Equal(t, *tc.auth, *cfg.ClientAuth)
			} else {
				require.Nil(t, cfg.ClientAuth)
			}

			tmp, err := ucfg.NewFrom(cfg)
			require.NoError(t, err)

			err = tmp.Unpack(&cfg)
			require.NoError(t, err)
			if tc.auth != nil {
				require.Equal(t, *tc.auth, *cfg.ClientAuth)
			} else {
				require.Nil(t, cfg.ClientAuth)
			}
		})
	}
}
