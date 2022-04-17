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
	"io/ioutil"
	"math/big"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/menderesk/beats/v7/libbeat/common"
)

var ser int64 = 1

func TestCAPinning(t *testing.T) {
	host := "127.0.0.1"

	t.Run("when the ca_sha256 field is not defined we use normal certificate validation", func(t *testing.T) {
		cfg := common.MustNewConfigFrom(map[string]interface{}{
			"verification_mode":       "strict",
			"certificate_authorities": []string{"ca_test.pem"},
		})

		config := &Config{}
		err := cfg.Unpack(config)
		require.NoError(t, err)

		tlsCfg, err := LoadTLSConfig(config)
		require.NoError(t, err)

		tls := tlsCfg.BuildModuleClientConfig(host)
		require.Nil(t, tls.VerifyConnection)
	})

	t.Run("when the ca_sha256 field is defined we use CA cert pinning", func(t *testing.T) {
		cfg := common.MustNewConfigFrom(map[string]interface{}{
			"ca_sha256": "hello",
		})

		config := &Config{}
		err := cfg.Unpack(config)
		require.NoError(t, err)

		tlsCfg, err := LoadTLSConfig(config)
		require.NoError(t, err)

		tls := tlsCfg.BuildModuleClientConfig(host)
		require.NotNil(t, tls.VerifyConnection)
	})

	t.Run("CA Root -> Certificate and we have the CA root pin", func(t *testing.T) {
		verificationModes := []TLSVerificationMode{
			VerifyFull,
			VerifyStrict,
			VerifyCertificate,
		}
		for _, mode := range verificationModes {
			t.Run(mode.String(), func(t *testing.T) {
				msg := []byte("OK received message")

				ca, err := genCA()
				require.NoError(t, err)

				serverCert, err := genSignedCert(ca, x509.KeyUsageDigitalSignature, false)
				require.NoError(t, err)

				mux := http.NewServeMux()
				mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					w.Write(msg)
				})

				// Select a random available port from the OS.
				addr := "localhost:0"

				l, err := net.Listen("tcp", addr)

				server := &http.Server{
					Handler: mux,
					TLSConfig: &tls.Config{
						Certificates: []tls.Certificate{
							serverCert,
						},
					},
				}

				// Start server and shut it down when the tests are over.
				go server.ServeTLS(l, "", "")
				defer l.Close()

				// Root CA Pool
				require.NoError(t, err)
				rootCAs := x509.NewCertPool()
				rootCAs.AddCert(ca.Leaf)

				// Get the pin of the RootCA.
				pin := Fingerprint(ca.Leaf)

				tlsC := &TLSConfig{
					Verification: mode,
					RootCAs:      rootCAs,
					CASha256:     []string{pin},
				}

				config := tlsC.BuildModuleClientConfig("localhost")
				hostToConnect := l.Addr().String()

				transport := &http.Transport{
					TLSClientConfig: config,
				}

				client := &http.Client{Transport: transport}

				port := strings.TrimPrefix(hostToConnect, "127.0.0.1:")

				req, err := http.NewRequest("GET", "https://localhost:"+port, nil)
				require.NoError(t, err)
				resp, err := client.Do(req)
				require.NoError(t, err)
				content, err := ioutil.ReadAll(resp.Body)
				require.NoError(t, err)

				assert.True(t, bytes.Equal(msg, content))

				// 1. create key-pair
				// 2. create pin
				// 3. start server
				// 4. Connect
				// 5. Check wrong key do not work
				// 6. Check good key work
				// 7. check plain text fails to work.
			})
		}
	})

	t.Run("CA Root -> Intermediate -> Certificate and we receive the CA Root Pin", func(t *testing.T) {
		msg := []byte("OK received message")

		ca, err := genCA()
		require.NoError(t, err)

		intermediate, err := genSignedCert(ca, x509.KeyUsageDigitalSignature|x509.KeyUsageCertSign, true)
		require.NoError(t, err)

		serverCert, err := genSignedCert(intermediate, x509.KeyUsageDigitalSignature, false)
		require.NoError(t, err)

		mux := http.NewServeMux()
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write(msg)
		})

		// Select a random available port from the OS.
		addr := "localhost:0"

		l, err := net.Listen("tcp", addr)
		require.NoError(t, err)

		// Server needs to provides the chain of trust, so server certificate + intermediate.
		// RootCAs will trust the intermediate, intermediate will trust the server.
		serverCert.Certificate = append(serverCert.Certificate, intermediate.Certificate...)

		server := &http.Server{
			Handler: mux,
			TLSConfig: &tls.Config{
				Certificates: []tls.Certificate{
					serverCert,
				},
			},
		}

		// Start server and shut it down when the tests are over.
		go server.ServeTLS(l, "", "")
		defer l.Close()

		// Root CA Pool
		rootCAs := x509.NewCertPool()
		rootCAs.AddCert(ca.Leaf)

		// Get the pin of the RootCA.
		pin := Fingerprint(ca.Leaf)

		tlsC := &TLSConfig{
			RootCAs:  rootCAs,
			CASha256: []string{pin},
		}

		config := tlsC.BuildModuleClientConfig("localhost")
		hostToConnect := l.Addr().String()

		transport := &http.Transport{
			TLSClientConfig: config,
		}

		client := &http.Client{Transport: transport}

		port := strings.TrimPrefix(hostToConnect, "127.0.0.1:")

		req, err := http.NewRequest("GET", "https://localhost:"+port, nil)
		require.NoError(t, err)
		resp, err := client.Do(req)
		require.NoError(t, err)
		content, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)

		assert.True(t, bytes.Equal(msg, content))
	})

	t.Run("When we have the wrong pin we refuse to connect", func(t *testing.T) {
		msg := []byte("OK received message")

		ca, err := genCA()
		require.NoError(t, err)

		intermediate, err := genSignedCert(ca, x509.KeyUsageDigitalSignature|x509.KeyUsageCertSign, true)
		require.NoError(t, err)

		serverCert, err := genSignedCert(intermediate, x509.KeyUsageDigitalSignature, false)
		require.NoError(t, err)

		mux := http.NewServeMux()
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write(msg)
		})

		// Select a random available port from the OS.
		addr := "localhost:0"

		l, err := net.Listen("tcp", addr)
		require.NoError(t, err)

		// Server needs to provides the chain of trust, so server certificate + intermediate.
		// RootCAs will trust the intermediate, intermediate will trust the server.
		serverCert.Certificate = append(serverCert.Certificate, intermediate.Certificate...)

		server := &http.Server{
			Handler: mux,
			TLSConfig: &tls.Config{
				Certificates: []tls.Certificate{
					serverCert,
				},
			},
		}

		// Start server and shut it down when the tests are over.
		go server.ServeTLS(l, "", "")
		defer l.Close()

		// Root CA Pool
		rootCAs := x509.NewCertPool()
		rootCAs.AddCert(ca.Leaf)

		// Get the pin of the RootCA.
		pin := "wrong-pin"

		tlsC := &TLSConfig{
			RootCAs:  rootCAs,
			CASha256: []string{pin},
		}

		config := tlsC.BuildModuleClientConfig("localhost")
		hostToConnect := l.Addr().String()

		transport := &http.Transport{
			TLSClientConfig: config,
		}

		client := &http.Client{Transport: transport}

		port := strings.TrimPrefix(hostToConnect, "127.0.0.1:")

		req, err := http.NewRequest("GET", "https://localhost:"+port, nil)
		require.NoError(t, err)
		_, err = client.Do(req)
		require.Error(t, err)
	})
}

func genCA() (tls.Certificate, error) {
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
	if err != nil {
		return tls.Certificate{}, errors.Wrap(err, "fail to generate RSA key")
	}

	caBytes, err := x509.CreateCertificate(rand.Reader, ca, ca, &caKey.PublicKey, caKey)
	if err != nil {
		return tls.Certificate{}, errors.Wrap(err, "fail to create certificate")
	}

	leaf, err := x509.ParseCertificate(caBytes)
	if err != nil {
		return tls.Certificate{}, errors.Wrap(err, "fail to parse certificate")
	}

	return tls.Certificate{
		Certificate: [][]byte{caBytes},
		PrivateKey:  caKey,
		Leaf:        leaf,
	}, nil
}

// genSignedCert generates a CA and KeyPair and remove the need to depends on code of agent.
func genSignedCert(ca tls.Certificate, keyUsage x509.KeyUsage, isCA bool) (tls.Certificate, error) {
	// Create another Cert/key
	cert := &x509.Certificate{
		DNSNames:     []string{"localhost"},
		SerialNumber: big.NewInt(2000),
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
		IsCA:                  isCA,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              keyUsage,
		BasicConstraintsValid: true,
	}

	certKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return tls.Certificate{}, errors.Wrap(err, "fail to generate RSA key")
	}

	certBytes, err := x509.CreateCertificate(
		rand.Reader,
		cert,
		ca.Leaf,
		&certKey.PublicKey,
		ca.PrivateKey,
	)

	if err != nil {
		return tls.Certificate{}, errors.Wrap(err, "fail to create signed certificate")
	}

	leaf, err := x509.ParseCertificate(certBytes)
	if err != nil {
		return tls.Certificate{}, errors.Wrap(err, "fail to parse the certificate")
	}

	return tls.Certificate{
		Certificate: [][]byte{certBytes},
		PrivateKey:  certKey,
		Leaf:        leaf,
	}, nil
}

func serial() *big.Int {
	ser = ser + 1
	return big.NewInt(ser)
}
