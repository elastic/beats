// +build !integration

package outputs

import (
	"crypto/tls"
	"testing"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/outputs/transport"
	"github.com/stretchr/testify/assert"
)

// test TLS config loading

func load(yamlStr string) (*TLSConfig, error) {
	var cfg TLSConfig
	config, err := common.NewConfigWithYAML([]byte(yamlStr), "")
	if err != nil {
		return nil, err
	}

	if err = config.Unpack(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func mustLoad(t *testing.T, yamlStr string) *TLSConfig {
	cfg, err := load(yamlStr)
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

func TestEmptyTlsConfig(t *testing.T) {
	cfg, err := load("")
	assert.Nil(t, err)

	assert.Equal(t, cfg, &TLSConfig{})
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
	assert.Equal(t, cfg, &TLSConfig{})
}

func TestNoLoadNilConfig(t *testing.T) {
	cfg, err := LoadTLSConfig(nil)
	assert.Nil(t, err)
	assert.Nil(t, cfg)
}

func TestNoLoadDisabledConfig(t *testing.T) {
	enabled := false
	cfg, err := LoadTLSConfig(&TLSConfig{Enabled: &enabled})
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
  `)

	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "mycert.pem", cfg.Certificate.Certificate)
	assert.Equal(t, "mycert.key", cfg.Certificate.Key)
	assert.Len(t, cfg.CAs, 2)
	assert.Equal(t, transport.VerifyNone, cfg.VerificationMode)
	assert.Len(t, cfg.CipherSuites, 2)
	assert.Equal(t,
		[]transport.TLSVersion{transport.TLSVersion11, transport.TLSVersion12},
		cfg.Versions)
	assert.Len(t, cfg.CurveTypes, 1)
}

func TestApplyEmptyConfig(t *testing.T) {
	tmp, err := LoadTLSConfig(&TLSConfig{})
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
    certificate: logstash/ca_test.pem
    key: logstash/ca_test.key
    certificate_authorities: [logstash/ca_test.pem]
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
			"supported_protocols: [UnknwonTLSv1.1]",
		},
		{
			"unknown curve type",
			"curve_types: ['unknown curve type']",
		},
	}

	for i, test := range tests {
		t.Logf("run test (%v): %v", i, test.title)

		config, err := common.NewConfigWithYAML([]byte(test.yaml), "")
		if err != nil {
			t.Error(err)
			continue
		}

		// one must fail: validators on Unpack or transformation to *tls.Config
		var tlscfg TLSConfig
		if err = config.Unpack(&tlscfg); err != nil {
			t.Log(err)
			continue
		}
		_, err = LoadTLSConfig(&tlscfg)
		t.Log(err)
		assert.Error(t, err)
	}
}
