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
	"bytes"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMakeVerifyServerConnection(t *testing.T) {
	testCerts := genTestCerts(t)

	certPool := x509.NewCertPool()
	certPool.AddCert(testCerts["ca"])

	testcases := map[string]struct {
		verificationMode TLSVerificationMode
		clientAuth       tls.ClientAuthType
		certAuthorities  *x509.CertPool
		peerCerts        []*x509.Certificate
		serverName       string
		expectedCallback bool
		expectedError    error
	}{
		"default verification without certificates when required": {
			verificationMode: VerifyFull,
			clientAuth:       tls.RequireAndVerifyClientCert,
			peerCerts:        nil,
			serverName:       "",
			expectedCallback: true,
			expectedError:    ErrMissingPeerCertificate,
		},
		"default verification with certificates when required with expired cert": {
			verificationMode: VerifyFull,
			clientAuth:       tls.RequireAndVerifyClientCert,
			certAuthorities:  certPool,
			peerCerts:        []*x509.Certificate{testCerts["expired"]},
			serverName:       "",
			expectedCallback: true,
			expectedError:    x509.CertificateInvalidError{Cert: testCerts["expired"], Reason: x509.Expired},
		},
		"default verification with certificates when required with incorrect server name in cert": {
			verificationMode: VerifyFull,
			clientAuth:       tls.RequireAndVerifyClientCert,
			certAuthorities:  certPool,
			peerCerts:        []*x509.Certificate{testCerts["correct"]},
			serverName:       "bad.example.com",
			expectedCallback: true,
			expectedError:    x509.HostnameError{Certificate: testCerts["correct"], Host: "bad.example.com"},
		},
		"default verification with certificates when required with correct cert": {
			verificationMode: VerifyFull,
			clientAuth:       tls.RequireAndVerifyClientCert,
			certAuthorities:  certPool,
			peerCerts:        []*x509.Certificate{testCerts["correct"]},
			serverName:       "localhost",
			expectedCallback: true,
			expectedError:    nil,
		},
		"default verification with certificates when required with correct wildcard cert": {
			verificationMode: VerifyFull,
			clientAuth:       tls.RequireAndVerifyClientCert,
			certAuthorities:  certPool,
			peerCerts:        []*x509.Certificate{testCerts["wildcard"]},
			serverName:       "hello.example.com",
			expectedCallback: true,
			expectedError:    nil,
		},
		"certificate verification with certificates when required with correct cert": {
			verificationMode: VerifyCertificate,
			clientAuth:       tls.RequireAndVerifyClientCert,
			certAuthorities:  certPool,
			peerCerts:        []*x509.Certificate{testCerts["correct"]},
			serverName:       "localhost",
			expectedCallback: true,
			expectedError:    nil,
		},
		"certificate verification with certificates when required with expired cert": {
			verificationMode: VerifyCertificate,
			clientAuth:       tls.RequireAndVerifyClientCert,
			certAuthorities:  certPool,
			peerCerts:        []*x509.Certificate{testCerts["expired"]},
			serverName:       "localhost",
			expectedCallback: true,
			expectedError:    x509.CertificateInvalidError{Cert: testCerts["expired"], Reason: x509.Expired},
		},
		"certificate verification with certificates when required with incorrect server name in cert": {
			verificationMode: VerifyCertificate,
			clientAuth:       tls.RequireAndVerifyClientCert,
			certAuthorities:  certPool,
			peerCerts:        []*x509.Certificate{testCerts["correct"]},
			serverName:       "bad.example.com",
			expectedCallback: true,
			expectedError:    nil,
		},
		"strict verification with certificates when required with correct cert": {
			verificationMode: VerifyStrict,
			clientAuth:       tls.RequireAndVerifyClientCert,
			certAuthorities:  certPool,
			peerCerts:        []*x509.Certificate{testCerts["correct"]},
			serverName:       "localhost",
			expectedCallback: false,
			expectedError:    nil,
		},
		"default verification with certificates when required with cert signed by unknown authority": {
			verificationMode: VerifyFull,
			clientAuth:       tls.RequireAndVerifyClientCert,
			certAuthorities:  certPool,
			peerCerts:        []*x509.Certificate{testCerts["unknown_authority"]},
			serverName:       "",
			expectedCallback: true,
			expectedError:    x509.UnknownAuthorityError{Cert: testCerts["unknown_authority"]},
		},
		"default verification without certificates not required": {
			verificationMode: VerifyFull,
			clientAuth:       tls.NoClientCert,
			peerCerts:        nil,
			serverName:       "",
			expectedCallback: true,
			expectedError:    nil,
		},
		"no verification without certificates not required": {
			verificationMode: VerifyNone,
			clientAuth:       tls.NoClientCert,
			peerCerts:        nil,
			serverName:       "",
			expectedError:    nil,
		},
	}

	for name, test := range testcases {
		t.Run(name, func(t *testing.T) {
			cfg := &TLSConfig{
				Verification: test.verificationMode,
				ClientAuth:   test.clientAuth,
				ClientCAs:    test.certAuthorities,
			}

			verifier := makeVerifyServerConnection(cfg)
			if !test.expectedCallback {
				assert.Nil(t, verifier)
				return
			}

			err := verifier(tls.ConnectionState{
				PeerCertificates: test.peerCerts,
				ServerName:       test.serverName,
			})
			if test.expectedError == nil {
				assert.NoError(t, err)
			} else {
				require.Error(t, err)
				// We want to ensure the error type/message are the expected ones
				// so we compare the types and the message
				assert.IsType(t, test.expectedError, err)
				assert.Contains(t, err.Error(), test.expectedError.Error())
			}
		})
	}
}

func TestTrustRootCA(t *testing.T) {
	certs := genTestCerts(t)

	nonEmptyCertPool := x509.NewCertPool()
	nonEmptyCertPool.AddCert(certs["wildcard"])
	nonEmptyCertPool.AddCert(certs["unknown_authority"])

	fingerprint := getFingerprint(certs["ca"])

	testCases := []struct {
		name                 string
		rootCAs              *x509.CertPool
		caTrustedFingerprint string
		peerCerts            []*x509.Certificate
		expectingError       bool
		expectedRootCAsLen   int
	}{
		{
			name:                 "RootCA cert matches the fingerprint and is added to cfg.RootCAs",
			caTrustedFingerprint: fingerprint,
			peerCerts:            []*x509.Certificate{certs["correct"], certs["ca"]},
			expectedRootCAsLen:   1,
		},
		{
			name:                 "RootCA cert doesn not matche the fingerprint and is not added to cfg.RootCAs",
			caTrustedFingerprint: fingerprint,
			peerCerts:            []*x509.Certificate{certs["correct"], certs["ca"]},
			expectedRootCAsLen:   0,
		},
		{
			name:                 "non empty CertPool has the RootCA added",
			rootCAs:              nonEmptyCertPool,
			caTrustedFingerprint: fingerprint,
			peerCerts:            []*x509.Certificate{certs["correct"], certs["ca"]},
			expectedRootCAsLen:   3,
		},
		{
			name:                 "invalis HEX encoding",
			caTrustedFingerprint: "INVALID ENCODING",
			expectedRootCAsLen:   0,
			expectingError:       true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := TLSConfig{
				RootCAs:              tc.rootCAs,
				CATrustedFingerprint: tc.caTrustedFingerprint,
			}
			err := trustRootCA(&cfg, tc.peerCerts)
			if tc.expectingError && err == nil {
				t.Fatal("expecting an error when calling trustRootCA")
			}

			if !tc.expectingError && err != nil {
				t.Fatalf("did not expect an error calling trustRootCA: %v", err)
			}

			if tc.expectedRootCAsLen != 0 {
				if cfg.RootCAs == nil {
					t.Fatal("cfg.RootCAs cannot be nil")
				}

				// we want to know the number of certificates in the CertPool (RootCAs), as it is not
				// directly available, we use this workaround of reading the number of subjects in the pool.
				if got, expected := len(cfg.RootCAs.Subjects()), tc.expectedRootCAsLen; got != expected {
					t.Fatalf("expecting cfg.RootCAs to have %d element, got %d instead", expected, got)
				}
			}
		})
	}
}

func TestMakeVerifyConnectionUsesCATrustedFingerprint(t *testing.T) {
	testCerts := genTestCerts(t)
	fingerprint := getFingerprint(testCerts["ca"])

	testcases := map[string]struct {
		verificationMode     TLSVerificationMode
		peerCerts            []*x509.Certificate
		serverName           string
		expectedCallback     bool
		expectingError       bool
		CATrustedFingerprint string
		CASHA256             []string
	}{
		"CATrustedFingerprint and verification mode:VerifyFull": {
			verificationMode:     VerifyFull,
			peerCerts:            []*x509.Certificate{testCerts["correct"], testCerts["ca"]},
			serverName:           "localhost",
			expectedCallback:     true,
			CATrustedFingerprint: fingerprint,
		},
		"CATrustedFingerprint and verification mode:VerifyCertificate": {
			verificationMode:     VerifyCertificate,
			peerCerts:            []*x509.Certificate{testCerts["correct"], testCerts["ca"]},
			serverName:           "localhost",
			expectedCallback:     true,
			CATrustedFingerprint: fingerprint,
		},
		"CATrustedFingerprint and verification mode:VerifyStrict": {
			verificationMode:     VerifyStrict,
			peerCerts:            []*x509.Certificate{testCerts["correct"], testCerts["ca"]},
			serverName:           "localhost",
			expectedCallback:     true,
			CATrustedFingerprint: fingerprint,
			CASHA256:             []string{Fingerprint(testCerts["correct"])},
		},
		"CATrustedFingerprint and verification mode:VerifyNone": {
			verificationMode: VerifyNone,
			peerCerts:        []*x509.Certificate{testCerts["correct"], testCerts["ca"]},
			serverName:       "localhost",
			expectedCallback: false,
		},
		"invalid CATrustedFingerprint and verification mode:VerifyFull returns error": {
			verificationMode:     VerifyFull,
			peerCerts:            []*x509.Certificate{testCerts["correct"], testCerts["ca"]},
			serverName:           "localhost",
			expectedCallback:     true,
			CATrustedFingerprint: "INVALID HEX ENCODING",
			expectingError:       true,
		},
		"invalid CATrustedFingerprint and verification mode:VerifyCertificate returns error": {
			verificationMode:     VerifyCertificate,
			peerCerts:            []*x509.Certificate{testCerts["correct"], testCerts["ca"]},
			serverName:           "localhost",
			expectedCallback:     true,
			CATrustedFingerprint: "INVALID HEX ENCODING",
			expectingError:       true,
		},
		"invalid CATrustedFingerprint and verification mode:VerifyStrict returns error": {
			verificationMode:     VerifyStrict,
			peerCerts:            []*x509.Certificate{testCerts["correct"], testCerts["ca"]},
			serverName:           "localhost",
			expectedCallback:     true,
			CATrustedFingerprint: "INVALID HEX ENCODING",
			expectingError:       true,
			CASHA256:             []string{Fingerprint(testCerts["correct"])},
		},
	}

	for name, test := range testcases {
		t.Run(name, func(t *testing.T) {
			cfg := &TLSConfig{
				Verification:         test.verificationMode,
				CATrustedFingerprint: test.CATrustedFingerprint,
				CASha256:             test.CASHA256,
			}

			verifier := makeVerifyConnection(cfg)
			if test.expectedCallback {
				require.NotNil(t, verifier, "makeVerifyConnection returned a nil verifier")
			} else {
				require.Nil(t, verifier)
				return
			}

			err := verifier(tls.ConnectionState{
				PeerCertificates: test.peerCerts,
				ServerName:       test.serverName,
				VerifiedChains:   [][]*x509.Certificate{test.peerCerts},
			})
			if test.expectingError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestMakeVerifyServerConnectionForIPs(t *testing.T) {
	testcases := map[string]struct {
		dnsNames         []string
		commonName       string
		expectingError   bool
		ips              []net.IP
		peerCerts        []*x509.Certificate
		serverName       string
		verificationMode TLSVerificationMode
	}{
		"IP matches the Certificate IPs field": {
			expectingError: false,
			ips:            []net.IP{net.IPv4(127, 0, 0, 1)},
			serverName:     "127.0.0.1",
			commonName:     "a.host.name.elastic.co",
		},
		"IP does not match the Certificate IPs field": {
			expectingError: true,
			ips:            []net.IP{net.IPv4(192, 168, 42, 42)},
			serverName:     "127.0.0.1",
			commonName:     "a.host.name.elastic.co",
		},
		"IP in SNA hostnames do not work": {
			expectingError: true,
			dnsNames:       []string{"127.0.0.1"},
			serverName:     "127.0.0.1",
			commonName:     "a.host.name.elastic.co",
		},
		"IP in CN works": {
			expectingError: false,
			serverName:     "127.0.0.1",
			commonName:     "127.0.0.1",
		},
	}

	ca, err := genCA()
	if err != nil {
		t.Fatalf("cannot generate CA certificate: %s", err)
	}

	rootCAs := x509.NewCertPool()
	rootCAs.AddCert(ca.Leaf)

	for name, test := range testcases {
		t.Run(name, func(t *testing.T) {
			peerCerts, err := genSignedCert(
				ca,
				x509.KeyUsageCertSign,
				false,
				test.commonName,
				test.dnsNames,
				test.ips,
				false)
			if err != nil {
				t.Fatalf("cannot generate peer certificate: %s", err)
			}

			cfg := &TLSConfig{
				RootCAs:      rootCAs,
				Verification: test.verificationMode,
				ServerName:   test.serverName,
			}
			verifier := makeVerifyConnection(cfg)

			err = verifier(tls.ConnectionState{
				PeerCertificates: []*x509.Certificate{peerCerts.Leaf},
				VerifiedChains:   [][]*x509.Certificate{test.peerCerts},
			})

			if test.expectingError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestVerificationMode(t *testing.T) {
	testcases := map[string]struct {
		verificationMode TLSVerificationMode
		expectingError   bool

		// hostname is used to make connection
		hostname string

		// ignoreCerts do not add the Root CA to the trust chain
		ignoreCerts bool

		// commonName used in the Certificate
		commonName string

		// dnsNames is used as the SNA DNSNames
		dnsNames []string

		// ips is used as the SNA IPAddresses
		ips []net.IP
	}{
		"VerifyFull validates domain": {
			verificationMode: VerifyFull,
			hostname:         "localhost",
			dnsNames:         []string{"localhost"},
		},
		"VerifyFull validates IPv4": {
			verificationMode: VerifyFull,
			hostname:         "127.0.0.1",
			ips:              []net.IP{net.IPv4(127, 0, 0, 1)},
		},
		"VerifyFull validates IPv6": {
			verificationMode: VerifyFull,
			hostname:         "::1",
			ips:              []net.IP{net.ParseIP("::1")},
		},
		"VerifyFull domain mismatch returns error": {
			verificationMode: VerifyFull,
			hostname:         "localhost",
			dnsNames:         []string{"example.com"},
			expectingError:   true,
		},
		"VerifyFull IPv4 mismatch returns error": {
			verificationMode: VerifyFull,
			hostname:         "127.0.0.1",
			ips:              []net.IP{net.IPv4(10, 0, 0, 1)},
			expectingError:   true,
		},
		"VerifyFull IPv6 mismatch returns error": {
			verificationMode: VerifyFull,
			hostname:         "::1",
			ips:              []net.IP{net.ParseIP("faca:b0de:baba::ca")},
			expectingError:   true,
		},
		"VerifyFull does not return error when SNA is empty and legacy Common Name is used": {
			verificationMode: VerifyFull,
			hostname:         "localhost",
			commonName:       "localhost",
			expectingError:   false,
		},
		"VerifyFull does not return error when SNA is empty and legacy Common Name is used with IP address": {
			verificationMode: VerifyFull,
			hostname:         "127.0.0.1",
			commonName:       "127.0.0.1",
			expectingError:   false,
		},

		"VerifyStrict validates domain": {
			verificationMode: VerifyStrict,
			hostname:         "localhost",
			dnsNames:         []string{"localhost"},
		},
		"VerifyStrict validates IPv4": {
			verificationMode: VerifyStrict,
			hostname:         "127.0.0.1",
			ips:              []net.IP{net.IPv4(127, 0, 0, 1)},
		},
		"VerifyStrict validates IPv6": {
			verificationMode: VerifyStrict,
			hostname:         "::1",
			ips:              []net.IP{net.ParseIP("::1")},
		},
		"VerifyStrict domain mismatch returns error": {
			verificationMode: VerifyStrict,
			hostname:         "127.0.0.1",
			dnsNames:         []string{"example.com"},
			expectingError:   true,
		},
		"VerifyStrict IPv4 mismatch returns error": {
			verificationMode: VerifyStrict,
			hostname:         "127.0.0.1",
			ips:              []net.IP{net.IPv4(10, 0, 0, 1)},
			expectingError:   true,
		},
		"VerifyStrict IPv6 mismatch returns error": {
			verificationMode: VerifyStrict,
			hostname:         "::1",
			ips:              []net.IP{net.ParseIP("faca:b0de:baba::ca")},
			expectingError:   true,
		},
		"VerifyStrict returns error when SNA is empty and legacy Common Name is used": {
			verificationMode: VerifyStrict,
			hostname:         "localhost",
			commonName:       "localhost",
			expectingError:   true,
		},
		"VerifyStrict returns error when SNA is empty and legacy Common Name is used with IP address": {
			verificationMode: VerifyStrict,
			hostname:         "127.0.0.1",
			commonName:       "127.0.0.1",
			expectingError:   true,
		},
		"VerifyStrict returns error when SNA is empty": {
			verificationMode: VerifyStrict,
			hostname:         "localhost",
			expectingError:   true,
		},

		"VerifyCertificate does not validate domain": {
			verificationMode: VerifyCertificate,
			hostname:         "localhost",
			dnsNames:         []string{"example.com"},
		},
		"VerifyCertificate does not validate IPv4": {
			verificationMode: VerifyCertificate,
			hostname:         "127.0.0.1",
			dnsNames:         []string{"example.com"}, // I believe it cannot be empty
		},
		"VerifyCertificate does not validate IPv6": {
			verificationMode: VerifyCertificate,
			hostname:         "127.0.0.1",
			ips:              []net.IP{net.ParseIP("faca:b0de:baba::ca")},
		},

		"VerifyNone accepts untrusted certificates": {
			verificationMode: VerifyNone,
			hostname:         "127.0.0.1",
			ignoreCerts:      true,
		},
	}
	caCert, err := genCA()
	if err != nil {
		t.Fatalf("could not generate root CA certificate: %s", err)
	}

	certPool := x509.NewCertPool()
	certPool.AddCert(caCert.Leaf)

	for name, test := range testcases {
		t.Run(name, func(t *testing.T) {
			certs, err := genSignedCert(caCert, x509.KeyUsageCertSign, false, test.commonName, test.dnsNames, test.ips, false)
			if err != nil {
				t.Fatalf("could not generate certificates: %s", err)
			}
			serverURL := startTestServer(t, "localhost:0", []tls.Certificate{certs})

			tlsC := TLSConfig{
				Verification: test.verificationMode,
				RootCAs:      certPool,
				ServerName:   test.hostname,
			}

			if test.ignoreCerts {
				tlsC.RootCAs = nil
				tlsC.ServerName = ""
			}

			client := http.Client{
				Transport: &http.Transport{
					TLSClientConfig: tlsC.BuildModuleClientConfig(test.hostname),
				},
			}

			resp, err := client.Get(serverURL.String()) //nolint:noctx // It is a test
			if err == nil {
				resp.Body.Close()
			}

			if test.expectingError {
				if err != nil {
					// We got the expected error, no need to check the status code
					return
				}
			}

			if err != nil {
				t.Fatalf("did not expect an error: %v", err)
			}

			if resp.StatusCode != 200 {
				t.Fatalf("expecting 200 got: %d", resp.StatusCode)
			}
		})
	}
}

// startTestServer starts a HTTP server for testing using the provided
// ceertificates and it binds to serverAddr.
//
// serverAddr must contain the port, e.g: localhost:12345. To get a random
// free port assigned, use port 0, e.g: "localhost:0".
//
// All requests are responded with an HTTP 200 OK and a plain
// text string
//
// The HTTP server will shutdown at the end of the test.
func startTestServer(t *testing.T, serverAddr string, serverCerts []tls.Certificate) url.URL {
	// Creates a listener on a random port selected by the OS
	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("could call net.Listen: %s", err)
	}
	t.Cleanup(func() { l.Close() })

	// l.Addr().String() will return something like: 127.0.0.1:12345,
	// add the protocol and parse the URL
	serverURL, err := url.Parse("https://" + l.Addr().String())
	if err != nil {
		t.Fatal(err)
	}

	server := http.Server{ //nolint:gosec // This server is used only for tests.
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, err := w.Write([]byte("SSL test server")); err != nil {
				t.Errorf("coluld not write to client: %s", err)
			}
		}),
		TLSConfig: &tls.Config{ //nolint:gosec // This TLS config is used only for testing.
			Certificates: serverCerts,
		},
	}
	t.Cleanup(func() { server.Close() })
	go func() {
		if err := server.ServeTLS(l, "", ""); err != nil {
			if !errors.Is(err, http.ErrServerClosed) {
				t.Errorf("HTTPS server exited unexpectedly: %s", err)
			}
		}
	}()

	return *serverURL
}

func getFingerprint(cert *x509.Certificate) string {
	caSHA256 := sha256.Sum256(cert.Raw)
	return hex.EncodeToString(caSHA256[:])
}

func genTestCerts(t *testing.T) map[string]*x509.Certificate {
	ca, err := genCA()
	if err != nil {
		t.Fatalf("cannot generate root CA: %s", err)
	}

	unknownCA, err := genCA()
	if err != nil {
		t.Fatalf("cannot generate second root CA: %s", err)
	}

	certs := map[string]*x509.Certificate{
		"ca": ca.Leaf,
	}

	certData := map[string]struct {
		ca       tls.Certificate
		keyUsage x509.KeyUsage
		isCA     bool
		dnsNames []string
		ips      []net.IP
		expired  bool
	}{
		"wildcard": {
			ca:       ca,
			keyUsage: x509.KeyUsageDigitalSignature,
			isCA:     false,
			dnsNames: []string{"*.example.com"},
		},
		"correct": {
			ca:       ca,
			keyUsage: x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
			isCA:     false,
			dnsNames: []string{"localhost"},
			// IPV4 and IPV6
			ips: []net.IP{{127, 0, 0, 1}, {0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}},
		},
		"unknown_authority": {
			ca:       unknownCA,
			keyUsage: x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
			isCA:     false,
			dnsNames: []string{"localhost"},
			// IPV4 and IPV6
			ips: []net.IP{{127, 0, 0, 1}, {0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}},
		},
		"expired": {
			ca:       ca,
			keyUsage: x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
			isCA:     false,
			dnsNames: []string{"localhost"},
			// IPV4 and IPV6
			ips:     []net.IP{{127, 0, 0, 1}, {0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}},
			expired: true,
		},
	}

	tmpDir := t.TempDir()
	for certName, data := range certData {
		cert, err := genSignedCert(
			data.ca,
			data.keyUsage,
			data.isCA,
			certName,
			data.dnsNames,
			data.ips,
			data.expired,
		)
		if err != nil {
			t.Fatalf("could not generate certificate '%s': %s", certName, err)
		}
		certs[certName] = cert.Leaf

		// We write the certificate to disk, so if the test fails the certs can
		// be inspected/reused
		certPEM := new(bytes.Buffer)
		pem.Encode(certPEM, &pem.Block{
			Type:  "CERTIFICATE",
			Bytes: cert.Leaf.Raw,
		})

		serverCertFile, err := os.Create(filepath.Join(tmpDir, certName+".crt"))
		if err != nil {
			t.Fatalf("creating file to write server certificate: %v", err)
		}
		if _, err := serverCertFile.Write(certPEM.Bytes()); err != nil {
			t.Fatalf("writing server certificate: %v", err)
		}

		if err := serverCertFile.Close(); err != nil {
			t.Fatalf("could not close certificate file: %s", err)
		}
	}

	t.Cleanup(func() {
		if t.Failed() {
			finalDir := filepath.Join(os.TempDir(), cleanStr(t.Name())+strconv.Itoa(rand.Int()))
			if err := os.Rename(tmpDir, finalDir); err != nil {
				t.Fatalf("could not rename directory with certificates: %s", err)
			}

			t.Logf("certificates persisted on: '%s'", finalDir)
		}
	})

	return certs
}

var cleanRegExp = regexp.MustCompile(`[^a-zA-Z0-9]`)

// cleanStr replaces all characters that do not match 'a-zA-Z0-9' by '_'
func cleanStr(path string) string {
	return cleanRegExp.ReplaceAllString(path, "_")
}
