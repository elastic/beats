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
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/youmark/pkcs8"

	"github.com/elastic/elastic-agent-libs/logp"
)

const logSelector = "tls"

// LoadCertificate will load a certificate from disk and return a tls.Certificate or error
func LoadCertificate(config *CertificateConfig) (*tls.Certificate, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	certificate := config.Certificate
	key := config.Key
	if certificate == "" {
		return nil, nil
	}

	log := logp.NewLogger(logSelector)
	passphrase := config.Passphrase
	if passphrase == "" && config.PassphrasePath != "" {
		p, err := os.ReadFile(config.PassphrasePath)
		if err != nil {
			return nil, fmt.Errorf("unable to read passphrase_file: %w", err)
		}
		passphrase = string(p)
	}

	certPEM, err := ReadPEMFile(log, certificate, passphrase)
	if err != nil {
		log.Errorf("Failed reading certificate file %v: %+v", certificate, err)
		return nil, fmt.Errorf("%w %v", err, certificate)
	}

	keyPEM, err := ReadPEMFile(log, key, passphrase)
	if err != nil {
		log.Errorf("Failed reading key file: %+v", err)
		return nil, fmt.Errorf("%w %v", err, key)
	}

	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		log.Errorf("Failed loading client certificate %+v", err)
		return nil, err
	}

	// Do not log the key if it was provided as a string in the configuration to avoid
	// leaking private keys in the debug logs. Log when the key is a file path.
	if IsPEMString(key) {
		log.Debugf("Loading certificate: %v with key from PEM string in config", certificate)
	} else {
		log.Debugf("Loading certificate: %v and key %v", certificate, key)
	}

	return &cert, nil
}

// ReadPEMFile reads a PEM formatted string either from disk or passed as a plain text starting with a "-"
// and decrypt it with the provided password and  return the raw content.
func ReadPEMFile(log *logp.Logger, s, passphrase string) ([]byte, error) {
	pass := []byte(passphrase)
	var blocks []*pem.Block

	r, err := NewPEMReader(s)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	content, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	for len(content) > 0 {
		var block *pem.Block

		block, content = pem.Decode(content)
		if block == nil {
			if len(blocks) == 0 {
				return nil, errors.New("no pem file")
			}
			break
		}

		switch {
		case x509.IsEncryptedPEMBlock(block): //nolint: staticcheck // deprecated, we have to get rid of it
			block, err := decryptPKCS1Key(*block, pass)
			if err != nil {
				log.Errorf("Dropping encrypted pem block with private key, block type '%s': %s", block.Type, err)
				continue
			}
			blocks = append(blocks, &block)
		case block.Type == "ENCRYPTED PRIVATE KEY":
			block, err := decryptPKCS8Key(*block, pass)
			if err != nil {
				log.Errorf("Dropping encrypted pem block with private key, block type '%s', could not decypt as PKCS8: %s", block.Type, err)
				continue
			}
			blocks = append(blocks, &block)
		default:
			blocks = append(blocks, block)
		}
	}

	if len(blocks) == 0 {
		return nil, errors.New("no PEM blocks")
	}

	// re-encode available, decrypted blocks
	buffer := bytes.NewBuffer(nil)
	for _, block := range blocks {
		err := pem.Encode(buffer, block)
		if err != nil {
			return nil, err
		}
	}
	return buffer.Bytes(), nil
}

func decryptPKCS1Key(block pem.Block, passphrase []byte) (pem.Block, error) {
	if len(passphrase) == 0 {
		return block, errors.New("no passphrase available")
	}

	// Note, decrypting pem might succeed even with wrong password, but
	// only noise will be stored in buffer in this case.
	buffer, err := x509.DecryptPEMBlock(&block, passphrase) //nolint: staticcheck // deprecated, we have to get rid of it
	if err != nil {
		return block, fmt.Errorf("failed to decrypt pem: %w", err)
	}

	// DEK-Info contains encryption info. Remove header to mark block as
	// unencrypted.
	delete(block.Headers, "DEK-Info")
	block.Bytes = buffer

	return block, nil
}

func decryptPKCS8Key(block pem.Block, passphrase []byte) (pem.Block, error) {
	if len(passphrase) == 0 {
		return block, errors.New("no passphrase available")
	}

	key, err := pkcs8.ParsePKCS8PrivateKey(block.Bytes, passphrase)
	if err != nil {
		return block, fmt.Errorf("failed to parse key: %w", err)
	}

	switch key.(type) {
	case *rsa.PrivateKey:
		block.Type = "RSA PRIVATE KEY"
	case *ecdsa.PrivateKey:
		block.Type = "ECDSA PRIVATE KEY"
	default:
		return block, fmt.Errorf("unknown key type %T", key)
	}

	buffer, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return block, fmt.Errorf("failed to marshal decrypted private key: %w", err)
	}
	block.Bytes = buffer

	return block, nil
}

// LoadCertificateAuthorities read the slice of CAcert and return a Certpool.
func LoadCertificateAuthorities(CAs []string) (*x509.CertPool, []error) {
	errors := []error{}

	if len(CAs) == 0 {
		return nil, nil
	}

	log := logp.NewLogger(logSelector)
	roots := x509.NewCertPool()
	for _, s := range CAs {
		r, err := NewPEMReader(s)
		if err != nil {
			log.Errorf("Failed reading CA certificate: %+v", err)
			errors = append(errors, fmt.Errorf("%w reading %v", err, r))
			continue
		}
		defer r.Close()

		pemData, err := ioutil.ReadAll(r)
		if err != nil {
			log.Errorf("Failed reading CA certificate: %+v", err)
			errors = append(errors, fmt.Errorf("%w reading %v", err, r))
			continue
		}

		if ok := roots.AppendCertsFromPEM(pemData); !ok {
			log.Error("Failed to add CA to the cert pool, CA is not a valid PEM document")
			errors = append(errors, fmt.Errorf("%w adding %v to the list of known CAs", ErrNotACertificate, r))
			continue
		}
		log.Debugf("Successfully loaded CA certificate: %v", r)
	}

	return roots, errors
}

func extractMinMaxVersion(versions []TLSVersion) (uint16, uint16) {
	if len(versions) == 0 {
		versions = TLSDefaultVersions
	}

	minVersion := uint16(0xffff)
	maxVersion := uint16(0)
	for _, version := range versions {
		v := uint16(version)
		if v < minVersion {
			minVersion = v
		}
		if v > maxVersion {
			maxVersion = v
		}
	}

	return minVersion, maxVersion
}

// ResolveTLSVersion takes the integer representation and return the name.
func ResolveTLSVersion(v uint16) string {
	return TLSVersion(v).String()
}

// ResolveCipherSuite takes the integer representation and return the cipher name.
func ResolveCipherSuite(cipher uint16) string {
	return CipherSuite(cipher).String()
}

// PEMReader allows to read a certificate in PEM format either through the disk or from a string.
type PEMReader struct {
	reader   io.ReadCloser
	debugStr string
}

// NewPEMReader returns a new PEMReader.
func NewPEMReader(certificate string) (*PEMReader, error) {
	if IsPEMString(certificate) {
		return &PEMReader{reader: ioutil.NopCloser(strings.NewReader(certificate)), debugStr: "inline"}, nil
	}

	r, err := os.Open(certificate)
	if err != nil {
		return nil, err
	}
	return &PEMReader{reader: r, debugStr: certificate}, nil
}

// Close closes the target io.ReadCloser.
func (p *PEMReader) Close() error {
	return p.reader.Close()
}

// Read read bytes from the io.ReadCloser.
func (p *PEMReader) Read(b []byte) (n int, err error) {
	return p.reader.Read(b)
}

func (p *PEMReader) String() string {
	return p.debugStr
}

// IsPEMString returns true if the provided string match a PEM formatted certificate. try to pem decode to validate.
func IsPEMString(s string) bool {
	// Trim the certificates to make sure we tolerate any yaml weirdness, we assume that the string starts
	// with "-" and let further validation verifies the PEM format.
	return strings.HasPrefix(strings.TrimSpace(s), "-")
}
