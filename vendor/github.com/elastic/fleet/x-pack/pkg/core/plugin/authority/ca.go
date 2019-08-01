// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package authority

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"log"
	"math/big"
	"time"

	"github.com/pkg/errors"
)

// CertificateAuthority is an abstraction for common certificate authority
// unique for process
type CertificateAuthority struct {
	caCert     *x509.Certificate
	privateKey crypto.PrivateKey
	caPEM      []byte
}

// Pair is a x509 Key/Cert pair
type Pair struct {
	Crt []byte
	Key []byte
}

// NewCA creates a new certificate authority capable of generating child certificates
func NewCA() (*CertificateAuthority, error) {
	ca := &x509.Certificate{
		SerialNumber: big.NewInt(1653),
		Subject: pkix.Name{
			Organization: []string{"elastic-fleet"},
			CommonName:   "localhost",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	publicKey := &privateKey.PublicKey
	caBytes, err := x509.CreateCertificate(rand.Reader, ca, ca, publicKey, privateKey)
	if err != nil {
		log.Println("create ca failed", err)
		return nil, errors.Wrap(err, "ca creation failed")
	}

	var pubKeyBytes, privateKeyBytes []byte

	certOut := bytes.NewBuffer(pubKeyBytes)
	keyOut := bytes.NewBuffer(privateKeyBytes)

	// Public key
	err = pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: caBytes})
	if err != nil {
		return nil, errors.Wrap(err, "signing ca certificate")
	}

	// Private key
	err = pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)})
	if err != nil {
		return nil, errors.Wrap(err, "generating ca private key")
	}

	// prepare tls
	caPEM := certOut.Bytes()
	caTLS, err := tls.X509KeyPair(caPEM, keyOut.Bytes())
	if err != nil {
		return nil, errors.Wrap(err, "generating ca x509 pair")
	}

	caCert, err := x509.ParseCertificate(caTLS.Certificate[0])
	if err != nil {
		return nil, errors.Wrap(err, "generating ca private key")
	}

	return &CertificateAuthority{
		privateKey: caTLS.PrivateKey,
		caCert:     caCert,
		caPEM:      caPEM,
	}, nil
}

// GeneratePair generates child certificate
func (c *CertificateAuthority) GeneratePair() (*Pair, error) {
	// Prepare certificate
	certTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(1658),
		Subject: pkix.Name{
			Organization: []string{"elastic-fleet"},
			CommonName:   "localhost",
		},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().AddDate(10, 0, 0),
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:    x509.KeyUsageDigitalSignature,
	}
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	publicKey := &privateKey.PublicKey

	// Sign the certificate
	certBytes, err := x509.CreateCertificate(rand.Reader, certTemplate, c.caCert, publicKey, c.privateKey)
	if err != nil {
		return nil, errors.Wrap(err, "signing certificate")
	}

	var pubKeyBytes, privateKeyBytes []byte

	certOut := bytes.NewBuffer(pubKeyBytes)
	keyOut := bytes.NewBuffer(privateKeyBytes)

	// Public key
	err = pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certBytes})
	if err != nil {
		return nil, errors.Wrap(err, "generating public key")
	}

	// Private key
	err = pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)})
	if err != nil {
		return nil, errors.Wrap(err, "generating private key")
	}

	return &Pair{
		Crt: certOut.Bytes(),
		Key: keyOut.Bytes(),
	}, nil
}

// Crt returns crt cert of certificate authority
func (c *CertificateAuthority) Crt() []byte {
	return c.caPEM
}
