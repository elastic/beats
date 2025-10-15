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
	"crypto/sha256"
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

type configs struct {
	cnPrefix   string
	dnsNames   []string
	clientCert bool
}

type Option func(opt *configs)

// WithClientCert generates a client certificate, without any IP or SAN/DNS.
// It overrides any other IP or name set by other means.
func WithClientCert(clientCert bool) Option {
	return func(opt *configs) {
		opt.clientCert = clientCert
	}
}

// WithCNPrefix adds cnPrefix as prefix for the CN.
func WithCNPrefix(cnPrefix string) Option {
	return func(opt *configs) {
		opt.cnPrefix = cnPrefix
	}
}

// WithDNSNames adds dnsNames to the DNSNames.
func WithDNSNames(dnsNames ...string) Option {
	return func(opt *configs) {
		opt.dnsNames = dnsNames
	}
}

// NewRootCA generates a new x509 Certificate using ECDSA P-384 and returns:
// - the private key
// - the certificate
// - the certificate and its key in PEM format as a byte slice.
//
// If any error occurs during the generation process, a non-nil error is returned.
func NewRootCA(opts ...Option) (crypto.PrivateKey, *x509.Certificate, Pair, error) {
	rootKey, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		return nil, nil, Pair{}, fmt.Errorf("could not create private key: %w", err)
	}

	cert, pair, err := newRootCert(rootKey, &rootKey.PublicKey, opts...)
	return rootKey, cert, pair, err
}

// NewRSARootCA generates a new x509 Certificate using RSA with a 2048-bit key and returns:
// - the private key
// - the certificate
// - the certificate and its key in PEM format as a byte slice.
//
// If any error occurs during the generation process, a non-nil error is returned.
func NewRSARootCA(opts ...Option) (crypto.PrivateKey, *x509.Certificate, Pair, error) {
	rootKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, Pair{}, fmt.Errorf("could not create private key: %w", err)
	}
	cert, pair, err := newRootCert(rootKey, &rootKey.PublicKey, opts...)
	return rootKey, cert, pair, err
}

// GenerateChildCert generates a ECDSA (P-384) x509 Certificate as a child of
// caCert and returns the following:
// - the certificate and private key as a tls.Certificate
// - a Pair with the certificate and its key im PEM format
//
// If any error occurs during the generation process, a non-nil error is returned.
func GenerateChildCert(name string, ips []net.IP, caPrivKey crypto.PrivateKey, caCert *x509.Certificate, opts ...Option) (*tls.Certificate, Pair, error) {
	priv, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		return nil, Pair{}, fmt.Errorf("could not create ECDSA private key: %w", err)
	}

	cert, childPair, err :=
		GenerateGenericChildCert(
			name,
			ips,
			priv,
			&priv.PublicKey,
			caPrivKey,
			caCert,
			opts...)
	if err != nil {
		return nil, Pair{}, fmt.Errorf(
			"could not generate child TLS certificate CA: %w", err)
	}

	return cert, childPair, nil
}

// GenerateRSAChildCert generates a RSA with a 2048-bit key x509 Certificate as a
// child of caCert and returns the following:
// - the certificate and private key as a tls.Certificate
// - a Pair with the certificate and its key im PEM format
//
// If any error occurs during the generation process, a non-nil error is returned.
func GenerateRSAChildCert(name string, ips []net.IP, caPrivKey crypto.PrivateKey, caCert *x509.Certificate, opts ...Option) (*tls.Certificate, Pair, error) {
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
			caCert,
			opts...)
	if err != nil {
		return nil, Pair{}, fmt.Errorf(
			"could not generate child TLS certificate: %w", err)
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
	caCert *x509.Certificate,
	opts ...Option) (*tls.Certificate, Pair, error) {

	cfg := getCgf(opts)

	cn := "Police Public Call Box"
	if cfg.cnPrefix != "" {
		cn = fmt.Sprintf("[%s] %s", cfg.cnPrefix, cn)
	}

	var dnsNames []string
	if name != "" {
		dnsNames = append(cfg.dnsNames, name)
	}
	notBefore, notAfter := makeNotBeforeAndAfter()
	certTemplate := &x509.Certificate{
		DNSNames:     dnsNames,
		IPAddresses:  ips,
		SerialNumber: big.NewInt(1658),
		Subject: pkix.Name{
			Locality:     []string{"anywhere in time and space"},
			Organization: []string{"TARDIS"},
			CommonName:   cn,
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,
		KeyUsage: x509.KeyUsageDigitalSignature |
			x509.KeyUsageKeyEncipherment |
			x509.KeyUsageKeyAgreement,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
	}

	if cfg.clientCert {
		certTemplate.IPAddresses = nil
		certTemplate.DNSNames = nil
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

// EncryptKey accepts a *ecdsa.PrivateKey or *rsa.PrivateKey, it encrypts it
// and returns the encrypted key in PEM format.
func EncryptKey(key crypto.PrivateKey, passphrase string) ([]byte, error) {
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("error converting private key to DER: %w", err)
	}

	var blockType string
	switch key.(type) {
	case *rsa.PrivateKey:
		blockType = "RSA PRIVATE KEY"
	case *ecdsa.PrivateKey:
		blockType = "EC PRIVATE KEY"
	default:
		return nil, fmt.Errorf("unsupported private key type: %T", key)
	}

	encPem, err := x509.EncryptPEMBlock( //nolint:staticcheck // we need to drop support for this, but while we don't, it needs to be tested.
		rand.Reader,
		blockType,
		keyDER,
		[]byte(passphrase),
		x509.PEMCipherAES128)
	if err != nil {
		return nil, fmt.Errorf("failed encrypting certificate key: %w", err)
	}

	certKeyEnc := pem.EncodeToMemory(encPem)
	return certKeyEnc, nil
}

// newRootCert creates a new self-signed root certificate using the provided
// private key and public key.
// It returns:
//   - the private key,
//   - the certificate,
//   - a Pair containing the certificate and private key in PEM format.
//
// If an error occurs during certificate creation, it returns a non-nil error.
func newRootCert(priv crypto.PrivateKey, pub crypto.PublicKey, opts ...Option) (*x509.Certificate, Pair, error) {
	cn := "High Council"
	cfg := getCgf(opts)
	if cfg.cnPrefix != "" {
		cn = fmt.Sprintf("[%s] %s", cfg.cnPrefix, cn)
	}
	notBefore, notAfter := makeNotBeforeAndAfter()

	rootTemplate := x509.Certificate{
		SerialNumber: big.NewInt(1653),
		Subject: pkix.Name{
			Country:            []string{"Gallifrey"},
			Locality:           []string{"The Capitol"},
			OrganizationalUnit: []string{"Time Lords"},
			Organization:       []string{"High Council of the Time Lords"},
			CommonName:         cn,
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
		SubjectKeyId:          generateSubjectKeyID(pub),
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

func getCgf(opts []Option) configs {
	cfg := configs{dnsNames: []string{}}
	for _, opt := range opts {
		opt(&cfg)
	}
	return cfg
}

func generateSubjectKeyID(pub crypto.PublicKey) []byte {
	// SubjectKeyId generated using method 1 in RFC 7093, Section 2:
	//   1) The keyIdentifier is composed of the leftmost 160-bits of the
	//   SHA-256 hash of the value of the BIT STRING subjectPublicKey
	//   (excluding the tag, length, and number of unused bits).
	var publicKeyBytes []byte
	switch publicKey := pub.(type) {
	case *rsa.PublicKey:
		publicKeyBytes = x509.MarshalPKCS1PublicKey(publicKey)
	case *ecdsa.PublicKey:
		//nolint:staticcheck // no alternative
		publicKeyBytes = elliptic.Marshal(publicKey.Curve, publicKey.X, publicKey.Y)
	}
	h := sha256.Sum256(publicKeyBytes)
	return h[:20]
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
	notAfter := now.Add(30 * 24 * time.Hour)
	return notBefore, notAfter
}
