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

package tlsutil

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/transport/tlscommon"
)

func TestSetupCertReload_NilInputs(t *testing.T) {
	err := SetupCertReload(nil, nil)
	require.NoError(t, err)

	err = SetupCertReload(&tls.Config{}, nil)
	require.NoError(t, err)

	err = SetupCertReload(nil, &tlscommon.ServerConfig{})
	require.NoError(t, err)
}

func TestSetupCertReload_Disabled(t *testing.T) {
	disabled := false
	cfg := &tlscommon.ServerConfig{
		CertificateReload: tlscommon.CertificateReload{
			Enabled: &disabled,
		},
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{{}},
	}

	err := SetupCertReload(tlsConfig, cfg)
	require.NoError(t, err)
	assert.Nil(t, tlsConfig.GetCertificate, "GetCertificate should not be set when reload is disabled")
	assert.Len(t, tlsConfig.Certificates, 1, "Certificates should not be cleared when reload is disabled")
}

func TestSetupCertReload_NoCertPaths(t *testing.T) {
	cfg := &tlscommon.ServerConfig{
		Certificate: tlscommon.CertificateConfig{},
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{{}},
	}

	err := SetupCertReload(tlsConfig, cfg)
	require.NoError(t, err)
	assert.Nil(t, tlsConfig.GetCertificate, "GetCertificate should not be set when cert paths are empty")
}

func TestSetupCertReload_ValidCert(t *testing.T) {
	certPath, keyPath := writeTestCert(t)

	cfg := &tlscommon.ServerConfig{
		Certificate: tlscommon.CertificateConfig{
			Certificate: certPath,
			Key:         keyPath,
		},
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{{}},
	}

	err := SetupCertReload(tlsConfig, cfg)
	require.NoError(t, err)
	assert.NotNil(t, tlsConfig.GetCertificate, "GetCertificate should be set")
	assert.Nil(t, tlsConfig.Certificates, "Certificates should be cleared")

	cert, err := tlsConfig.GetCertificate(&tls.ClientHelloInfo{})
	require.NoError(t, err)
	assert.NotNil(t, cert)
}

func TestSetupCertReload_InvalidCertPath(t *testing.T) {
	cfg := &tlscommon.ServerConfig{
		Certificate: tlscommon.CertificateConfig{
			Certificate: "/nonexistent/cert.pem",
			Key:         "/nonexistent/key.pem",
		},
	}

	tlsConfig := &tls.Config{}

	err := SetupCertReload(tlsConfig, cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create certificate reloader")
}

func TestSetupCertReload_CustomReloadInterval(t *testing.T) {
	certPath, keyPath := writeTestCert(t)

	cfg := &tlscommon.ServerConfig{
		Certificate: tlscommon.CertificateConfig{
			Certificate: certPath,
			Key:         keyPath,
		},
		CertificateReload: tlscommon.CertificateReload{
			ReloadInterval: 10 * time.Second,
		},
	}

	tlsConfig := &tls.Config{}

	err := SetupCertReload(tlsConfig, cfg)
	require.NoError(t, err)
	assert.NotNil(t, tlsConfig.GetCertificate)
}

func TestSetupCertReload_PassphraseFromFile(t *testing.T) {
	dir := t.TempDir()
	passphrasePath := filepath.Join(dir, "passphrase")
	require.NoError(t, os.WriteFile(passphrasePath, []byte("testpass\n"), 0600))

	passphrase, err := resolvePassphrase(tlscommon.CertificateConfig{
		PassphrasePath: passphrasePath,
	})
	require.NoError(t, err)
	assert.Equal(t, "testpass", passphrase)
}

func TestSetupCertReload_PassphraseInline(t *testing.T) {
	passphrase, err := resolvePassphrase(tlscommon.CertificateConfig{
		Passphrase: "inlinepass",
	})
	require.NoError(t, err)
	assert.Equal(t, "inlinepass", passphrase)
}

func TestSetupCertReload_PassphraseInlineTakesPrecedence(t *testing.T) {
	dir := t.TempDir()
	passphrasePath := filepath.Join(dir, "passphrase")
	require.NoError(t, os.WriteFile(passphrasePath, []byte("filepass"), 0600))

	passphrase, err := resolvePassphrase(tlscommon.CertificateConfig{
		Passphrase:     "inlinepass",
		PassphrasePath: passphrasePath,
	})
	require.NoError(t, err)
	assert.Equal(t, "inlinepass", passphrase)
}

func TestSetupCertReload_PassphraseFileMissing(t *testing.T) {
	_, err := resolvePassphrase(tlscommon.CertificateConfig{
		PassphrasePath: "/nonexistent/passphrase",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read passphrase file")
}

func TestSetupCertReload_NoPassphrase(t *testing.T) {
	passphrase, err := resolvePassphrase(tlscommon.CertificateConfig{})
	require.NoError(t, err)
	assert.Empty(t, passphrase)
}

func writeTestCert(t *testing.T) (certPath, keyPath string) {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)

	keyDER, err := x509.MarshalECPrivateKey(key)
	require.NoError(t, err)

	dir := t.TempDir()
	certPath = filepath.Join(dir, "cert.pem")
	keyPath = filepath.Join(dir, "key.pem")

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	require.NoError(t, os.WriteFile(certPath, certPEM, 0600))
	require.NoError(t, os.WriteFile(keyPath, keyPEM, 0600))

	return certPath, keyPath
}
