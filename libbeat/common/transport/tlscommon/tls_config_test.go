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
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io/ioutil"
	"log"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"testing"
	"time"

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

func TestVerificationMode(t *testing.T) {
	testcases := map[string]struct {
		verificationMode TLSVerificationMode
		serverName       string
		certHostname     string
		expectingError   bool
		ignoreCerts      bool
		emptySNA         bool
		legacyCN         bool
	}{
		"VerifyFull validates domain": {
			verificationMode: VerifyFull,
			serverName:       "localhost",
			certHostname:     "localhost",
		},
		"VerifyFull validates IPv4": {
			verificationMode: VerifyFull,
			serverName:       "127.0.0.1",
			certHostname:     "127.0.0.1",
		},
		"VerifyFull validates IPv6": {
			verificationMode: VerifyFull,
			serverName:       "::1",
			certHostname:     "::1",
		},
		"VerifyFull domain mismatch returns error": {
			verificationMode: VerifyFull,
			serverName:       "localhost",
			certHostname:     "elastic.co",
			expectingError:   true,
		},
		"VerifyFull IPv4 mismatch returns error": {
			verificationMode: VerifyFull,
			serverName:       "127.0.0.1",
			certHostname:     "1.2.3.4",
			expectingError:   true,
		},
		"VerifyFull IPv6 mismatch returns error": {
			verificationMode: VerifyFull,
			serverName:       "::1",
			certHostname:     "faca:b0de:baba::ca",
			expectingError:   true,
		},
		"VerifyFull does not return error when SNA is empty and legacy Common Name is used": {
			verificationMode: VerifyFull,
			serverName:       "localhost",
			certHostname:     "localhost",
			emptySNA:         true,
			legacyCN:         true,
			expectingError:   false,
		},
		"VerifyFull does not return error when SNA is empty and legacy Common Name is used with IP address": {
			verificationMode: VerifyFull,
			serverName:       "127.0.0.1",
			certHostname:     "127.0.0.1",
			emptySNA:         true,
			legacyCN:         true,
			expectingError:   false,
		},

		"VerifyStrict": {
			verificationMode: VerifyStrict,
			serverName:       "localhost",
			certHostname:     "localhost",
		},
		"VerifyStrict validates domain": {
			verificationMode: VerifyStrict,
			serverName:       "localhost",
			certHostname:     "localhost",
		},
		"VerifyStrict validates IPv4": {
			verificationMode: VerifyStrict,
			serverName:       "127.0.0.1",
			certHostname:     "127.0.0.1",
		},
		"VerifyStrict validates IPv6": {
			verificationMode: VerifyStrict,
			serverName:       "::1",
			certHostname:     "::1",
		},
		"VerifyStrict domain mismatch returns error": {
			verificationMode: VerifyStrict,
			serverName:       "127.0.0.1",
			certHostname:     "elastic.co",
			expectingError:   true,
		},
		"VerifyStrict IPv4 mismatch returns error": {
			verificationMode: VerifyStrict,
			serverName:       "127.0.0.1",
			certHostname:     "1.2.3.4",
			expectingError:   true,
		},
		"VerifyStrict IPv6 mismatch returns error": {
			verificationMode: VerifyStrict,
			serverName:       "::1",
			certHostname:     "faca:b0de:baba::ca",
			expectingError:   true,
		},
		"VerifyStrict return error when SNA is empty and legacy Common Name is used": {
			verificationMode: VerifyStrict,
			serverName:       "localhost",
			certHostname:     "localhost",
			emptySNA:         true,
			legacyCN:         true,
			expectingError:   true,
		},
		"VerifyStrict return error when SNA is empty and legacy Common Name is used with IP address": {
			verificationMode: VerifyStrict,
			serverName:       "127.0.0.1",
			certHostname:     "127.0.0.1",
			emptySNA:         true,
			legacyCN:         true,
			expectingError:   true,
		},
		"VerifyStrict returns error when SNA is empty": {
			verificationMode: VerifyStrict,
			serverName:       "localhost",
			certHostname:     "localhost",
			emptySNA:         true,
			expectingError:   true,
		},

		"VerifyCertificate does not validate domain": {
			verificationMode: VerifyCertificate,
			serverName:       "localhost",
			certHostname:     "elastic.co",
		},
		"VerifyCertificate does not validate IPv4": {
			verificationMode: VerifyCertificate,
			serverName:       "127.0.0.1",
			certHostname:     "elastic.co",
		},
		"VerifyCertificate does not validate IPv6": {
			verificationMode: VerifyCertificate,
			serverName:       "127.0.0.1",
			certHostname:     "faca:b0de:baba::ca",
		},

		"VerifyNone accepts untrusted certificates": {
			verificationMode: VerifyNone,
			serverName:       "127.0.0.1",
			certHostname:     "faca:b0de:baba::ca",
			ignoreCerts:      true,
		},
	}

	for name, test := range testcases {
		t.Run(name, func(t *testing.T) {
			serverURL, caCert := startTestServer(t, test.certHostname, test.emptySNA, test.legacyCN)
			certPool := x509.NewCertPool()
			certPool.AddCert(caCert)

			tlsC := TLSConfig{
				Verification: test.verificationMode,
				RootCAs:      certPool,
				ServerName:   test.serverName,
			}

			if test.ignoreCerts {
				tlsC.RootCAs = nil
				tlsC.ServerName = ""
			}

			client := http.Client{
				Transport: &http.Transport{
					TLSClientConfig: tlsC.BuildModuleClientConfig(test.serverName),
				},
			}

			resp, err := client.Get(serverURL.String())
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

// startTestServer starts a HTTP server for testing and returns it's certificates.
// If `address` is a hostname it will be added to the leaf certificate CN.
// Regardless of being a hostname or IP, `address` will be added to the correct
// SNA.
//
// New certificates are generated for each HTTP server, they use RSA 1024 bits, it
// is not the safest, but it's enough for tests.
// The HTTP server will shutdown at the end of the test.
func startTestServer(t *testing.T, address string, emptySNA, legacyCN bool) (url.URL, *x509.Certificate) {
	// Creates a listener on a random port selected by the OS
	l, err := net.Listen("tcp", "localhost:0")
	t.Cleanup(func() { l.Close() })

	// l.Addr().String() will return something like: 127.0.0.1:12345,
	// add the protocol and parse the URL
	serverURL, err := url.Parse("https://" + l.Addr().String())
	if err != nil {
		t.Fatal(err)
	}

	// Generate server ceritficates for the given address
	// and start the server
	caCert, serverCert := genVerifyCerts(t, address, emptySNA, legacyCN)
	server := http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("SSL test server"))
		}),
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{serverCert},
		},
	}
	t.Cleanup(func() { server.Close() })
	go server.ServeTLS(l, "", "")

	return *serverURL, caCert
}

func genVerifyCerts(t *testing.T, hostnameOrIP string, emptySNA, legacyCN bool) (*x509.Certificate, tls.Certificate) {
	t.Helper()

	hostname := ""
	ipAddress := net.ParseIP(hostnameOrIP)
	subjectCommonName := "You Know, for Search"

	if legacyCN {
		// Legacy behaviour of using the Common Name field to hold
		// a hostname or IP address
		subjectCommonName = hostnameOrIP
	}

	// We set either hostname or ipAddress
	if ipAddress == nil {
		hostname = hostnameOrIP
	}

	// ========================== Root CA Cert
	ca := &x509.Certificate{
		SerialNumber: big.NewInt(42),
		Subject: pkix.Name{
			Organization:  []string{"Root CA Corp"},
			Country:       []string{"DE"},
			Province:      []string{""},
			Locality:      []string{"Berlin"},
			StreetAddress: []string{"PostdamerPlatz"},
			PostalCode:    []string{"42"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(100, 0, 0), // 100 years validity
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	// ========================== Generate RootCA private Key
	caPrivKey, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		log.Panicf("generating RSA key for CA cert: %v", err)
	}

	// ========================== Generate RootCA Cert
	caBytes, err := x509.CreateCertificate(rand.Reader, ca, ca, &caPrivKey.PublicKey, caPrivKey)
	if err != nil {
		log.Panicf("generating CA certificate: %v", err)
	}

	caPEM := new(bytes.Buffer)
	pem.Encode(caPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caBytes,
	})

	// ========================== Generate Server Certificate (leaf)
	cert := &x509.Certificate{
		SerialNumber: big.NewInt(100),
		Subject: pkix.Name{
			Organization:  []string{"My Server Application Corp"},
			Country:       []string{"DE"},
			Province:      []string{""},
			Locality:      []string{"Berlin"},
			StreetAddress: []string{"AlexanderPlatz"},
			PostalCode:    []string{"100"},
			CommonName:    subjectCommonName,
		},

		// SNA - Subject Alternate Name we don't populate
		EmailAddresses: nil,
		URIs:           nil,

		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(10, 0, 0),
		SubjectKeyId: []byte{1, 2, 3, 4, 6},
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}

	// Set SNA based on what we got
	if !emptySNA {
		if hostname != "" {
			cert.DNSNames = []string{hostnameOrIP}
		}
		if ipAddress != nil {
			cert.IPAddresses = []net.IP{ipAddress}
		}
	}

	certPrivKey, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		log.Panicf("generating certificate private key: %v", err)
	}

	// =========================== Use CA to sign/generate the server (leaf) certificate
	certBytes, err := x509.CreateCertificate(rand.Reader, cert, ca, &certPrivKey.PublicKey, caPrivKey)
	if err != nil {
		log.Panicf("generating certificate: %v", err)
	}

	rootCACert, err := x509.ParseCertificate(caBytes)
	if err != nil {
		t.Fatalf("could not parse rootBytes into a certificate: %v", err)
	}

	certPEM := new(bytes.Buffer)
	pem.Encode(certPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})

	certPrivKeyPEM := new(bytes.Buffer)
	pem.Encode(certPrivKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(certPrivKey),
	})

	serverCert, err := tls.X509KeyPair(certPEM.Bytes(), certPrivKeyPEM.Bytes())
	if err != nil {
		t.Fatalf("could not convert server certificate to tls.Certificate: %v", err)
	}

	return rootCACert, serverCert
}
