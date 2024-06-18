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

package httpcommon

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/elastic/elastic-agent-libs/transport/tlscommon"
	"github.com/stretchr/testify/require"
)

var ser int64 = 1

func Test_HTTPTransportSettings_DiagRequests(t *testing.T) {
	t.Run("nil settings", func(t *testing.T) {
		var settings *HTTPTransportSettings
		p := settings.DiagRequests(nil)()
		require.Equal(t, []byte(`error: nil httpcommon.HTTPTransportSettings`), p)
	})

	t.Run("no requests", func(t *testing.T) {
		settings := &HTTPTransportSettings{}
		p := settings.DiagRequests(nil)()
		require.Equal(t, []byte(`error: 0 requests`), p)
	})

	t.Run("request OK", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer srv.Close()
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		settings := DefaultHTTPTransportSettings()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL, nil)
		require.NoError(t, err)
		p := settings.DiagRequests([]*http.Request{req})()

		require.Contains(t, string(p), "No TLS settings")
		require.Contains(t, string(p), "request 0 successful.")
	})

	t.Run("TLS server with no settings", func(t *testing.T) {
		srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer srv.Close()
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		settings := DefaultHTTPTransportSettings()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL, nil)
		require.NoError(t, err)
		p := settings.DiagRequests([]*http.Request{req})()

		require.Contains(t, string(p), "No TLS settings")
		require.Contains(t, string(p), "request 0 error:")
	})

	t.Run("TLS server with settings", func(t *testing.T) {
		srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer srv.Close()
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		ca := pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: srv.TLS.Certificates[0].Certificate[0],
		})
		require.NotEmpty(t, ca)
		settings := DefaultHTTPTransportSettings()
		settings.TLS = &tlscommon.Config{
			CAs: []string{string(ca)},
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL, nil)
		require.NoError(t, err)
		p := settings.DiagRequests([]*http.Request{req})()

		require.Contains(t, string(p), "TLS settings detected")
		require.Contains(t, string(p), "request 0 successful.")
	})
}

func Test_isGoHTTPResp(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	t.Run("http request", func(t *testing.T) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.Replace(srv.URL, "https", "http", 1), nil)
		require.NoError(t, err)
		resp, err := srv.Client().Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		require.True(t, isGoHTTPResp(resp), "expected go http response on https server")
	})
	t.Run("https request", func(t *testing.T) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL, nil)
		require.NoError(t, err)
		resp, err := srv.Client().Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		require.False(t, isGoHTTPResp(resp), "expected go https response on https server")
	})
}

func Test_HTTPRequestOnHTTPSPort(t *testing.T) {
	// This checks if HTTP was use used to communicate with an HTTPS server resulting in an error
	// isGoHTTPResp does the same but requires that the responding server is a go http.Server
	t.Skip("test used to validate behaviour of http.Client on external servers")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://google.com:443", nil)
	require.NoError(t, err)
	_, err = (&http.Client{}).Do(req) //nolint:bodyclose // expected to return an error
	require.Error(t, err)

	var nErr *net.OpError
	require.ErrorAs(t, err, &nErr)
	require.Contains(t, diagError(err), "possible cause: HTTP schema used for HTTPS server.")
}

func Test_diagError(t *testing.T) {
	t.Run("TLS server no client CA", func(t *testing.T) {
		srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer srv.Close()
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL, nil)
		require.NoError(t, err)

		_, err = (&http.Client{}).Do(req) //nolint:bodyclose // expected to return an error
		require.Error(t, err)
		require.Contains(t, diagError(err), "caused by no trusted client CA.")
	})

	t.Run("HTTPS schema used on HTTP server", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer srv.Close()
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.Replace(srv.URL, "http", "https", 1), nil)
		require.NoError(t, err)

		_, err = srv.Client().Do(req) //nolint:bodyclose // expected to return an error
		require.Error(t, err)
		require.Contains(t, diagError(err), "caused by using HTTPS schema on HTTP server.")
	})

	t.Run("Server cert expired", func(t *testing.T) {
		ca := genCA(t)
		crt := genSignedCert(t, ca, x509.KeyUsageDigitalSignature, false, "localhost", []string{"localhost"}, []net.IP{net.ParseIP("127.0.0.1")}, true)
		pool := x509.NewCertPool()
		pool.AddCert(ca.Leaf)

		srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		srv.TLS = &tls.Config{ //nolint:gosec //used for tests
			Certificates: []tls.Certificate{crt},
		}
		srv.StartTLS()
		defer srv.Close()
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL, nil)
		require.NoError(t, err)

		client := http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{ //nolint:gosec //used for tests
					RootCAs: pool,
				},
			},
		}
		_, err = client.Do(req) //nolint:bodyclose // expected to return an error
		require.Error(t, err)
		require.Contains(t, diagError(err), "caused by invalid server certificate.")
	})

	t.Run("Server requires client auth", func(t *testing.T) {
		ca := genCA(t)
		crt := genSignedCert(t, ca, x509.KeyUsageDigitalSignature, false, "localhost", []string{"localhost"}, []net.IP{net.ParseIP("127.0.0.1")}, false)
		pool := x509.NewCertPool()
		pool.AddCert(ca.Leaf)

		srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		srv.TLS = &tls.Config{ //nolint:gosec //used for tests
			Certificates: []tls.Certificate{crt},
			ClientAuth:   tls.RequireAndVerifyClientCert,
		}
		srv.StartTLS()
		defer srv.Close()
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL, nil)
		require.NoError(t, err)

		client := http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{ //nolint:gosec //used for tests
					RootCAs: pool,
				},
			},
		}
		_, err = client.Do(req) //nolint:bodyclose // expected to return an error
		require.Error(t, err)
		require.Contains(t, diagError(err), "caused by missing mTLS client cert.")
	})

	t.Run("Server requires client auth, client cert expired", func(t *testing.T) {
		ca := genCA(t)
		clientCA := genCA(t)
		crt := genSignedCert(t, ca, x509.KeyUsageDigitalSignature, false, "localhost", []string{"localhost"}, []net.IP{net.ParseIP("127.0.0.1")}, false)
		clientCrt := genSignedCert(t, clientCA, x509.KeyUsageDigitalSignature, false, "localhost", []string{"localhost"}, []net.IP{net.ParseIP("127.0.0.1")}, true)
		pool := x509.NewCertPool()
		pool.AddCert(ca.Leaf)
		serverPool := x509.NewCertPool()
		serverPool.AddCert(clientCA.Leaf)

		srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		srv.TLS = &tls.Config{ //nolint:gosec //used for tests
			Certificates: []tls.Certificate{crt},
			ClientAuth:   tls.RequireAndVerifyClientCert,
			ClientCAs:    serverPool,
		}
		srv.StartTLS()
		defer srv.Close()
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL, nil)
		require.NoError(t, err)

		client := http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{ //nolint:gosec //used for tests
					Certificates: []tls.Certificate{clientCrt},
					RootCAs:      pool,
				},
			},
		}
		_, err = client.Do(req) //nolint:bodyclose // expected to return an error
		require.Error(t, err)
		// different OSes seem to report different TLS errors, so just check for the "expired" string.
		require.Contains(t, diagError(err), "expired")
	})
}

// copied from tlscommon
func genCA(t *testing.T) tls.Certificate {
	t.Helper()
	ca := &x509.Certificate{
		SerialNumber: serial(),
		Subject: pkix.Name{
			CommonName:    "localhost",
			Organization:  []string{"TESTING"},
			Country:       []string{"CANADA"},
			Province:      []string{"QUEBEC"},
			Locality:      []string{"MONTREAL"},
			StreetAddress: []string{"testing road"},
			PostalCode:    []string{"HOH OHO"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(1 * time.Hour),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	caKey, err := rsa.GenerateKey(rand.Reader, 2048) // less secure key for quicker testing.
	require.NoError(t, err)

	caBytes, err := x509.CreateCertificate(rand.Reader, ca, ca, &caKey.PublicKey, caKey)
	require.NoError(t, err)

	leaf, err := x509.ParseCertificate(caBytes)
	require.NoError(t, err)

	return tls.Certificate{
		Certificate: [][]byte{caBytes},
		PrivateKey:  caKey,
		Leaf:        leaf,
	}
}

// genSignedCert generates a CA and KeyPair and remove the need to depends on code of agent.
func genSignedCert(
	t *testing.T,
	ca tls.Certificate,
	keyUsage x509.KeyUsage,
	isCA bool,
	commonName string,
	dnsNames []string,
	ips []net.IP,
	expired bool,
) tls.Certificate {
	t.Helper()
	if commonName == "" {
		commonName = "You know, for search"
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(5 * time.Hour)

	if expired {
		notBefore = notBefore.Add(-42 * time.Hour)
		notAfter = notAfter.Add(-42 * time.Hour)
	}
	// Create another Cert/key
	cert := &x509.Certificate{
		SerialNumber: big.NewInt(2000),

		// SNA - Subject Alternative Name fields
		IPAddresses: ips,
		DNSNames:    dnsNames,

		Subject: pkix.Name{
			CommonName:    commonName,
			Organization:  []string{"TESTING"},
			Country:       []string{"CANADA"},
			Province:      []string{"QUEBEC"},
			Locality:      []string{"MONTREAL"},
			StreetAddress: []string{"testing road"},
			PostalCode:    []string{"HOH OHO"},
		},

		NotBefore:             notBefore,
		NotAfter:              notAfter,
		IsCA:                  isCA,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              keyUsage,
		BasicConstraintsValid: true,
	}

	certKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	certBytes, err := x509.CreateCertificate(
		rand.Reader,
		cert,
		ca.Leaf,
		&certKey.PublicKey,
		ca.PrivateKey,
	)
	require.NoError(t, err)

	leaf, err := x509.ParseCertificate(certBytes)
	require.NoError(t, err)

	return tls.Certificate{
		Certificate: [][]byte{certBytes},
		PrivateKey:  certKey,
		Leaf:        leaf,
	}
}

func serial() *big.Int {
	ser = ser + 1
	return big.NewInt(ser)
}
