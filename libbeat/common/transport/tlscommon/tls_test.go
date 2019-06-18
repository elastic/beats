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

// +build !integration

package tlscommon

import (
	"crypto/tls"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/libbeat/common"
)

// test TLS config loading

func load(yamlStr string) (*Config, error) {
	var cfg Config
	config, err := common.NewConfigWithYAML([]byte(yamlStr), "")
	if err != nil {
		return nil, err
	}

	if err = config.Unpack(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func mustLoad(t *testing.T, yamlStr string) *Config {
	cfg, err := load(yamlStr)
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

func TestEmptyTlsConfig(t *testing.T) {
	cfg, err := load("")
	assert.Nil(t, err)

	assert.Equal(t, cfg, &Config{})
}

func TestLoadWithEmptyValues(t *testing.T) {
	cfg, err := load(`
    enabled:
    verification_mode:
    certificate:
    key:
    key_passphrase:
    certificate_authorities:
    cipher_suites:
    curve_types:
    supported_protocols:
  `)

	assert.Nil(t, err)
	assert.Equal(t, cfg, &Config{})
}

func TestNoLoadNilConfig(t *testing.T) {
	cfg, err := LoadTLSConfig(nil)
	assert.Nil(t, err)
	assert.Nil(t, cfg)
}

func TestNoLoadDisabledConfig(t *testing.T) {
	enabled := false
	cfg, err := LoadTLSConfig(&Config{Enabled: &enabled})
	assert.Nil(t, err)
	assert.Nil(t, cfg)
}

func TestValuesSet(t *testing.T) {
	cfg, err := load(`
    enabled: true
    certificate_authorities: ["ca1.pem", "ca2.pem"]
    certificate: mycert.pem
    key: mycert.key
    verification_mode: none
    cipher_suites:
      - ECDHE-ECDSA-AES-256-CBC-SHA
      - ECDHE-ECDSA-AES-256-GCM-SHA384
    supported_protocols: [TLSv1.1, TLSv1.2]
    curve_types:
      - P-521
    renegotiation: freely
  `)

	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "mycert.pem", cfg.Certificate.Certificate)
	assert.Equal(t, "mycert.key", cfg.Certificate.Key)
	assert.Len(t, cfg.CAs, 2)
	assert.Equal(t, VerifyNone, cfg.VerificationMode)
	assert.Len(t, cfg.CipherSuites, 2)
	assert.Equal(t,
		[]TLSVersion{TLSVersion11, TLSVersion12},
		cfg.Versions)
	assert.Len(t, cfg.CurveTypes, 1)
	assert.Equal(t,
		tls.RenegotiateFreelyAsClient,
		tls.RenegotiationSupport(cfg.Renegotiation))
}

func TestApplyEmptyConfig(t *testing.T) {
	tmp, err := LoadTLSConfig(&Config{})
	if err != nil {
		t.Fatal(err)
	}

	cfg := tmp.BuildModuleConfig("")
	assert.Equal(t, int(tls.VersionTLS10), int(cfg.MinVersion))
	assert.Equal(t, int(tls.VersionTLS12), int(cfg.MaxVersion))
	assert.Len(t, cfg.Certificates, 0)
	assert.Nil(t, cfg.RootCAs)
	assert.Equal(t, false, cfg.InsecureSkipVerify)
	assert.Len(t, cfg.CipherSuites, 0)
	assert.Len(t, cfg.CurvePreferences, 0)
}

func TestApplyWithConfig(t *testing.T) {
	tmp, err := LoadTLSConfig(mustLoad(t, `
    certificate: ca_test.pem
    key: ca_test.key
    certificate_authorities: [ca_test.pem]
    verification_mode: none
    cipher_suites:
      - "ECDHE-ECDSA-AES-256-CBC-SHA"
      - "ECDHE-ECDSA-AES-256-GCM-SHA384"
    curve_types: [P-384]
  `))
	if err != nil {
		t.Fatal(err)
	}

	cfg := tmp.BuildModuleConfig("")
	assert.NotNil(t, cfg)
	assert.Len(t, cfg.Certificates, 1)
	assert.NotNil(t, cfg.RootCAs)
	assert.Equal(t, true, cfg.InsecureSkipVerify)
	assert.Len(t, cfg.CipherSuites, 2)
	assert.Equal(t, int(tls.VersionTLS10), int(cfg.MinVersion))
	assert.Equal(t, int(tls.VersionTLS12), int(cfg.MaxVersion))
	assert.Len(t, cfg.CurvePreferences, 1)
}

func TestServerConfigDefaults(t *testing.T) {
	t.Run("when CA is not explicitly set", func(t *testing.T) {
		var c ServerConfig
		config := common.MustNewConfigFrom([]byte(``))
		err := config.Unpack(&c)
		require.NoError(t, err)
		tmp, err := LoadTLSServerConfig(&c)
		require.NoError(t, err)

		cfg := tmp.BuildModuleConfig("")

		assert.NotNil(t, cfg)
		// values not set by default
		assert.Len(t, cfg.Certificates, 0)
		assert.Nil(t, cfg.ClientCAs)
		assert.Len(t, cfg.CipherSuites, 0)
		assert.Len(t, cfg.CurvePreferences, 0)
		// values set by default
		assert.Equal(t, false, cfg.InsecureSkipVerify)
		assert.Equal(t, int(tls.VersionTLS10), int(cfg.MinVersion))
		assert.Equal(t, int(tls.VersionTLS12), int(cfg.MaxVersion))
		assert.Equal(t, tls.NoClientCert, cfg.ClientAuth)
	})
	t.Run("when CA is explicitly set", func(t *testing.T) {

		yamlStr := `
    certificate_authorities: [ca_test.pem]
`
		var c ServerConfig
		config, err := common.NewConfigWithYAML([]byte(yamlStr), "")
		err = config.Unpack(&c)
		require.NoError(t, err)
		tmp, err := LoadTLSServerConfig(&c)
		require.NoError(t, err)

		cfg := tmp.BuildModuleConfig("")

		assert.NotNil(t, cfg)
		// values not set by default
		assert.Len(t, cfg.Certificates, 0)
		assert.NotNil(t, cfg.ClientCAs)
		assert.Len(t, cfg.CipherSuites, 0)
		assert.Len(t, cfg.CurvePreferences, 0)
		// values set by default
		assert.Equal(t, false, cfg.InsecureSkipVerify)
		assert.Equal(t, int(tls.VersionTLS10), int(cfg.MinVersion))
		assert.Equal(t, int(tls.VersionTLS12), int(cfg.MaxVersion))
		assert.Equal(t, tls.RequireAndVerifyClientCert, cfg.ClientAuth)
	})
}

func TestApplyWithServerConfig(t *testing.T) {
	yamlStr := `
    certificate: ca_test.pem
    key: ca_test.key
    certificate_authorities: [ca_test.pem]
    verification_mode: none
    client_authentication: optional
    supported_protocols: [TLSv1.1, TLSv1.2]
    cipher_suites:
      - "ECDHE-ECDSA-AES-256-CBC-SHA"
      - "ECDHE-ECDSA-AES-256-GCM-SHA384"
    curve_types: [P-384]
  `
	var c ServerConfig
	config, err := common.NewConfigWithYAML([]byte(yamlStr), "")
	if !assert.NoError(t, err) {
		return
	}

	err = config.Unpack(&c)
	if !assert.NoError(t, err) {
		return
	}
	tmp, err := LoadTLSServerConfig(&c)
	if !assert.NoError(t, err) {
		return
	}

	cfg := tmp.BuildModuleConfig("")
	assert.NotNil(t, cfg)
	assert.Len(t, cfg.Certificates, 1)
	assert.NotNil(t, cfg.ClientCAs)
	assert.Equal(t, true, cfg.InsecureSkipVerify)
	assert.Len(t, cfg.CipherSuites, 2)
	assert.Equal(t, int(tls.VersionTLS11), int(cfg.MinVersion))
	assert.Equal(t, int(tls.VersionTLS12), int(cfg.MaxVersion))
	assert.Len(t, cfg.CurvePreferences, 1)
	assert.Equal(t, tls.VerifyClientCertIfGiven, cfg.ClientAuth)
}

func TestCertificateFails(t *testing.T) {
	tests := []struct {
		title string
		yaml  string
	}{
		{
			"certificate without key",
			"certificate: mycert.pem",
		},
		{
			"key without certificate",
			"key: mycert.key",
		},
		{
			"unknown cipher suite",
			"cipher_suites: ['unknown cipher suite']",
		},
		{
			"unknown version",
			"supported_protocols: [UnknownTLSv1.1]",
		},
		{
			"unknown curve type",
			"curve_types: ['unknown curve type']",
		},
		{
			"unknown renegotiation type",
			"renegotiation: always",
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("run test (%v): %v", i, test.title), func(t *testing.T) {
			config, err := common.NewConfigWithYAML([]byte(test.yaml), "")
			if err != nil {
				t.Error(err)
				return
			}

			// one must fail: validators on Unpack or transformation to *tls.Config
			var tlscfg Config
			if err = config.Unpack(&tlscfg); err != nil {
				t.Log(err)
				return
			}
			_, err = LoadTLSConfig(&tlscfg)
			t.Log(err)
			assert.Error(t, err)
		})
	}
}

func TestResolveTLSVersion(t *testing.T) {
	v := ResolveTLSVersion(tls.VersionTLS11)
	assert.Equal(t, "TLSv1.1", v)
}

func TestResolveCipherSuite(t *testing.T) {
	c := ResolveCipherSuite(tls.TLS_RSA_WITH_AES_128_CBC_SHA)
	assert.Equal(t, "RSA-AES-128-CBC-SHA", c)
}
