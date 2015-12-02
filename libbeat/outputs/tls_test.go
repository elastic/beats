package outputs

import (
	"crypto/tls"
	"testing"

	"github.com/stretchr/testify/assert"

	"gopkg.in/yaml.v2"
)

// test TLS config loading

func load(yamlStr string) (*TLSConfig, error) {
	var cfg TLSConfig
	if err := yaml.Unmarshal([]byte(yamlStr), &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func TestEmptyTlsConfig(t *testing.T) {
	cfg, err := load("")
	assert.Nil(t, err)

	assert.Equal(t, "", cfg.Certificate)
	assert.Equal(t, "", cfg.CertificateKey)
	assert.Len(t, cfg.CAs, 0)
	assert.Equal(t, false, cfg.Insecure)
	assert.Len(t, cfg.CipherSuites, 0)
	assert.Equal(t, "", cfg.MinVersion)
	assert.Equal(t, "", cfg.MaxVersion)
	assert.Len(t, cfg.CurveTypes, 0)
}

func TestLoadWithEmptyValues(t *testing.T) {
	cfg, err := load(`
    disabled:
    certificate:
    certificate-key:
    certificate-authorities:
    insecure:
    cipher-suites:
    min-verion:
    max-version:
    curve-types:
  `)

	assert.Nil(t, err)

	assert.Equal(t, "", cfg.Certificate)
	assert.Equal(t, "", cfg.CertificateKey)
	assert.Len(t, cfg.CAs, 0)
	assert.Equal(t, false, cfg.Insecure)
	assert.Len(t, cfg.CipherSuites, 0)
	assert.Equal(t, "", cfg.MinVersion)
	assert.Equal(t, "", cfg.MaxVersion)
	assert.Len(t, cfg.CurveTypes, 0)
}

func TestValuesSet(t *testing.T) {
	cfg, err := load(`
    disabled: true
    certificate: mycert.pem
    certificate_key: mycert.key
    certificate_authorities: ["ca1.pem", "ca2.pem"]
    insecure: true
    cipher_suites:
      - ECDHE-ECDSA-AES-256-CBC-SHA
      - ECDHE-ECDSA-AES-256-GCM-SHA384
    min_version: 1.1
    max_version: 1.2
    curve_types:
      - P-521
  `)

	assert.Nil(t, err)

	assert.Equal(t, "mycert.pem", cfg.Certificate)
	assert.Equal(t, "mycert.key", cfg.CertificateKey)
	assert.Len(t, cfg.CAs, 2)
	assert.Equal(t, true, cfg.Insecure)
	assert.Len(t, cfg.CipherSuites, 2)
	assert.Equal(t, "1.1", cfg.MinVersion)
	assert.Equal(t, "1.2", cfg.MaxVersion)
	assert.Len(t, cfg.CurveTypes, 1)
}

func TestNoLoadNilConfig(t *testing.T) {
	cfg, err := LoadTLSConfig(nil)
	assert.Nil(t, err)
	assert.Nil(t, cfg)
}

func TestApplyEmptyConfig(t *testing.T) {
	cfg, err := LoadTLSConfig(&TLSConfig{})
	assert.Nil(t, err)

	assert.Equal(t, int(tls.VersionTLS10), int(cfg.MinVersion))
	assert.Equal(t, 0, int(cfg.MaxVersion))
	assert.Len(t, cfg.Certificates, 0)
	assert.Nil(t, cfg.RootCAs)
	assert.Equal(t, false, cfg.InsecureSkipVerify)
	assert.Len(t, cfg.CipherSuites, 0)
	assert.Len(t, cfg.CurvePreferences, 0)
}

func TestApplyWithConfig(t *testing.T) {
	cfg, err := LoadTLSConfig(&TLSConfig{
		Certificate:    "logstash/ca_test.pem",
		CertificateKey: "logstash/ca_test.key",
		CAs:            []string{"logstash/ca_test.pem"},
		Insecure:       true,
		CipherSuites: []string{
			"ECDHE-ECDSA-AES-256-CBC-SHA",
			"ECDHE-ECDSA-AES-256-GCM-SHA384",
		},
		MinVersion: "1.0",
		MaxVersion: "1.2",
		CurveTypes: []string{"P-384"},
	})
	assert.Nil(t, err)

	assert.NotNil(t, cfg)
	assert.Len(t, cfg.Certificates, 1)
	assert.NotNil(t, cfg.RootCAs)
	assert.Equal(t, true, cfg.InsecureSkipVerify)
	assert.Len(t, cfg.CipherSuites, 2)
	assert.Equal(t, int(tls.VersionTLS10), int(cfg.MinVersion))
	assert.Equal(t, int(tls.VersionTLS12), int(cfg.MaxVersion))
	assert.Len(t, cfg.CurvePreferences, 1)
}

func TestCertificateWithoutKeyFail(t *testing.T) {
	_, err := LoadTLSConfig(&TLSConfig{
		Certificate: "mycert.pem",
	})
	assert.NotNil(t, err)
}

func TestCertificateKeyWithoutCertFail(t *testing.T) {
	_, err := LoadTLSConfig(&TLSConfig{
		CertificateKey: "mycert.key",
	})
	assert.NotNil(t, err)
}

func TestUnknownCipherSuiteFail(t *testing.T) {
	_, err := LoadTLSConfig(&TLSConfig{
		CipherSuites: []string{
			"Unknown cipher suite",
		},
	})
	assert.NotNil(t, err)
}

func TestUnknownVersionFail(t *testing.T) {
	_, err := LoadTLSConfig(&TLSConfig{
		MinVersion: "Unknown-Version",
	})
	assert.NotNil(t, err)
}

func TestUnknownCurveType(t *testing.T) {
	_, err := LoadTLSConfig(&TLSConfig{
		CurveTypes: []string{
			"Unknown-Curve-Type",
		},
	})
	assert.NotNil(t, err)
}
