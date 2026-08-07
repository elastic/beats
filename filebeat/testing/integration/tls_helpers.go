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

package integration

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// serialMax is the exclusive upper bound for random certificate serial numbers
// (2^128), allocated once and shared across all calls to randomSerial.
var serialMax = new(big.Int).Lsh(big.NewInt(1), 128)

// TestCA is an in-test generated certificate authority used to issue TLS leaf
// certificates for integration tests.
type TestCA struct {
	// CertFile is the path to the CA certificate PEM file.
	CertFile string

	cert *x509.Certificate
	key  *ecdsa.PrivateKey
}

// GenerateTestCA creates a new self-signed CA and writes its certificate to a
// temp file owned by t. The private key is retained in memory for signing.
func GenerateTestCA(t *testing.T) *TestCA {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("GenerateTestCA: generate key: %v", err)
	}

	serial := randomSerial(t)
	tmpl := &x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: "test-ca"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
	}

	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("GenerateTestCA: create certificate: %v", err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("GenerateTestCA: parse certificate: %v", err)
	}

	certFile := writePEMFile(t, t.TempDir(), "ca.crt", "CERTIFICATE", der)

	return &TestCA{CertFile: certFile, cert: cert, key: key}
}

// Issue generates a leaf certificate signed by this CA. The leaf cert has SANs
// for 127.0.0.1 and localhost and supports both server and client auth.
// Returns paths to the PEM certificate and private key files.
func (ca *TestCA) Issue(t *testing.T) (certFile, keyFile string) {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("TestCA.Issue: generate key: %v", err)
	}

	tmpl := &x509.Certificate{
		SerialNumber: randomSerial(t),
		Subject:      pkix.Name{CommonName: "test-leaf"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
		DNSNames:     []string{"localhost"},
	}

	der, err := x509.CreateCertificate(rand.Reader, tmpl, ca.cert, &key.PublicKey, ca.key)
	if err != nil {
		t.Fatalf("TestCA.Issue: create certificate: %v", err)
	}

	// Both files share one temp directory to avoid creating two cleanup entries.
	dir := t.TempDir()
	certFile = writePEMFile(t, dir, "cert.crt", "CERTIFICATE", der)

	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatalf("TestCA.Issue: marshal key: %v", err)
	}
	keyFile = writePEMFile(t, dir, "key.pem", "EC PRIVATE KEY", keyDER)

	return certFile, keyFile
}

func writePEMFile(t *testing.T, dir, name, pemType string, der []byte) string {
	t.Helper()
	p := filepath.Join(dir, name)
	data := pem.EncodeToMemory(&pem.Block{Type: pemType, Bytes: der})
	if err := os.WriteFile(p, data, 0600); err != nil {
		t.Fatalf("writePEMFile %s: %v", name, err)
	}
	return p
}

func randomSerial(t *testing.T) *big.Int {
	t.Helper()
	n, err := rand.Int(rand.Reader, serialMax)
	if err != nil {
		t.Fatalf("randomSerial: %v", err)
	}
	return n
}
