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
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/pkg/logp/logptest"
)

type caReloadKeyPair struct {
	certPEM, keyPEM []byte
	cert            *x509.Certificate
	tlsCert         tls.Certificate
}

// newCAReloadKeyPair creates a self-signed certificate that is its own CA.
func newCAReloadKeyPair(t *testing.T, cn string, server bool) caReloadKeyPair {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err, "generating key")
	serial, err := rand.Int(rand.Reader, big.NewInt(1<<62))
	require.NoError(t, err, "generating serial")
	tmpl := &x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: cn},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	if server {
		tmpl.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
		tmpl.IPAddresses = []net.IP{net.ParseIP("127.0.0.1")}
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	require.NoError(t, err, "creating certificate")
	cert, err := x509.ParseCertificate(der)
	require.NoError(t, err, "parsing certificate")
	keyDER, err := x509.MarshalECPrivateKey(key)
	require.NoError(t, err, "marshaling key")
	kp := caReloadKeyPair{
		certPEM: pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}),
		keyPEM:  pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}),
		cert:    cert,
	}
	kp.tlsCert, err = tls.X509KeyPair(kp.certPEM, kp.keyPEM)
	require.NoError(t, err, "building key pair")
	return kp
}

// TestServerClientCAReload checks that a running server trusts the client CAs
// currently on disk: a CA added to the bundle after startup must be accepted,
// and a CA removed from it rejected, without a restart.
func TestServerClientCAReload(t *testing.T) {
	const reloadInterval = time.Second

	srv := newCAReloadKeyPair(t, "server", true)
	oldClient := newCAReloadKeyPair(t, "agent-old", false)
	newClient := newCAReloadKeyPair(t, "agent-new", false)

	dir := t.TempDir()
	certPath, keyPath := filepath.Join(dir, "srv.crt"), filepath.Join(dir, "srv.key")
	bundlePath := filepath.Join(dir, "client-trust-bundle.crt")
	require.NoError(t, os.WriteFile(certPath, srv.certPEM, 0o600), "writing server cert")
	require.NoError(t, os.WriteFile(keyPath, srv.keyPEM, 0o600), "writing server key")
	require.NoError(t, os.WriteFile(bundlePath, oldClient.certPEM, 0o600), "writing initial bundle")

	required := TLSClientAuthRequired
	reload := true
	common, err := LoadTLSServerConfig(&ServerConfig{
		Certificate:       CertificateConfig{Certificate: certPath, Key: keyPath},
		CAs:               []string{bundlePath},
		ClientAuth:        &required,
		CertificateReload: CertificateReload{Enabled: &reload, ReloadInterval: reloadInterval},
	}, logptest.NewTestingLogger(t, ""))
	require.NoError(t, err, "loading server TLS config")

	ln, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err, "listening")
	server := &http.Server{
		Handler:           http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}),
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() { _ = server.Serve(tls.NewListener(ln, common.BuildServerConfig("127.0.0.1"))) }()
	t.Cleanup(func() { _ = server.Close() })
	addr := ln.Addr().String()

	roots := x509.NewCertPool()
	roots.AddCert(srv.cert)
	// Like a client with certificate reloading, present the certificate
	// whatever CAs the server advertises, so only the server decides.
	request := func(client caReloadKeyPair) error {
		c := http.Client{Timeout: 5 * time.Second, Transport: &http.Transport{
			DisableKeepAlives: true,
			TLSClientConfig: &tls.Config{
				RootCAs: roots,
				GetClientCertificate: func(*tls.CertificateRequestInfo) (*tls.Certificate, error) {
					return &client.tlsCert, nil
				},
			},
		}}
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://"+addr+"/", nil)
		if err != nil {
			return err
		}
		resp, err := c.Do(req)
		if err != nil {
			return err
		}
		return resp.Body.Close()
	}

	steps := []struct {
		name         string
		bundle       []byte // written to the bundle before the step, if not nil
		client       caReloadKeyPair
		wantAccepted bool
	}{
		{name: "initial CA is accepted", client: oldClient, wantAccepted: true},
		{name: "CA not in the bundle is rejected", client: newClient},
		{name: "CA added to the bundle is accepted", bundle: newClient.certPEM, client: newClient, wantAccepted: true},
		{name: "CA removed from the bundle is rejected", client: oldClient},
	}
	for _, step := range steps {
		if step.bundle != nil {
			require.NoError(t, os.WriteFile(bundlePath, step.bundle, 0o600), "%s: rewriting bundle", step.name)
		}
		// the CA reloader re-reads the bundle on the first handshake after the reload interval
		require.EventuallyWithT(t, func(c *assert.CollectT) {
			err := request(step.client)
			assert.Equal(c, step.wantAccepted, err == nil, "%s: request error: %v", step.name, err)
		}, 5*reloadInterval, 100*time.Millisecond, step.name)
	}
}
