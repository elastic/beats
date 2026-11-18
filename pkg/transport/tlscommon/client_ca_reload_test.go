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
	tests := []struct {
		name string
		// serve starts server on ln using cfg, the way callers in this repo do.
		serve func(server *http.Server, ln net.Listener, cfg *tls.Config)
		// wantProto is the application protocol expected to be negotiated.
		wantProto string
	}{
		{
			name: "tls.NewListener",
			serve: func(server *http.Server, ln net.Listener, cfg *tls.Config) {
				_ = server.Serve(tls.NewListener(ln, cfg))
			},
		},
		{
			// net/http sets up ALPN on its own copy of the config since Go 1.27, so
			// this catches the reload handling dropping changes made to copies.
			name: "ServeTLS",
			serve: func(server *http.Server, ln net.Listener, cfg *tls.Config) {
				server.TLSConfig = cfg
				_ = server.ServeTLS(ln, "", "")
			},
			wantProto: "h2",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			testServerClientCAReload(t, tc.serve, tc.wantProto)
		})
	}
}

func testServerClientCAReload(t *testing.T, serve func(*http.Server, net.Listener, *tls.Config), wantProto string) {
	const reloadInterval = time.Second

	srv := newCAReloadKeyPair(t, "server", true)
	oldClient := newCAReloadKeyPair(t, "agent-old", false)
	newClient := newCAReloadKeyPair(t, "agent-new", false)
	// Trusted by the server, but only for serving: it must not authenticate a client.
	serverOnly := newCAReloadKeyPair(t, "server-only", true)
	extraClient := newCAReloadKeyPair(t, "agent-extra", false)

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
	go serve(server, ln, common.BuildServerConfig("127.0.0.1"))
	t.Cleanup(func() { _ = server.Close() })
	addr := ln.Addr().String()

	roots := x509.NewCertPool()
	roots.AddCert(srv.cert)
	// Like a client with certificate reloading, present the certificate
	// whatever CAs the server advertises, so only the server decides. The
	// advertised CA names are returned so they can be checked.
	request := func(client caReloadKeyPair) (negotiatedProto string, acceptableCAs [][]byte, err error) {
		c := http.Client{Timeout: 5 * time.Second, Transport: &http.Transport{
			DisableKeepAlives: true,
			ForceAttemptHTTP2: true,
			TLSClientConfig: &tls.Config{
				RootCAs: roots,
				GetClientCertificate: func(info *tls.CertificateRequestInfo) (*tls.Certificate, error) {
					acceptableCAs = info.AcceptableCAs
					return &client.tlsCert, nil
				},
			},
		}}
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://"+addr+"/", nil)
		if err != nil {
			return "", nil, err
		}
		resp, err := c.Do(req)
		if err != nil {
			return "", acceptableCAs, err
		}
		return resp.TLS.NegotiatedProtocol, acceptableCAs, resp.Body.Close()
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
		{
			// Accepting a CA only present in this bundle shows it was reloaded, so the
			// next step fails because of the key usage and not a stale bundle.
			name:         "bundle with a serverAuth-only certificate accepts clientAuth certificates",
			bundle:       append(append([]byte{}, extraClient.certPEM...), serverOnly.certPEM...),
			client:       extraClient,
			wantAccepted: true,
		},
		{name: "trusted certificate without the clientAuth key usage is rejected", client: serverOnly},
	}
	for _, step := range steps {
		if step.bundle != nil {
			require.NoError(t, os.WriteFile(bundlePath, step.bundle, 0o600), "%s: rewriting bundle", step.name)
		}
		// the CA reloader re-reads the bundle on the first handshake after the reload interval
		require.EventuallyWithT(t, func(c *assert.CollectT) {
			proto, cas, err := request(step.client)
			assert.Equal(c, step.wantAccepted, err == nil, "%s: request error: %v", step.name, err)
			// The server must not advertise the CA names captured at startup, which
			// would mislead clients choosing a certificate once the bundle changes.
			assert.Empty(c, cas, "%s: advertised CA names", step.name)
			if err == nil {
				assert.Equal(c, wantProto, proto, "%s: negotiated protocol", step.name)
			}
		}, 5*reloadInterval, 100*time.Millisecond, step.name)
	}
}
