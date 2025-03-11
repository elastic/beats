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
	"crypto/tls"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/go-ucfg"

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
    supported_protocols: [TLSv1.2, TLSv1.3]
    renegotiation: freely
  `)

	assert.NoError(t, err)
	assert.Equal(t, cfg.VerificationMode, VerifyFull)
}

func TestLoadUnsupportedProtocols(t *testing.T) {
	cfg, err := load(`
    enabled: true
    certificate: mycert.pem
    key: mycert.key
    verification_mode: ""
    supported_protocols: [TLSv1.0, TLSv1.2]
    renegotiation: freely
  `)

	assert.ErrorContains(t, err, "unsupported tls version")
	assert.Nil(t, cfg)
}

func TestLoadWithEmptyVerificationMode(t *testing.T) {
	cfg, err := load(`
    enabled: true
    verification_mode:
    supported_protocols: [TLSv1.2, TLSv1.3]
    curve_types:
      - P-384
    renegotiation: freely
  `)

	assert.NoError(t, err)
	assert.Equal(t, cfg.VerificationMode, VerifyFull)
}

func TestRepackConfig(t *testing.T) {
	cfg, err := load(`
    enabled: true
    verification_mode: certificate
    supported_protocols: [TLSv1.2, TLSv1.3]
    cipher_suites:
      - ECDHE-ECDSA-AES-256-GCM-SHA384
    certificate_authorities:
      - /path/to/ca.crt
    certificate: /path/to/cert.crt
    key: /path/to/key.crt
    curve_types:
      - P-384
    renegotiation: freely
    ca_sha256:
      - example
    ca_trusted_fingerprint: fingerprint
  `)

	assert.NoError(t, err)
	assert.Equal(t, cfg.VerificationMode, VerifyCertificate)

	tmp, err := ucfg.NewFrom(cfg)
	assert.NoError(t, err)

	err = tmp.Unpack(cfg)
	assert.NoError(t, err)
	assert.Equal(t, cfg.VerificationMode, VerifyCertificate)
}

func TestRepackConfigFromJSON(t *testing.T) {
	cfg, err := loadJSON(`{
    "enabled": true,
    "verification_mode": "certificate",
    "supported_protocols": ["TLSv1.2", "TLSv1.3"],
    "cipher_suites": ["ECDHE-ECDSA-AES-256-GCM-SHA384"],
    "certificate_authorities": ["/path/to/ca.crt"],
    "certificate": "/path/to/cert.crt",
    "key": "/path/to/key.crt",
    "curve_types": "P-384",
    "renegotiation": "freely",
    "ca_sha256": ["example"],
    "ca_trusted_fingerprint": "fingerprint"
  }`)

	assert.NoError(t, err)
	assert.Equal(t, cfg.VerificationMode, VerifyCertificate)

	tmp, err := ucfg.NewFrom(cfg)
	assert.NoError(t, err)

	err = tmp.Unpack(cfg)
	assert.NoError(t, err)
	assert.Equal(t, cfg.VerificationMode, VerifyCertificate)
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

func Test_TLSVerificaionMode_Unpack(t *testing.T) {
	tests := []struct {
		name   string
		hasErr bool
		in     interface{}
		exp    TLSVerificationMode
	}{{
		name:   "nil",
		hasErr: false,
		in:     nil,
		exp:    VerifyFull,
	}, {
		name:   "empty string",
		hasErr: false,
		in:     "",
		exp:    VerifyFull,
	}, {
		name:   "unknown string",
		hasErr: true,
		in:     "unknown",
	}, {
		name:   "string",
		hasErr: false,
		in:     "strict",
		exp:    VerifyStrict,
	}, {
		name:   "int64",
		hasErr: false,
		in:     int64(1),
		exp:    VerifyNone,
	}, {
		name:   "uint64",
		hasErr: false,
		in:     uint64(1),
		exp:    VerifyNone,
	}, {
		name:   "unknown type",
		hasErr: true,
		in:     uint8(1),
	}}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			v := new(TLSVerificationMode)
			err := v.Unpack(tc.in)
			if tc.hasErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.exp, *v)
			}
		})
	}
}

func Test_TLSClientAuth_Unpack(t *testing.T) {
	tests := []struct {
		name   string
		hasErr bool
		in     interface{}
		exp    TLSClientAuth
	}{{
		name:   "nil",
		hasErr: false,
		in:     nil,
		exp:    TLSClientAuthNone,
	}, {
		name:   "empty string",
		hasErr: false,
		in:     "",
		exp:    TLSClientAuthNone,
	}, {
		name:   "unknown string",
		hasErr: true,
		in:     "unknown",
	}, {
		name:   "string",
		hasErr: false,
		in:     "optional",
		exp:    TLSClientAuthOptional,
	}, {
		name:   "int64",
		hasErr: false,
		in:     int64(3),
		exp:    TLSClientAuthOptional,
	}, {
		name:   "uint64",
		hasErr: false,
		in:     uint64(3),
		exp:    TLSClientAuthOptional,
	}, {
		name:   "unknown type",
		hasErr: true,
		in:     uint8(1),
	}}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			v := new(TLSClientAuth)
			err := v.Unpack(tc.in)
			if tc.hasErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.exp, *v)
			}
		})
	}
}

func Test_CipherSuite_Unpack(t *testing.T) {
	tests := []struct {
		name   string
		hasErr bool
		in     interface{}
		exp    CipherSuite
	}{{
		name:   "unknown string",
		hasErr: true,
		in:     "unknown",
	}, {
		name:   "string",
		hasErr: false,
		in:     "RSA-AES-128-CBC-SHA",
		exp:    CipherSuite(tls.TLS_RSA_WITH_AES_128_CBC_SHA),
	}, {
		name:   "int64",
		hasErr: false,
		in:     int64(47),
		exp:    CipherSuite(tls.TLS_RSA_WITH_AES_128_CBC_SHA),
	}, {
		name:   "uint64",
		hasErr: false,
		in:     uint64(47),
		exp:    CipherSuite(tls.TLS_RSA_WITH_AES_128_CBC_SHA),
	}, {
		name:   "unknown type",
		hasErr: true,
		in:     uint8(1),
	}}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			v := new(CipherSuite)
			err := v.Unpack(tc.in)
			if tc.hasErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.exp, *v)
			}
		})
	}
}

func Test_tlsCurveType_Unpack(t *testing.T) {
	tests := []struct {
		name   string
		hasErr bool
		in     interface{}
		exp    tlsCurveType
	}{{
		name:   "unknown string",
		hasErr: true,
		in:     "unknown",
	}, {
		name:   "string",
		hasErr: false,
		in:     "P-256",
		exp:    tlsCurveType(tls.CurveP256),
	}, {
		name:   "int64",
		hasErr: false,
		in:     int64(23),
		exp:    tlsCurveType(tls.CurveP256),
	}, {
		name:   "uint64",
		hasErr: false,
		in:     uint64(23),
		exp:    tlsCurveType(tls.CurveP256),
	}, {
		name:   "unknown type",
		hasErr: true,
		in:     uint8(1),
	}}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			v := new(tlsCurveType)
			err := v.Unpack(tc.in)
			if tc.hasErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.exp, *v)
			}
		})
	}
}

func Test_TLSRenegotiationSupport_Unpack(t *testing.T) {
	tests := []struct {
		name   string
		hasErr bool
		in     interface{}
		exp    TLSRenegotiationSupport
	}{{
		name:   "unknown string",
		hasErr: true,
		in:     "unknown",
	}, {
		name:   "string",
		hasErr: false,
		in:     "never",
		exp:    TLSRenegotiationSupport(tls.RenegotiateNever),
	}, {
		name:   "int64",
		hasErr: false,
		in:     int64(0),
		exp:    TLSRenegotiationSupport(tls.RenegotiateNever),
	}, {
		name:   "uint64",
		hasErr: false,
		in:     uint64(0),
		exp:    TLSRenegotiationSupport(tls.RenegotiateNever),
	}, {
		name:   "unknown type",
		hasErr: true,
		in:     uint8(1),
	}}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			v := new(TLSRenegotiationSupport)
			err := v.Unpack(tc.in)
			if tc.hasErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.exp, *v)
			}
		})
	}
}
