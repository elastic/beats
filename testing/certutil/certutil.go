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

package certutil

import (
	"bytes"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"time"
)

// Pair is a certificate and its private key in PEM format.
type Pair struct {
	Cert []byte
	Key  []byte
}

// NewRootCA generates a new x509 Certificate using ECDSA P-384 and returns:
// - the private key
// - the certificate
// - the certificate and its key in PEM format as a byte slice.
//
// If any error occurs during the generation process, a non-nil error is returned.
func NewRootCA() (crypto.PrivateKey, *x509.Certificate, Pair, error) {
	rootKey, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		return nil, nil, Pair{}, fmt.Errorf("could not create private key: %w", err)
	}

	cert, pair, err := newRootCert(rootKey, &rootKey.PublicKey)
	return rootKey, cert, pair, err
}

// NewRSARootCA generates a new x509 Certificate using RSA with a 2048-bit key and returns:
// - the private key
// - the certificate
// - the certificate and its key in PEM format as a byte slice.
//
// If any error occurs during the generation process, a non-nil error is returned.
func NewRSARootCA() (crypto.PrivateKey, *x509.Certificate, Pair, error) {
	rootKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, Pair{}, fmt.Errorf("could not create private key: %w", err)
	}
	cert, pair, err := newRootCert(rootKey, &rootKey.PublicKey)
	return rootKey, cert, pair, err
}

// GenerateChildCert generates a ECDSA (P-384) x509 Certificate as a child of
// caCert and returns the following:
// - the certificate and private key as a tls.Certificate
// - a Pair with the certificate and its key im PEM format
//
// If any error occurs during the generation process, a non-nil error is returned.
func GenerateChildCert(name string, ips []net.IP, caPrivKey crypto.PrivateKey, caCert *x509.Certificate) (*tls.Certificate, Pair, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, Pair{}, fmt.Errorf("could not create RSA private key: %w", err)
	}

	cert, childPair, err :=
		GenerateGenericChildCert(
			name,
			ips,
			priv,
			&priv.PublicKey,
			caPrivKey,
			caCert)
	if err != nil {
		return nil, Pair{}, fmt.Errorf(
			"could not generate child TLS certificate CA: %w", err)
	}

	return cert, childPair, nil
}

// GenerateGenericChildCert generates a x509 Certificate using priv and pub
// as the certificate's private and public keys and as a child of caCert.
// Use this function if you need fine control over keys or ips and certificate name,
// otherwise prefer GenerateChildCert or NewRootAndChildCerts/NewRSARootAndChildCerts
//
// It returns the following:
// - the certificate and private key as a tls.Certificate
// - a Pair with the certificate and its key im PEM format
//
// If any error occurs during the generation process, a non-nil error is returned.
func GenerateGenericChildCert(
	name string,
	ips []net.IP,
	priv crypto.PrivateKey,
	pub crypto.PublicKey,
	caPrivKey crypto.PrivateKey,
	caCert *x509.Certificate) (*tls.Certificate, Pair, error) {

	notBefore, notAfter := makeNotBeforeAndAfter()

	certTemplate := &x509.Certificate{
		DNSNames:     []string{name},
		IPAddresses:  ips,
		SerialNumber: big.NewInt(1658),
		Subject: pkix.Name{
			Locality:     []string{"anywhere in time and space"},
			Organization: []string{"TARDIS"},
			CommonName:   "Police Public Call Box",
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,
		KeyUsage:  x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
	}

	certRawBytes, err := x509.CreateCertificate(
		rand.Reader, certTemplate, caCert, pub, caPrivKey)
	if err != nil {
		return nil, Pair{}, fmt.Errorf("could not create CA: %w", err)
	}

	privateKeyDER, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return nil, Pair{}, fmt.Errorf("could not marshal private key: %w", err)
	}

	// PEM private key
	var privBytesOut []byte
	privateKeyBuff := bytes.NewBuffer(privBytesOut)
	err = pem.Encode(privateKeyBuff,
		&pem.Block{Type: keyBlockType(priv), Bytes: privateKeyDER})
	if err != nil {
		return nil, Pair{}, fmt.Errorf("could not pem.Encode private key: %w", err)
	}
	privateKeyPemBytes := privateKeyBuff.Bytes()

	// PEM certificate
	var certBytesOut []byte
	certBuff := bytes.NewBuffer(certBytesOut)
	err = pem.Encode(certBuff, &pem.Block{
		Type: "CERTIFICATE", Bytes: certRawBytes})
	if err != nil {
		return nil, Pair{}, fmt.Errorf("could not pem.Encode certificate: %w", err)
	}
	certPemBytes := certBuff.Bytes()

	// TLS Certificate
	tlsCert, err := tls.X509KeyPair(certPemBytes, privateKeyPemBytes)
	if err != nil {
		return nil, Pair{}, fmt.Errorf("could not create key pair: %w", err)
	}

	return &tlsCert, Pair{
		Cert: certPemBytes,
		Key:  privateKeyPemBytes,
	}, nil
}

// NewRootAndChildCerts returns an ECDSA (P-384) root CA and a child certificate
// and their keys for "localhost" and "127.0.0.1".
func NewRootAndChildCerts() (Pair, Pair, error) {
	rootKey, rootCACert, rootPair, err := NewRootCA()
	if err != nil {
		return Pair{}, Pair{}, fmt.Errorf("could not generate root CA: %w", err)
	}

	priv, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		return Pair{}, Pair{}, fmt.Errorf("could not create private key: %w", err)
	}

	childPair, err := defaultChildCert(rootKey, priv, &priv.PublicKey, rootCACert)
	return rootPair, childPair, err
}

// NewRSARootAndChildCerts returns an RSA (2048-bit) root CA and a child
// certificate and their keys for "localhost" and "127.0.0.1".
func NewRSARootAndChildCerts() (Pair, Pair, error) {
	rootKey, rootCACert, rootPair, err := NewRSARootCA()
	if err != nil {
		return Pair{}, Pair{}, fmt.Errorf("could not generate RSA root CA: %w", err)
	}

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return Pair{}, Pair{}, fmt.Errorf("could not create RSA private key: %w", err)
	}

	childPair, err := defaultChildCert(rootKey, priv, &priv.PublicKey, rootCACert)
	return rootPair, childPair, err
}

// newRootCert creates a new self-signed root certificate using the provided
// private key and public key.
// It returns:
//   - the private key,
//   - the certificate,
//   - a Pair containing the certificate and private key in PEM format.
//
// If an error occurs during certificate creation, it returns a non-nil error.
func newRootCert(priv crypto.PrivateKey, pub crypto.PublicKey) (*x509.Certificate, Pair, error) {
	notBefore, notAfter := makeNotBeforeAndAfter()

	rootTemplate := x509.Certificate{
		SerialNumber: big.NewInt(1653),
		Subject: pkix.Name{
			Country:            []string{"Gallifrey"},
			Locality:           []string{"The Capitol"},
			OrganizationalUnit: []string{"Time Lords"},
			Organization:       []string{"High Council of the Time Lords"},
			CommonName:         "High Council",
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	rootCertRawBytes, err := x509.CreateCertificate(
		rand.Reader, &rootTemplate, &rootTemplate, pub, priv)
	if err != nil {
		return nil, Pair{}, fmt.Errorf("could not create CA: %w", err)
	}

	rootPrivKeyDER, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return nil, Pair{}, fmt.Errorf("could not marshal private key: %w", err)
	}

	// PEM private key
	var rootPrivBytesOut []byte
	rootPrivateKeyBuff := bytes.NewBuffer(rootPrivBytesOut)
	err = pem.Encode(rootPrivateKeyBuff,
		&pem.Block{Type: keyBlockType(priv), Bytes: rootPrivKeyDER})
	if err != nil {
		return nil, Pair{}, fmt.Errorf("could not pem.Encode private key: %w", err)
	}

	// PEM certificate
	var rootCertBytesOut []byte
	rootCertPemBuff := bytes.NewBuffer(rootCertBytesOut)
	err = pem.Encode(rootCertPemBuff,
		&pem.Block{Type: "CERTIFICATE", Bytes: rootCertRawBytes})
	if err != nil {
		return nil, Pair{}, fmt.Errorf("could not pem.Encode certificate: %w", err)
	}

	// tls.Certificate
	rootTLSCert, err := tls.X509KeyPair(
		rootCertPemBuff.Bytes(), rootPrivateKeyBuff.Bytes())
	if err != nil {
		return nil, Pair{}, fmt.Errorf("could not create key pair: %w", err)
	}

	rootCACert, err := x509.ParseCertificate(rootTLSCert.Certificate[0])
	if err != nil {
		return nil, Pair{}, fmt.Errorf("could not parse certificate: %w", err)
	}

	return rootCACert, Pair{
		Cert: rootCertPemBuff.Bytes(),
		Key:  rootPrivateKeyBuff.Bytes(),
	}, nil
}

// defaultChildCert generates a child certificate for localhost and 127.0.0.1.
// It returns the certificate and its key as a Pair and an error if any happens.
func defaultChildCert(
	rootPriv,
	priv crypto.PrivateKey,
	pub crypto.PublicKey,
	rootCACert *x509.Certificate) (Pair, error) {
	_, childPair, err :=
		GenerateGenericChildCert(
			"localhost",
			[]net.IP{net.ParseIP("127.0.0.1")},
			priv,
			pub,
			rootPriv,
			rootCACert)
	if err != nil {
		return Pair{}, fmt.Errorf(
			"could not generate child TLS certificate CA: %w", err)
	}
	return childPair, nil
}

// keyBlockType returns the correct PEM block type for the given private key.
func keyBlockType(priv crypto.PrivateKey) string {
	switch priv.(type) {
	case *rsa.PrivateKey:
		return "RSA PRIVATE KEY"
	case *ecdsa.PrivateKey:
		return "EC PRIVATE KEY"
	default:
		panic(fmt.Errorf("unsupported private key type: %T", priv))
	}
}

// makeNotBeforeAndAfter returns:
//   - notBefore: 1 minute before now
//   - notAfter: 7 days after now
func makeNotBeforeAndAfter() (time.Time, time.Time) {
	now := time.Now()
	notBefore := now.Add(-1 * time.Minute)
	notAfter := now.Add(7 * 24 * time.Hour)
	return notBefore, notAfter
}
