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

//go:build !integration
// +build !integration

package tlscommon

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v8/libbeat/common"
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
    certificate_authorities:
    cipher_suites:
    curve_types:
    supported_protocols:
  `)

	assert.NoError(t, err)
	assert.Equal(t, cfg, &Config{})
}

func TestNoLoadNilConfig(t *testing.T) {
	cfg, err := LoadTLSConfig(nil)
	assert.NoError(t, err)
	assert.Nil(t, cfg)
}

func TestNoLoadDisabledConfig(t *testing.T) {
	enabled := false
	cfg, err := LoadTLSConfig(&Config{Enabled: &enabled})
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
    certificate: ca_test.pem
    key: ca_test.key
    certificate_authorities: [ca_test.pem]
    verification_mode: none
    cipher_suites:
      - "ECDHE-ECDSA-AES-256-CBC-SHA"
      - "ECDHE-ECDSA-AES-256-GCM-SHA384"
    curve_types: [P-384]
    renegotiation: once
  `))
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
		config := common.MustNewConfigFrom(`
certificate: mycert.pem
key: mykey.pem
`)
		err := config.Unpack(&c)
		require.NoError(t, err)
		c.Certificate = CertificateConfig{} // prevent reading non-existent files
		tmp, err := LoadTLSServerConfig(&c)
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
    certificate_authorities: [ca_test.pem]
    certificate: mycert.pem
    key: mykey.pem
`
		var c ServerConfig
		config, err := common.NewConfigWithYAML([]byte(yamlStr), "")
		err = config.Unpack(&c)
		c.Certificate = CertificateConfig{} // prevent reading non-existent files
		require.NoError(t, err)
		tmp, err := LoadTLSServerConfig(&c)
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
    certificate: ca_test.pem
    key: ca_test.key
    certificate_authorities: [ca_test.pem]
    verification_mode: none
    client_authentication: optional
    cipher_suites:
      - "ECDHE-ECDSA-AES-256-CBC-SHA"
      - "ECDHE-ECDSA-AES-256-GCM-SHA384"
    curve_types: [P-384]
  `
	var c ServerConfig
	config, err := common.NewConfigWithYAML([]byte(yamlStr), "")
	for i, ver := range TLSDefaultVersions {
		config.SetString("supported_protocols", i, ver.String())
	}

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

func TestPEMString(t *testing.T) {
	t.Run("is PEM formatted String", func(t *testing.T) {
		c := `-----BEGIN CERTIFICATE-----
MIIDCjCCAfKgAwIBAgITJ706Mu2wJlKckpIvkWxEHvEyijANBgkqhkiG9w0BAQsF
ADAUMRIwEAYDVQQDDAlsb2NhbGhvc3QwIBcNMTkwNzIyMTkyOTA0WhgPMjExOTA2
MjgxOTI5MDRaMBQxEjAQBgNVBAMMCWxvY2FsaG9zdDCCASIwDQYJKoZIhvcNAQEB
BQADggEPADCCAQoCggEBANce58Y/JykI58iyOXpxGfw0/gMvF0hUQAcUrSMxEO6n
fZRA49b4OV4SwWmA3395uL2eB2NB8y8qdQ9muXUdPBWE4l9rMZ6gmfu90N5B5uEl
94NcfBfYOKi1fJQ9i7WKhTjlRkMCgBkWPkUokvBZFRt8RtF7zI77BSEorHGQCk9t
/D7BS0GJyfVEhftbWcFEAG3VRcoMhF7kUzYwp+qESoriFRYLeDWv68ZOvG7eoWnP
PsvZStEVEimjvK5NSESEQa9xWyJOmlOKXhkdymtcUd/nXnx6UTCFgnkgzSdTWV41
CI6B6aJ9svCTI2QuoIq2HxX/ix7OvW1huVmcyHVxyUECAwEAAaNTMFEwHQYDVR0O
BBYEFPwN1OceFGm9v6ux8G+DZ3TUDYxqMB8GA1UdIwQYMBaAFPwN1OceFGm9v6ux
8G+DZ3TUDYxqMA8GA1UdEwEB/wQFMAMBAf8wDQYJKoZIhvcNAQELBQADggEBAG5D
874A4YI7YUwOVsVAdbWtgp1d0zKcPRR+r2OdSbTAV5/gcS3jgBJ3i1BN34JuDVFw
3DeJSYT3nxy2Y56lLnxDeF8CUTUtVQx3CuGkRg1ouGAHpO/6OqOhwLLorEmxi7tA
H2O8mtT0poX5AnOAhzVy7QW0D/k4WaoLyckM5hUa6RtvgvLxOwA0U+VGurCDoctu
8F4QOgTAWyh8EZIwaKCliFRSynDpv3JTUwtfZkxo6K6nce1RhCWFAsMvDZL8Dgc0
yvgJ38BRsFOtkRuAGSf6ZUwTO8JJRRIFnpUzXflAnGivK9M13D5GEQMmIl6U9Pvk
sxSmbIUfc2SGJGCJD4I=
-----END CERTIFICATE-----`
		assert.True(t, IsPEMString(c))
	})

	// Well use `|` if you want to keep the newline, theses are required so the PEM document is valid.
	t.Run("From the YAML/multiline", func(t *testing.T) {
		cfg, err := load(`
enabled: true
verification_mode: null
certificate: null
key: null
key_passphrase: null
certificate_authorities:
  - |
    -----BEGIN CERTIFICATE-----
    MIIDCjCCAfKgAwIBAgITJ706Mu2wJlKckpIvkWxEHvEyijANBgkqhkiG9w0BAQsF
    ADAUMRIwEAYDVQQDDAlsb2NhbGhvc3QwIBcNMTkwNzIyMTkyOTA0WhgPMjExOTA2
    MjgxOTI5MDRaMBQxEjAQBgNVBAMMCWxvY2FsaG9zdDCCASIwDQYJKoZIhvcNAQEB
    BQADggEPADCCAQoCggEBANce58Y/JykI58iyOXpxGfw0/gMvF0hUQAcUrSMxEO6n
    fZRA49b4OV4SwWmA3395uL2eB2NB8y8qdQ9muXUdPBWE4l9rMZ6gmfu90N5B5uEl
    94NcfBfYOKi1fJQ9i7WKhTjlRkMCgBkWPkUokvBZFRt8RtF7zI77BSEorHGQCk9t
    /D7BS0GJyfVEhftbWcFEAG3VRcoMhF7kUzYwp+qESoriFRYLeDWv68ZOvG7eoWnP
    PsvZStEVEimjvK5NSESEQa9xWyJOmlOKXhkdymtcUd/nXnx6UTCFgnkgzSdTWV41
    CI6B6aJ9svCTI2QuoIq2HxX/ix7OvW1huVmcyHVxyUECAwEAAaNTMFEwHQYDVR0O
    BBYEFPwN1OceFGm9v6ux8G+DZ3TUDYxqMB8GA1UdIwQYMBaAFPwN1OceFGm9v6ux
    8G+DZ3TUDYxqMA8GA1UdEwEB/wQFMAMBAf8wDQYJKoZIhvcNAQELBQADggEBAG5D
    874A4YI7YUwOVsVAdbWtgp1d0zKcPRR+r2OdSbTAV5/gcS3jgBJ3i1BN34JuDVFw
    3DeJSYT3nxy2Y56lLnxDeF8CUTUtVQx3CuGkRg1ouGAHpO/6OqOhwLLorEmxi7tA
    H2O8mtT0poX5AnOAhzVy7QW0D/k4WaoLyckM5hUa6RtvgvLxOwA0U+VGurCDoctu
    8F4QOgTAWyh8EZIwaKCliFRSynDpv3JTUwtfZkxo6K6nce1RhCWFAsMvDZL8Dgc0
    yvgJ38BRsFOtkRuAGSf6ZUwTO8JJRRIFnpUzXflAnGivK9M13D5GEQMmIl6U9Pvk
    sxSmbIUfc2SGJGCJD4I=
    -----END CERTIFICATE-----
cipher_suites: null
curve_types: null
supported_protocols: null
  `)
		assert.NoError(t, err)
		assert.True(t, IsPEMString(cfg.CAs[0]))
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

func TestCertificate(t *testing.T) {
	// Write certificate to a temporary file.
	c := `-----BEGIN CERTIFICATE-----
MIIDCjCCAfKgAwIBAgITJ706Mu2wJlKckpIvkWxEHvEyijANBgkqhkiG9w0BAQsF
ADAUMRIwEAYDVQQDDAlsb2NhbGhvc3QwIBcNMTkwNzIyMTkyOTA0WhgPMjExOTA2
MjgxOTI5MDRaMBQxEjAQBgNVBAMMCWxvY2FsaG9zdDCCASIwDQYJKoZIhvcNAQEB
BQADggEPADCCAQoCggEBANce58Y/JykI58iyOXpxGfw0/gMvF0hUQAcUrSMxEO6n
fZRA49b4OV4SwWmA3395uL2eB2NB8y8qdQ9muXUdPBWE4l9rMZ6gmfu90N5B5uEl
94NcfBfYOKi1fJQ9i7WKhTjlRkMCgBkWPkUokvBZFRt8RtF7zI77BSEorHGQCk9t
/D7BS0GJyfVEhftbWcFEAG3VRcoMhF7kUzYwp+qESoriFRYLeDWv68ZOvG7eoWnP
PsvZStEVEimjvK5NSESEQa9xWyJOmlOKXhkdymtcUd/nXnx6UTCFgnkgzSdTWV41
CI6B6aJ9svCTI2QuoIq2HxX/ix7OvW1huVmcyHVxyUECAwEAAaNTMFEwHQYDVR0O
BBYEFPwN1OceFGm9v6ux8G+DZ3TUDYxqMB8GA1UdIwQYMBaAFPwN1OceFGm9v6ux
8G+DZ3TUDYxqMA8GA1UdEwEB/wQFMAMBAf8wDQYJKoZIhvcNAQELBQADggEBAG5D
874A4YI7YUwOVsVAdbWtgp1d0zKcPRR+r2OdSbTAV5/gcS3jgBJ3i1BN34JuDVFw
3DeJSYT3nxy2Y56lLnxDeF8CUTUtVQx3CuGkRg1ouGAHpO/6OqOhwLLorEmxi7tA
H2O8mtT0poX5AnOAhzVy7QW0D/k4WaoLyckM5hUa6RtvgvLxOwA0U+VGurCDoctu
8F4QOgTAWyh8EZIwaKCliFRSynDpv3JTUwtfZkxo6K6nce1RhCWFAsMvDZL8Dgc0
yvgJ38BRsFOtkRuAGSf6ZUwTO8JJRRIFnpUzXflAnGivK9M13D5GEQMmIl6U9Pvk
sxSmbIUfc2SGJGCJD4I=
-----END CERTIFICATE-----`
	f, err := ioutil.TempFile("", "certificate.crt")
	f.WriteString(c)
	f.Close()
	assert.NoError(t, err)
	defer os.Remove(f.Name())

	t.Run("certificate authorities", func(t *testing.T) {
		t.Run("From configuration", func(t *testing.T) {
			cfg, err := load(`
enabled: true
verification_mode: null
certificate: null
key: null
key_passphrase: null
certificate_authorities:
  - |
    -----BEGIN CERTIFICATE-----
    MIIDCjCCAfKgAwIBAgITJ706Mu2wJlKckpIvkWxEHvEyijANBgkqhkiG9w0BAQsF
    ADAUMRIwEAYDVQQDDAlsb2NhbGhvc3QwIBcNMTkwNzIyMTkyOTA0WhgPMjExOTA2
    MjgxOTI5MDRaMBQxEjAQBgNVBAMMCWxvY2FsaG9zdDCCASIwDQYJKoZIhvcNAQEB
    BQADggEPADCCAQoCggEBANce58Y/JykI58iyOXpxGfw0/gMvF0hUQAcUrSMxEO6n
    fZRA49b4OV4SwWmA3395uL2eB2NB8y8qdQ9muXUdPBWE4l9rMZ6gmfu90N5B5uEl
    94NcfBfYOKi1fJQ9i7WKhTjlRkMCgBkWPkUokvBZFRt8RtF7zI77BSEorHGQCk9t
    /D7BS0GJyfVEhftbWcFEAG3VRcoMhF7kUzYwp+qESoriFRYLeDWv68ZOvG7eoWnP
    PsvZStEVEimjvK5NSESEQa9xWyJOmlOKXhkdymtcUd/nXnx6UTCFgnkgzSdTWV41
    CI6B6aJ9svCTI2QuoIq2HxX/ix7OvW1huVmcyHVxyUECAwEAAaNTMFEwHQYDVR0O
    BBYEFPwN1OceFGm9v6ux8G+DZ3TUDYxqMB8GA1UdIwQYMBaAFPwN1OceFGm9v6ux
    8G+DZ3TUDYxqMA8GA1UdEwEB/wQFMAMBAf8wDQYJKoZIhvcNAQELBQADggEBAG5D
    874A4YI7YUwOVsVAdbWtgp1d0zKcPRR+r2OdSbTAV5/gcS3jgBJ3i1BN34JuDVFw
    3DeJSYT3nxy2Y56lLnxDeF8CUTUtVQx3CuGkRg1ouGAHpO/6OqOhwLLorEmxi7tA
    H2O8mtT0poX5AnOAhzVy7QW0D/k4WaoLyckM5hUa6RtvgvLxOwA0U+VGurCDoctu
    8F4QOgTAWyh8EZIwaKCliFRSynDpv3JTUwtfZkxo6K6nce1RhCWFAsMvDZL8Dgc0
    yvgJ38BRsFOtkRuAGSf6ZUwTO8JJRRIFnpUzXflAnGivK9M13D5GEQMmIl6U9Pvk
    sxSmbIUfc2SGJGCJD4I=
    -----END CERTIFICATE-----
cipher_suites: null
curve_types: null
supported_protocols: null
  `)
			assert.NoError(t, err)
			tlsC, err := LoadTLSConfig(cfg)
			assert.NoError(t, err)
			assert.NotNil(t, tlsC)
		})

		t.Run("From disk", func(t *testing.T) {
			// Create a dummy configuration and append the CA after.
			cfg, err := load(`
enabled: true
verification_mode: null
certificate: null
key: null
key_passphrase: null
certificate_authorities:
cipher_suites: null
curve_types: null
supported_protocols: null
  `)

			cfg.CAs = []string{f.Name()}
			tlsC, err := LoadTLSConfig(cfg)
			assert.NoError(t, err)

			assert.NotNil(t, tlsC)
		})

		t.Run("mixed from disk and embed", func(t *testing.T) {
			// Create a dummy configuration and append the CA after.
			cfg, err := load(`
enabled: true
verification_mode: null
certificate: null
key: null
key_passphrase: null
certificate_authorities:
cipher_suites: null
curve_types: null
supported_protocols: null
  `)

			cfg.CAs = []string{f.Name(), c}
			tlsC, err := LoadTLSConfig(cfg)
			assert.NoError(t, err)

			assert.NotNil(t, tlsC)
		})
	})

	t.Run("Certificate and Private keys", func(t *testing.T) {
		key := `
-----BEGIN PRIVATE KEY-----
MIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQDXHufGPycpCOfI
sjl6cRn8NP4DLxdIVEAHFK0jMRDup32UQOPW+DleEsFpgN9/ebi9ngdjQfMvKnUP
Zrl1HTwVhOJfazGeoJn7vdDeQebhJfeDXHwX2DiotXyUPYu1ioU45UZDAoAZFj5F
KJLwWRUbfEbRe8yO+wUhKKxxkApPbfw+wUtBicn1RIX7W1nBRABt1UXKDIRe5FM2
MKfqhEqK4hUWC3g1r+vGTrxu3qFpzz7L2UrRFRIpo7yuTUhEhEGvcVsiTppTil4Z
HcprXFHf5158elEwhYJ5IM0nU1leNQiOgemifbLwkyNkLqCKth8V/4sezr1tYblZ
nMh1cclBAgMBAAECggEBAKdP5jyOicqknoG9/G564RcDsDyRt64NuO7I6hBg7SZx
Jn7UKWDdFuFP/RYtoabn6QOxkVVlydp5Typ3Xu7zmfOyss479Q/HIXxmmbkD0Kp0
eRm2KN3y0b6FySsS40KDRjKGQCuGGlNotW3crMw6vOvvsLTlcKgUHF054UVCHoK/
Piz7igkDU7NjvJeha53vXL4hIjb10UtJNaGPxIyFLYRZdRPyyBJX7Yt3w8dgz8WM
epOPu0dq3bUrY3WQXcxKZo6sQjE1h7kdl4TNji5jaFlvD01Y8LnyG0oThOzf0tve
Gaw+kuy17gTGZGMIfGVcdeb+SlioXMAAfOps+mNIwTECgYEA/gTO8W0hgYpOQJzn
BpWkic3LAoBXWNpvsQkkC3uba8Fcps7iiEzotXGfwYcb5Ewf5O3Lrz1EwLj7GTW8
VNhB3gb7bGOvuwI/6vYk2/dwo84bwW9qRWP5hqPhNZ2AWl8kxmZgHns6WTTxpkRU
zrfZ5eUrBDWjRU2R8uppgRImsxMCgYEA2MxuL/C/Ko0d7XsSX1kM4JHJiGpQDvb5
GUrlKjP/qVyUysNF92B9xAZZHxxfPWpdfGGBynhw7X6s+YeIoxTzFPZVV9hlkpAA
5igma0n8ZpZEqzttjVdpOQZK8o/Oni/Q2S10WGftQOOGw5Is8+LY30XnLvHBJhO7
TKMurJ4KCNsCgYAe5TDSVmaj3dGEtFC5EUxQ4nHVnQyCpxa8npL+vor5wSvmsfUF
hO0s3GQE4sz2qHecnXuPldEd66HGwC1m2GKygYDk/v7prO1fQ47aHi9aDQB9N3Li
e7Vmtdn3bm+lDjtn0h3Qt0YygWj+wwLZnazn9EaWHXv9OuEMfYxVgYKpdwKBgEze
Zy8+WDm5IWRjn8cI5wT1DBT/RPWZYgcyxABrwXmGZwdhp3wnzU/kxFLAl5BKF22T
kRZ+D+RVZvVutebE9c937BiilJkb0AXLNJwT9pdVLnHcN2LHHHronUhV7vetkop+
kGMMLlY0lkLfoGq1AxpfSbIea9KZam6o6VKxEnPDAoGAFDCJm+ZtsJK9nE5GEMav
NHy+PwkYsHhbrPl4dgStTNXLenJLIJ+Ke0Pcld4ZPfYdSyu/Tv4rNswZBNpNsW9K
0NwJlyMBfayoPNcJKXrH/csJY7hbKviAHr1eYy9/8OL0dHf85FV+9uY5YndLcsDc
nygO9KTJuUiBrLr0AHEnqko=
-----END PRIVATE KEY-----
`

		t.Run("embed", func(t *testing.T) {
			// Create a dummy configuration and append the CA after.
			cfg, err := load(`
enabled: true
verification_mode: null
certificate: null
key: null
key_passphrase: null
certificate_authorities:
cipher_suites: null
curve_types: null
supported_protocols: null
  `)
			cfg.Certificate.Certificate = c
			cfg.Certificate.Key = key

			tlsC, err := LoadTLSConfig(cfg)
			assert.NoError(t, err)

			assert.NotNil(t, tlsC)
		})

		t.Run("embed small key", func(t *testing.T) {
			// Create a dummy configuration and append the CA after.
			cfg, err := load(`
enabled: true
verification_mode: null
certificate: null
key: null
key_passphrase: null
certificate_authorities:
cipher_suites: null
curve_types: null
supported_protocols: null
  `)
			certificate := `
-----BEGIN CERTIFICATE-----
MIIBmzCCAUCgAwIBAgIRAOQpDyaFimzmueynALHkFEcwCgYIKoZIzj0EAwIwJjEk
MCIGA1UEChMbVEVTVCAtIEVsYXN0aWMgSW50ZWdyYXRpb25zMB4XDTIxMDIwMjE1
NTkxMFoXDTQxMDEyODE1NTkxMFowJjEkMCIGA1UEChMbVEVTVCAtIEVsYXN0aWMg
SW50ZWdyYXRpb25zMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEBc7UEvBd+5SG
Z6QQfgBaPh/VAlf7ovpa/wfSmbHfBhee+dTvdAO1p90lannCkZmc7OfWAlQ1eTgJ
QW668CJwE6NPME0wDgYDVR0PAQH/BAQDAgeAMBMGA1UdJQQMMAoGCCsGAQUFBwMB
MAwGA1UdEwEB/wQCMAAwGAYDVR0RBBEwD4INZWxhc3RpYy1hZ2VudDAKBggqhkjO
PQQDAgNJADBGAiEAhpGWL4lxsdb3+hHv0y4ppw6B7IJJLCeCwHLyHt2Dkx4CIQD6
OEU+yuHzbWa18JVkHafxwnpwQmxwZA3VNitM/AyGTQ==
-----END CERTIFICATE-----
`
			key := `
-----BEGIN PRIVATE KEY-----
MIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQgFDQJ1CPLXrUbUFqj
ED8dqsGuVQdcPK7CHpsCeTtAgQqhRANCAAQFztQS8F37lIZnpBB+AFo+H9UCV/ui
+lr/B9KZsd8GF5751O90A7Wn3SVqecKRmZzs59YCVDV5OAlBbrrwInAT
-----END PRIVATE KEY-----
`
			cfg.Certificate.Certificate = certificate
			cfg.Certificate.Key = key

			tlsC, err := LoadTLSConfig(cfg)
			assert.NoError(t, err)

			assert.NotNil(t, tlsC)
		})

		t.Run("From disk", func(t *testing.T) {
			k, err := ioutil.TempFile("", "certificate.key")
			k.WriteString(key)
			k.Close()
			assert.NoError(t, err)
			defer os.Remove(k.Name())
			// Create a dummy configuration and append the CA after.
			cfg, err := load(`
enabled: true
verification_mode: null
certificate: null
key: null
key_passphrase: null
certificate_authorities:
cipher_suites: null
curve_types: null
supported_protocols: null
  `)

			cfg.Certificate.Certificate = f.Name()
			cfg.Certificate.Key = k.Name()

			tlsC, err := LoadTLSConfig(cfg)
			assert.NoError(t, err)

			assert.NotNil(t, tlsC)
		})
	})
}
