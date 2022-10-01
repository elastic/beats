// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package lumberjack

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"testing"
	"time"
)

type Cert struct {
	signedCertDER []byte          // DER encoded certificate from x509.CreateCertificate.
	key           *rsa.PrivateKey // RSA public / private key pair.
}

// CertPEM returns the cert encoded as PEM.
func (c Cert) CertPEM(t testing.TB) []byte { return pemEncode(t, c.signedCertDER, "CERTIFICATE") }

// KeyPEM returns the private key encoded as PEM.
func (c Cert) KeyPEM(t testing.TB) []byte {
	return pemEncode(t, x509.MarshalPKCS1PrivateKey(c.key), "RSA PRIVATE KEY")
}

func (c Cert) TLSCertificate(t testing.TB) tls.Certificate {
	pair, err := tls.X509KeyPair(c.CertPEM(t), c.KeyPEM(t))
	if err != nil {
		t.Fatal(err)
	}

	return pair
}

// generateCertData creates a root CA, server, and client cert suitable for
// testing mTLS.
func generateCertData(t testing.TB) (rootCA, client, server Cert) {
	t.Helper()

	// CA cert
	ca := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization:  []string{"Elastic"},
			Country:       []string{"US"},
			Locality:      []string{"San Francisco"},
			StreetAddress: []string{"West El Camino Real"},
			PostalCode:    []string{"94040"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(0, 0, 1),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	var err error
	rootCA.key, err = rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		t.Fatal(err)
	}

	rootCA.signedCertDER, err = x509.CreateCertificate(rand.Reader, ca, ca, &rootCA.key.PublicKey, rootCA.key)
	if err != nil {
		t.Fatal(err)
	}

	// Server cert
	{
		// set up our server certificate
		serverCert := &x509.Certificate{
			SerialNumber: big.NewInt(2),
			Subject: pkix.Name{
				Organization:  []string{"Elastic"},
				Country:       []string{"US"},
				Locality:      []string{"San Francisco"},
				StreetAddress: []string{"West El Camino Real"},
				PostalCode:    []string{"94040"},
				CommonName:    "server",
			},
			IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
			DNSNames:     []string{"localhost"},
			NotBefore:    time.Now(),
			NotAfter:     time.Now().AddDate(0, 0, 1),
			SubjectKeyId: []byte{1, 2, 3, 4, 5},
			ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
			KeyUsage:     x509.KeyUsageDigitalSignature,
		}

		server.key, err = rsa.GenerateKey(rand.Reader, 4096)
		if err != nil {
			t.Fatal(err)
		}

		server.signedCertDER, err = x509.CreateCertificate(rand.Reader, serverCert, ca, &server.key.PublicKey, rootCA.key)
		if err != nil {
			t.Fatal(err)
		}
	}

	// Client cert.
	{
		clientCert := &x509.Certificate{
			SerialNumber: big.NewInt(3),
			Subject: pkix.Name{
				Organization:  []string{"Elastic"},
				Country:       []string{"US"},
				Locality:      []string{"San Francisco"},
				StreetAddress: []string{"West El Camino Real"},
				PostalCode:    []string{"94040"},
				CommonName:    "client",
			},
			NotBefore:      time.Now(),
			NotAfter:       time.Now().AddDate(0, 0, 1),
			SubjectKeyId:   []byte{1, 2, 3, 4, 5},
			ExtKeyUsage:    []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
			KeyUsage:       x509.KeyUsageDigitalSignature,
			EmailAddresses: []string{"client@example.com"},
		}

		client.key, err = rsa.GenerateKey(rand.Reader, 4096)
		if err != nil {
			t.Fatal(err)
		}

		client.signedCertDER, err = x509.CreateCertificate(rand.Reader, clientCert, ca, &client.key.PublicKey, rootCA.key)
		if err != nil {
			t.Fatal(err)
		}
	}

	return rootCA, client, server
}

func pemEncode(t testing.TB, certBytes []byte, certType string) []byte {
	t.Helper()

	pemData := new(bytes.Buffer)
	if err := pem.Encode(pemData, &pem.Block{Type: certType, Bytes: certBytes}); err != nil {
		t.Fatal(err)
	}

	return pemData.Bytes()
}
