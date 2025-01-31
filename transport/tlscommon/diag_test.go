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
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"testing"

	"github.com/elastic/elastic-agent-libs/transport/tlscommontest"
	"github.com/stretchr/testify/require"
)

const verificationDefault = "verification_mode=full"

func Test_Config_DiagCerts(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		var config *Config
		p := config.DiagCerts()()
		require.Equal(t, []byte("error: nil tlscommon.Config\n"), p)
	})

	t.Run("empty", func(t *testing.T) {
		config := &Config{}
		p := config.DiagCerts()()

		require.Contains(t, string(p), verificationDefault)
		require.Contains(t, string(p), "certificate keypair is nil.")
		require.Contains(t, string(p), "certificate_authorities not provided, using system certificates.")
	})

	t.Run("with CA and cert", func(t *testing.T) {
		ca, cas := makeCAs(t)
		cert := makeCertificateConfig(t, ca)
		config := &Config{
			Certificate: cert,
			CAs:         cas,
		}
		p := config.DiagCerts()()

		require.Contains(t, string(p), verificationDefault)
		require.Contains(t, string(p), "certificate keypair OK.")
		require.Contains(t, string(p), "certificate_authorities provided.")
	})
}

func Test_ServerConfig_DiagCerts(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		var config *ServerConfig
		p := config.DiagCerts()()
		require.Equal(t, []byte("error: nil tlscommon.ServerConfig\n"), p)
	})

	t.Run("empty", func(t *testing.T) {
		config := &ServerConfig{}
		p := config.DiagCerts()()

		require.Contains(t, string(p), verificationDefault)
		require.Contains(t, string(p), "client_auth=<nil>")
		require.Contains(t, string(p), "certificate keypair is nil.")
		require.Contains(t, string(p), "certificate_authorities not provided, using system certificates.")
	})

	t.Run("with CA and cert", func(t *testing.T) {
		ca, cas := makeCAs(t)
		cert := makeCertificateConfig(t, ca)
		config := &ServerConfig{
			Certificate: cert,
			CAs:         cas,
		}
		p := config.DiagCerts()()

		require.Contains(t, string(p), verificationDefault)
		require.Contains(t, string(p), "client_auth=<nil>")
		require.Contains(t, string(p), "certificate keypair OK.")
		require.Contains(t, string(p), "certificate_authorities provided.")
	})
}

func makeCAs(t *testing.T) (tls.Certificate, []string) {
	t.Helper()
	ca, err := tlscommontest.GenCA()
	require.NoError(t, err)
	p := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: ca.Certificate[0],
	})
	require.NotEmpty(t, p)
	return ca, []string{string(p)}

}

func makeCertificateConfig(t *testing.T, ca tls.Certificate) CertificateConfig {
	t.Helper()
	crt, err := tlscommontest.GenSignedCert(ca, x509.KeyUsageDigitalSignature, false, "localhost", []string{"localhost"}, nil, false)
	require.NoError(t, err)
	crtBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: crt.Certificate[0],
	})
	require.NotEmpty(t, crtBytes)
	keyBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(crt.PrivateKey.(*rsa.PrivateKey)),
	})
	require.NotEmpty(t, keyBytes)
	return CertificateConfig{
		Certificate: string(crtBytes),
		Key:         string(keyBytes),
	}
}
