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
	cryptorand "crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"math/big"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// GetCertFingerPrint takes a certificate and returns its HEX encoded SHA-256
func GetCertFingerprint(cert *x509.Certificate) string {
	caSHA256 := sha256.Sum256(cert.Raw)
	return hex.EncodeToString(caSHA256[:])
}

func GenTestCerts(t *testing.T) map[string]*x509.Certificate {
	t.Helper()
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
		err = pem.Encode(certPEM, &pem.Block{
			Type:  "CERTIFICATE",
			Bytes: cert.Leaf.Raw,
		})
		require.NoErrorf(t, err, "failed to encode certificste to PEM")

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

	caKey, err := rsa.GenerateKey(cryptorand.Reader, 2048) // less secure key for quicker testing.
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("fail to generate RSA key: %w", err)
	}

	ca.SubjectKeyId = generateSubjectKeyID(&caKey.PublicKey)

	caBytes, err := x509.CreateCertificate(cryptorand.Reader, ca, ca, &caKey.PublicKey, caKey)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("fail to create certificate: %w", err)
	}

	leaf, err := x509.ParseCertificate(caBytes)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("fail to parse certificate: %w", err)
	}

	return tls.Certificate{
		Certificate: [][]byte{caBytes},
		PrivateKey:  caKey,
		Leaf:        leaf,
	}, nil
}

var cleanRegExp = regexp.MustCompile(`[^a-zA-Z0-9]`)

// cleanStr replaces all characters that do not match 'a-zA-Z0-9' by '_'
func cleanStr(path string) string {
	return cleanRegExp.ReplaceAllString(path, "_")
}

var ser int64 = 1

func serial() *big.Int {
	ser = ser + 1
	return big.NewInt(ser)
}

func generateSubjectKeyID(publicKey *rsa.PublicKey) []byte {
	// SubjectKeyId generated using method 1 in RFC 7093, Section 2:
	//   1) The keyIdentifier is composed of the leftmost 160-bits of the
	//   SHA-256 hash of the value of the BIT STRING subjectPublicKey
	//   (excluding the tag, length, and number of unused bits).
	publicKeyBytes := x509.MarshalPKCS1PublicKey(publicKey)
	h := sha256.Sum256(publicKeyBytes)
	return h[:20]
}

// genSignedCert generates a CA and KeyPair and remove the need to depends on code of agent.
func genSignedCert(
	ca tls.Certificate,
	keyUsage x509.KeyUsage,
	isCA bool,
	commonName string,
	dnsNames []string,
	ips []net.IP,
	expired bool,
) (tls.Certificate, error) {
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

	certKey, err := rsa.GenerateKey(cryptorand.Reader, 2048)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("fail to generate RSA key: %w", err)
	}

	if isCA {
		cert.SubjectKeyId = generateSubjectKeyID(&certKey.PublicKey)
	}

	certBytes, err := x509.CreateCertificate(
		cryptorand.Reader,
		cert,
		ca.Leaf,
		&certKey.PublicKey,
		ca.PrivateKey,
	)

	if err != nil {
		return tls.Certificate{}, fmt.Errorf("fail to create signed certificate: %w", err)
	}

	leaf, err := x509.ParseCertificate(certBytes)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("fail to parse the certificate: %w", err)
	}

	return tls.Certificate{
		Certificate: [][]byte{certBytes},
		PrivateKey:  certKey,
		Leaf:        leaf,
	}, nil
}
