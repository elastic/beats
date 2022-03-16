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
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMakeVerifyServerConnection(t *testing.T) {
	testCerts := openTestCerts(t)

	testCA, errs := LoadCertificateAuthorities([]string{
		filepath.Join("testdata", "ca.crt"),
		filepath.Join("testdata", "cacert.crt"),
	})
	if len(errs) > 0 {
		t.Fatalf("failed to load test certificate authorities: %+v", errs)
	}

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
			expectedError:    MissingPeerCertificate,
		},
		"default verification with certificates when required with expired cert": {
			verificationMode: VerifyFull,
			clientAuth:       tls.RequireAndVerifyClientCert,
			certAuthorities:  testCA,
			peerCerts:        []*x509.Certificate{testCerts["expired"]},
			serverName:       "",
			expectedCallback: true,
			expectedError:    x509.CertificateInvalidError{Cert: testCerts["expired"], Reason: x509.Expired},
		},
		"default verification with certificates when required with incorrect server name in cert": {
			verificationMode: VerifyFull,
			clientAuth:       tls.RequireAndVerifyClientCert,
			certAuthorities:  testCA,
			peerCerts:        []*x509.Certificate{testCerts["correct"]},
			serverName:       "bad.example.com",
			expectedCallback: true,
			expectedError:    x509.HostnameError{Certificate: testCerts["correct"], Host: "bad.example.com"},
		},
		"default verification with certificates when required with correct cert": {
			verificationMode: VerifyFull,
			clientAuth:       tls.RequireAndVerifyClientCert,
			certAuthorities:  testCA,
			peerCerts:        []*x509.Certificate{testCerts["correct"]},
			serverName:       "localhost",
			expectedCallback: true,
			expectedError:    nil,
		},
		"default verification with certificates when required with correct wildcard cert": {
			verificationMode: VerifyFull,
			clientAuth:       tls.RequireAndVerifyClientCert,
			certAuthorities:  testCA,
			peerCerts:        []*x509.Certificate{testCerts["wildcard"]},
			serverName:       "hello.example.com",
			expectedCallback: true,
			expectedError:    nil,
		},
		"certificate verification with certificates when required with correct cert": {
			verificationMode: VerifyCertificate,
			clientAuth:       tls.RequireAndVerifyClientCert,
			certAuthorities:  testCA,
			peerCerts:        []*x509.Certificate{testCerts["correct"]},
			serverName:       "localhost",
			expectedCallback: true,
			expectedError:    nil,
		},
		"certificate verification with certificates when required with expired cert": {
			verificationMode: VerifyCertificate,
			clientAuth:       tls.RequireAndVerifyClientCert,
			certAuthorities:  testCA,
			peerCerts:        []*x509.Certificate{testCerts["expired"]},
			serverName:       "localhost",
			expectedCallback: true,
			expectedError:    x509.CertificateInvalidError{Cert: testCerts["expired"], Reason: x509.Expired},
		},
		"certificate verification with certificates when required with incorrect server name in cert": {
			verificationMode: VerifyCertificate,
			clientAuth:       tls.RequireAndVerifyClientCert,
			certAuthorities:  testCA,
			peerCerts:        []*x509.Certificate{testCerts["correct"]},
			serverName:       "bad.example.com",
			expectedCallback: true,
			expectedError:    nil,
		},
		"strict verification with certificates when required with correct cert": {
			verificationMode: VerifyStrict,
			clientAuth:       tls.RequireAndVerifyClientCert,
			certAuthorities:  testCA,
			peerCerts:        []*x509.Certificate{testCerts["correct"]},
			serverName:       "localhost",
			expectedCallback: false,
			expectedError:    nil,
		},
		"default verification with certificates when required with cert signed by unkown authority": {
			verificationMode: VerifyFull,
			clientAuth:       tls.RequireAndVerifyClientCert,
			certAuthorities:  testCA,
			peerCerts:        []*x509.Certificate{testCerts["unknown authority"]},
			serverName:       "",
			expectedCallback: true,
			expectedError:    x509.UnknownAuthorityError{Cert: testCerts["unknown authority"]},
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

func openTestCerts(t testing.TB) map[string]*x509.Certificate {
	t.Helper()
	certs := make(map[string]*x509.Certificate, 0)

	for testcase, certname := range map[string]string{
		"expired":           "tls.crt",
		"unknown authority": "unsigned_tls.crt",
		"correct":           "client1.crt",
		"wildcard":          "server.crt",
		"es-leaf":           "es-leaf.crt",
		"es-root-ca":        "es-root-ca-cert.crt",
	} {

		certBytes, err := ioutil.ReadFile(filepath.Join("testdata", certname))
		if err != nil {
			t.Fatalf("reading file %q: %+v", certname, err)
		}
		block, _ := pem.Decode(certBytes)
		testCert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			t.Fatalf("parsing certificate %q: %+v", certname, err)
		}
		certs[testcase] = testCert
	}

	return certs
}

func TestTrustRootCA(t *testing.T) {
	certs := openTestCerts(t)

	nonEmptyCertPool := x509.NewCertPool()
	nonEmptyCertPool.AddCert(certs["wildcard"])
	nonEmptyCertPool.AddCert(certs["unknown authority"])

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
			caTrustedFingerprint: "e83171aa133b2b507e057fe091e296a7e58e9653c2b88d203b64a47eef6ec62b",
			peerCerts:            []*x509.Certificate{certs["es-leaf"], certs["es-root-ca"]},
			expectedRootCAsLen:   1,
		},
		{
			name:                 "RootCA cert doesn not matche the fingerprint and is not added to cfg.RootCAs",
			caTrustedFingerprint: "e83171aa133b2b507e057fe091e296a7e58e9653c2b88d203b64a47eef6ec62b",
			peerCerts:            []*x509.Certificate{certs["es-leaf"], certs["es-root-ca"]},
			expectedRootCAsLen:   0,
		},
		{
			name:                 "non empty CertPool has the RootCA added",
			rootCAs:              nonEmptyCertPool,
			caTrustedFingerprint: "e83171aa133b2b507e057fe091e296a7e58e9653c2b88d203b64a47eef6ec62b",
			peerCerts:            []*x509.Certificate{certs["es-leaf"], certs["es-root-ca"]},
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
	testCerts := openTestCerts(t)

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
			peerCerts:            []*x509.Certificate{testCerts["es-leaf"], testCerts["es-root-ca"]},
			serverName:           "localhost",
			expectedCallback:     true,
			CATrustedFingerprint: "e83171aa133b2b507e057fe091e296a7e58e9653c2b88d203b64a47eef6ec62b",
		},
		"CATrustedFingerprint and verification mode:VerifyCertificate": {
			verificationMode:     VerifyCertificate,
			peerCerts:            []*x509.Certificate{testCerts["es-leaf"], testCerts["es-root-ca"]},
			serverName:           "localhost",
			expectedCallback:     true,
			CATrustedFingerprint: "e83171aa133b2b507e057fe091e296a7e58e9653c2b88d203b64a47eef6ec62b",
		},
		"CATrustedFingerprint and verification mode:VerifyStrict": {
			verificationMode:     VerifyStrict,
			peerCerts:            []*x509.Certificate{testCerts["es-leaf"], testCerts["es-root-ca"]},
			serverName:           "localhost",
			expectedCallback:     true,
			CATrustedFingerprint: "e83171aa133b2b507e057fe091e296a7e58e9653c2b88d203b64a47eef6ec62b",
			CASHA256:             []string{Fingerprint(testCerts["es-leaf"])},
		},
		"CATrustedFingerprint and verification mode:VerifyNone": {
			verificationMode: VerifyNone,
			peerCerts:        []*x509.Certificate{testCerts["es-leaf"], testCerts["es-root-ca"]},
			serverName:       "localhost",
			expectedCallback: false,
		},
		"invalid CATrustedFingerprint and verification mode:VerifyFull returns error": {
			verificationMode:     VerifyFull,
			peerCerts:            []*x509.Certificate{testCerts["es-leaf"], testCerts["es-root-ca"]},
			serverName:           "localhost",
			expectedCallback:     true,
			CATrustedFingerprint: "INVALID HEX ENCODING",
			expectingError:       true,
		},
		"invalid CATrustedFingerprint and verification mode:VerifyCertificate returns error": {
			verificationMode:     VerifyCertificate,
			peerCerts:            []*x509.Certificate{testCerts["es-leaf"], testCerts["es-root-ca"]},
			serverName:           "localhost",
			expectedCallback:     true,
			CATrustedFingerprint: "INVALID HEX ENCODING",
			expectingError:       true,
		},
		"invalid CATrustedFingerprint and verification mode:VerifyStrict returns error": {
			verificationMode:     VerifyStrict,
			peerCerts:            []*x509.Certificate{testCerts["es-leaf"], testCerts["es-root-ca"]},
			serverName:           "localhost",
			expectedCallback:     true,
			CATrustedFingerprint: "INVALID HEX ENCODING",
			expectingError:       true,
			CASHA256:             []string{Fingerprint(testCerts["es-leaf"])},
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
