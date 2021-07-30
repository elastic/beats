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
)

func TestMakeVerifyServerConnection(t *testing.T) {
	testCerts, err := openTestCerts()
	if err != nil {
		t.Fatalf("failed to open test certs: %+v", err)
	}

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
			test := test
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
				assert.Nil(t, err)
			} else {
				assert.Error(t, test.expectedError, err)
			}
		})
	}

}

func openTestCerts() (map[string]*x509.Certificate, error) {
	certs := make(map[string]*x509.Certificate, 0)

	for testcase, certname := range map[string]string{
		"expired":           "tls.crt",
		"unknown authority": "unsigned_tls.crt",
		"correct":           "client1.crt",
		"wildcard":          "server.crt",
	} {

		certBytes, err := ioutil.ReadFile(filepath.Join("testdata", certname))
		if err != nil {
			return nil, err
		}
		block, _ := pem.Decode(certBytes)
		testCert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, err
		}
		certs[testcase] = testCert
	}

	return certs, nil
}
