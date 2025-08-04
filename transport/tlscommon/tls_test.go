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
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

func TestEmptyTlsConfig(t *testing.T) {
	cfg, err := load("")
	assert.NoError(t, err)

	assert.Equal(t, cfg, &Config{})
}

func TestLoadWithEmptyValues(t *testing.T) {
	cfg, err := load(`
    enabled:
    verification_mode:
    certificate:
    key:
    key_passphrase:
    key_passphrase_path:
    certificate_authorities:
    cipher_suites:
    curve_types:
    supported_protocols:
  `)

	assert.NoError(t, err)
	assert.Equal(t, cfg, &Config{})
}

func TestNoLoadNilConfig(t *testing.T) {
	cfg, err := LoadTLSConfig(nil, logptest.NewTestingLogger(t, ""))
	assert.NoError(t, err)
	assert.Nil(t, cfg)
}

func TestNoLoadDisabledConfig(t *testing.T) {
	enabled := false
	cfg, err := LoadTLSConfig(&Config{Enabled: &enabled}, logptest.NewTestingLogger(t, ""))
	assert.NoError(t, err)
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
      - ECDHE-ECDSA-AES-128-GCM-SHA256
      - ECDHE-ECDSA-AES-256-GCM-SHA384
    supported_protocols: [TLSv1.3]
    curve_types:
      - P-384
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
		[]TLSVersion{TLSVersion13},
		cfg.Versions)
	assert.Len(t, cfg.CurveTypes, 1)
	assert.Equal(t,
		tls.RenegotiateFreelyAsClient,
		tls.RenegotiationSupport(cfg.Renegotiation))
}

func TestApplyEmptyConfig(t *testing.T) {
	tmp, err := LoadTLSConfig(&Config{}, logptest.NewTestingLogger(t, ""))
	if err != nil {
		t.Fatal(err)
	}

	cfg := tmp.BuildModuleClientConfig("")
	assert.Equal(t, int(TLSVersionDefaultMin), int(cfg.MinVersion))
	assert.Equal(t, int(TLSVersionDefaultMax), int(cfg.MaxVersion))
	assert.Len(t, cfg.Certificates, 0)
	assert.Nil(t, cfg.RootCAs)
	assert.Equal(t, true, cfg.InsecureSkipVerify)
	assert.Len(t, cfg.CipherSuites, 0)
	assert.Len(t, cfg.CurvePreferences, 0)
	assert.Equal(t, tls.RenegotiateNever, cfg.Renegotiation)
}

func TestApplyWithConfig(t *testing.T) {
	tmp, err := LoadTLSConfig(mustLoad(t, `
    certificate: testdata/ca_test.pem
    key: testdata/ca_test.key
    certificate_authorities: [testdata/ca_test.pem]
    verification_mode: none
    cipher_suites:
      - "ECDHE-ECDSA-AES-128-GCM-SHA256"
      - "ECDHE-ECDSA-AES-256-GCM-SHA384"
    curve_types: [P-384]
    renegotiation: once
  `), logptest.NewTestingLogger(t, ""))
	if err != nil {
		t.Fatal(err)
	}

	cfg := tmp.BuildModuleClientConfig("")
	assert.NotNil(t, cfg)
	assert.Len(t, cfg.Certificates, 1)
	assert.NotNil(t, cfg.RootCAs)
	assert.Equal(t, true, cfg.InsecureSkipVerify)
	assert.Len(t, cfg.CipherSuites, 2)
	assert.Equal(t, int(TLSVersionDefaultMin), int(cfg.MinVersion))
	assert.Equal(t, int(TLSVersionDefaultMax), int(cfg.MaxVersion))
	assert.Len(t, cfg.CurvePreferences, 1)
	assert.Equal(t, tls.RenegotiateOnceAsClient, cfg.Renegotiation)
}

func TestServerConfigDefaults(t *testing.T) {
	t.Run("when CA is not explicitly set", func(t *testing.T) {
		var c ServerConfig
		config := config.MustNewConfigFrom(`
certificate: mycert.pem
key: mykey.pem
`)
		err := config.Unpack(&c)
		require.NoError(t, err)
		c.Certificate = CertificateConfig{} // prevent reading non-existent files
		tmp, err := LoadTLSServerConfig(&c, logptest.NewTestingLogger(t, ""))
		require.NoError(t, err)

		cfg := tmp.BuildModuleClientConfig("")

		assert.NotNil(t, cfg)
		// values not set by default
		assert.Len(t, cfg.Certificates, 0)
		assert.Nil(t, cfg.ClientCAs)
		assert.Len(t, cfg.CipherSuites, 0)
		assert.Len(t, cfg.CurvePreferences, 0)
		// values set by default
		assert.Equal(t, true, cfg.InsecureSkipVerify)
		assert.Equal(t, int(TLSVersionDefaultMin), int(cfg.MinVersion))
		assert.Equal(t, int(TLSVersionDefaultMax), int(cfg.MaxVersion))
		assert.Equal(t, tls.NoClientCert, cfg.ClientAuth)
	})
	t.Run("when CA is explicitly set", func(t *testing.T) {

		yamlStr := `
    certificate_authorities: [testdata/ca_test.pem]
    certificate: mycert.pem
    key: mykey.pem
`
		var c ServerConfig
		config, err := config.NewConfigWithYAML([]byte(yamlStr), "")
		require.NoError(t, err)
		err = config.Unpack(&c)
		require.NoError(t, err)
		c.Certificate = CertificateConfig{} // prevent reading non-existent files
		require.NoError(t, err)
		tmp, err := LoadTLSServerConfig(&c, logptest.NewTestingLogger(t, ""))
		require.NoError(t, err)

		cfg := tmp.BuildModuleClientConfig("")

		assert.NotNil(t, cfg)
		// values not set by default
		assert.Len(t, cfg.Certificates, 0)
		assert.NotNil(t, cfg.ClientCAs)
		assert.Len(t, cfg.CipherSuites, 0)
		assert.Len(t, cfg.CurvePreferences, 0)
		// values set by default
		assert.Equal(t, true, cfg.InsecureSkipVerify)
		assert.Equal(t, int(TLSVersionDefaultMin), int(cfg.MinVersion))
		assert.Equal(t, int(TLSVersionDefaultMax), int(cfg.MaxVersion))
		assert.Equal(t, tls.RequireAndVerifyClientCert, cfg.ClientAuth)
	})
}

func TestApplyWithServerConfig(t *testing.T) {
	yamlStr := `
    certificate: testdata/ca_test.pem
    key: testdata/ca_test.key
    certificate_authorities: [testdata/ca_test.pem]
    verification_mode: none
    client_authentication: optional
    cipher_suites:
      - "ECDHE-ECDSA-AES-128-GCM-SHA256"
      - "ECDHE-ECDSA-AES-256-GCM-SHA384"
    curve_types: [P-384]
  `
	var c ServerConfig
	config, err := config.NewConfigWithYAML([]byte(yamlStr), "")
	for i, ver := range TLSDefaultVersions {
		err := config.SetString("supported_protocols", i, ver.String())
		require.NoError(t, err)
	}

	if !assert.NoError(t, err) {
		return
	}

	err = config.Unpack(&c)
	if !assert.NoError(t, err) {
		return
	}
	tmp, err := LoadTLSServerConfig(&c, logptest.NewTestingLogger(t, ""))
	if !assert.NoError(t, err) {
		return
	}

	cfg := tmp.BuildModuleClientConfig("")
	assert.NotNil(t, cfg)
	assert.Len(t, cfg.Certificates, 1)
	assert.NotNil(t, cfg.ClientCAs)
	assert.Equal(t, true, cfg.InsecureSkipVerify)
	assert.Len(t, cfg.CipherSuites, 2)
	assert.Equal(t, int(TLSVersionDefaultMin), int(cfg.MinVersion))
	assert.Equal(t, int(TLSVersionDefaultMax), int(cfg.MaxVersion))
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
			config, err := config.NewConfigWithYAML([]byte(test.yaml), "")
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
			_, err = LoadTLSConfig(&tlscfg, logptest.NewTestingLogger(t, ""))
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

func TestPEMString(t *testing.T) {
	t.Run("is PEM formatted String", func(t *testing.T) {
		_, cert := makeKeyCertPair(t, blockTypePKCS1, "")
		assert.True(t, IsPEMString(cert))
	})

	t.Run("is not a PEM formatted String", func(t *testing.T) {
		c := "/tmp/certificate"
		assert.False(t, IsPEMString(c))
	})

	t.Run("is an empty string", func(t *testing.T) {
		c := ""
		assert.False(t, IsPEMString(c))
	})
}

func TestCertificateAuthorities(t *testing.T) {
	t.Run("From configuration", func(t *testing.T) {
		_, cert := makeKeyCertPair(t, blockTypePKCS1, "")
		cfg, err := load(`enabled: true`)
		require.NoError(t, err)
		cfg.CAs = []string{cert}

		tlsC, err := LoadTLSConfig(cfg, logptest.NewTestingLogger(t, ""))
		assert.NoError(t, err)
		assert.NotNil(t, tlsC)
	})

	t.Run("From disk", func(t *testing.T) {
		// Create a dummy configuration and append the CA after.
		_, cert := makeKeyCertPair(t, blockTypePKCS1, "")
		certFile := writeTestFile(t, cert)
		cfg, err := load(`enabled: true`)
		require.NoError(t, err)
		cfg.CAs = []string{certFile}

		tlsC, err := LoadTLSConfig(cfg, logptest.NewTestingLogger(t, ""))
		assert.NoError(t, err)
		assert.NotNil(t, tlsC)
	})

	t.Run("mixed from disk and embed", func(t *testing.T) {
		// Create a dummy configuration and append the CA after.
		_, cert := makeKeyCertPair(t, blockTypePKCS1, "")
		certFile := writeTestFile(t, cert)
		cfg, err := load(`enabled: true`)
		require.NoError(t, err)
		cfg.CAs = []string{certFile, cert}

		tlsC, err := LoadTLSConfig(cfg, logptest.NewTestingLogger(t, ""))
		assert.NoError(t, err)

		assert.NotNil(t, tlsC)
	})

}

// TestFIPSCertifacteAndKeys tests encrypted private keys
func TestCertificateAndKeys(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")
	t.Run("embed PKCS#1 key", func(t *testing.T) {
		// Create a dummy configuration and append the CA after.
		key, cert := makeKeyCertPair(t, blockTypePKCS1, "")
		cfg, err := load(`enabled: true`)
		require.NoError(t, err)
		cfg.Certificate.Certificate = cert
		cfg.Certificate.Key = key

		tlsC, err := LoadTLSConfig(cfg, logger)
		require.NoError(t, err)
		assert.NotNil(t, tlsC)
	})

	t.Run("embed PKCS#8 key", func(t *testing.T) {
		// Create a dummy configuration and append the CA after.
		key, cert := makeKeyCertPair(t, blockTypePKCS8, "")
		cfg, err := load(`enabled: true`)
		require.NoError(t, err)
		cfg.Certificate.Certificate = cert
		cfg.Certificate.Key = key

		tlsC, err := LoadTLSConfig(cfg, logger)
		require.NoError(t, err)
		assert.NotNil(t, tlsC)
	})

	t.Run("from disk PKCS#1 key", func(t *testing.T) {
		// Create a dummy configuration and append the CA after.
		key, cert := makeKeyCertPair(t, blockTypePKCS1, "")
		cfg, err := load(`enabled: true`)
		require.NoError(t, err)
		cfg.Certificate.Certificate = writeTestFile(t, cert)
		cfg.Certificate.Key = writeTestFile(t, key)

		tlsC, err := LoadTLSConfig(cfg, logger)
		require.NoError(t, err)
		assert.NotNil(t, tlsC)
	})
}

func TestKeyPassphrase(t *testing.T) {

	t.Run("unencrypted key file with passphrase", func(t *testing.T) {
		cfg, err := LoadTLSConfig(mustLoad(t, `
    enabled: true
    certificate: testdata/ca.crt
    key: testdata/ca.key
    key_passphrase: Abcd1234!
    `), logptest.NewTestingLogger(t, ""))
		require.NoError(t, err)
		assert.Equal(t, 1, len(cfg.Certificates), "expected 1 certificate to be loaded")
	})
}
