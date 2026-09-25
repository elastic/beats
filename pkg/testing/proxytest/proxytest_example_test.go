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

//go:build example

package proxytest

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/testing/certutil"
)

// TestRunHTTPSProxy is an example of how to use the proxytest outside tests,
// and it instructs how to perform a request through the proxy using cURL.
// From the repo's root, run this test with:
// go test -tags example -v -run TestRunHTTPSProxy$ ./testing/proxytest
func TestRunHTTPSProxy(t *testing.T) {
	// Create a temporary directory to store certificates
	tmpDir := t.TempDir()

	// ========================= generate certificates =========================
	serverCAKey, serverCACert, serverCAPair, err := certutil.NewRootCA(
		certutil.WithCNPrefix("server"))
	require.NoError(t, err, "error creating root CA")

	serverCert, _, err := certutil.GenerateChildCert(
		"localhost",
		[]net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback, net.IPv6zero},
		serverCAKey,
		serverCACert, certutil.WithCNPrefix("server"))
	require.NoError(t, err, "error creating server certificate")
	serverCACertPool := x509.NewCertPool()
	serverCACertPool.AddCert(serverCACert)

	proxyCAKey, proxyCACert, proxyCAPair, err := certutil.NewRootCA(
		certutil.WithCNPrefix("proxy"))
	require.NoError(t, err, "error creating root CA")

	proxyCert, proxyCertPair, err := certutil.GenerateChildCert(
		"localhost",
		[]net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback, net.IPv6zero},
		proxyCAKey,
		proxyCACert,
		certutil.WithCNPrefix("proxy"))
	require.NoError(t, err, "error creating server certificate")

	clientCAKey, clientCACert, clientCAPair, err := certutil.NewRootCA(
		certutil.WithCNPrefix("client"))
	require.NoError(t, err, "error creating root CA")
	clientCACertPool := x509.NewCertPool()
	clientCACertPool.AddCert(clientCACert)

	_, clientCertPair, err := certutil.GenerateChildCert(
		"localhost",
		[]net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback, net.IPv6zero},
		clientCAKey,
		clientCACert,
		certutil.WithCNPrefix("client"))
	require.NoError(t, err, "error creating server certificate")

	// =========================== save certificates ===========================
	serverCACertFile := filepath.Join(tmpDir, "serverCA.crt")
	if err := os.WriteFile(serverCACertFile, serverCAPair.Cert, 0644); err != nil {
		t.Fatal(err)
	}
	serverCAKeyFile := filepath.Join(tmpDir, "serverCA.key")
	if err := os.WriteFile(serverCAKeyFile, serverCAPair.Key, 0644); err != nil {
		t.Fatal(err)
	}

	proxyCACertFile := filepath.Join(tmpDir, "proxyCA.crt")
	if err := os.WriteFile(proxyCACertFile, proxyCAPair.Cert, 0644); err != nil {
		t.Fatal(err)
	}
	proxyCAKeyFile := filepath.Join(tmpDir, "proxyCA.key")
	if err := os.WriteFile(proxyCAKeyFile, proxyCAPair.Key, 0644); err != nil {
		t.Fatal(err)
	}
	proxyCertFile := filepath.Join(tmpDir, "proxyCert.crt")
	if err := os.WriteFile(proxyCertFile, proxyCertPair.Cert, 0644); err != nil {
		t.Fatal(err)
	}
	proxyKeyFile := filepath.Join(tmpDir, "proxyCert.key")
	if err := os.WriteFile(proxyKeyFile, proxyCertPair.Key, 0644); err != nil {
		t.Fatal(err)
	}

	clientCACertFile := filepath.Join(tmpDir, "clientCA.crt")
	if err := os.WriteFile(clientCACertFile, clientCAPair.Cert, 0644); err != nil {
		t.Fatal(err)
	}
	clientCAKeyFile := filepath.Join(tmpDir, "clientCA.key")
	if err := os.WriteFile(clientCAKeyFile, clientCAPair.Key, 0644); err != nil {
		t.Fatal(err)
	}
	clientCertCertFile := filepath.Join(tmpDir, "clientCert.crt")
	if err := os.WriteFile(clientCertCertFile, clientCertPair.Cert, 0644); err != nil {
		t.Fatal(err)
	}
	clientCertKeyFile := filepath.Join(tmpDir, "clientCert.key")
	if err := os.WriteFile(clientCertKeyFile, clientCertPair.Key, 0644); err != nil {
		t.Fatal(err)
	}

	// ========================== create target server =========================
	targetHost := "not-a-server.co"
	server := httptest.NewUnstartedServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("It works!"))
		}))
	server.TLS = &tls.Config{
		Certificates: []tls.Certificate{*serverCert},
		MinVersion:   tls.VersionTLS13,
	}
	server.StartTLS()
	t.Logf("target server running on %s", server.URL)

	// ============================== create proxy =============================
	proxy := New(t,
		WithVerboseLog(),
		WithRequestLog("https", t.Logf),
		WithRewrite(targetHost+":443", server.URL[8:]),
		WithMITMCA(proxyCAKey, proxyCACert),
		WithHTTPClient(&http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					RootCAs:    serverCACertPool,
					MinVersion: tls.VersionTLS13,
				},
			},
		}),
		WithServerTLSConfig(&tls.Config{
			Certificates: []tls.Certificate{*proxyCert},
			ClientCAs:    clientCACertPool,
			ClientAuth:   tls.VerifyClientCertIfGiven,
			MinVersion:   tls.VersionTLS13,
		}))
	err = proxy.StartTLS()
	require.NoError(t, err, "error starting proxy")
	t.Logf("proxy running on %s", proxy.LocalhostURL)
	defer proxy.Close()

	// ============================ test instructions ==========================

	u := "https://" + targetHost
	t.Logf("make request to %q using proxy %q", u, proxy.LocalhostURL)

	t.Logf(`curl \
--proxy-cacert %s \
--proxy-cert %s \
--proxy-key %s \
--cacert %s \
--proxy %s \
%s`,
		proxyCACertFile,
		clientCertCertFile,
		clientCertKeyFile,
		proxyCACertFile,
		proxy.URL,
		u,
	)

	t.Log("CTRL+C to stop")
	<-context.Background().Done()
}
