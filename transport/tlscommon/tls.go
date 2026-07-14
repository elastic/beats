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
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"

	"github.com/elastic/elastic-agent-libs/logp"
)

// LoadCertificate will load a certificate from disk and return a tls.Certificate or error
func LoadCertificate(config *CertificateConfig, logger *logp.Logger) (*tls.Certificate, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	certificate := config.Certificate
	key := config.Key
	if certificate == "" {
		return nil, nil
	}

	log := logger.Named("tls")
	passphrase, err := config.resolvePassphrase()
	if err != nil {
		return nil, err
	}

	certPEM, err := readPEMFile(log, certificate, passphrase, config.DisableLegacyPEMSupport)
	if err != nil {
		log.Errorf("Failed reading certificate %s: %v", pemSource(certificate), err)
		return nil, fmt.Errorf("%w (%s)", err, pemSource(certificate))
	}

	keyPEM, err := readPEMFile(log, key, passphrase, config.DisableLegacyPEMSupport)
	if err != nil {
		log.Errorf("Failed reading key: %v", err)
		return nil, fmt.Errorf("failed reading key: %w", err)
	}

	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		log.Errorf("Failed loading client certificate %+v", err)
		return nil, err
	}

	if isInlinePEM(key) {
		log.Debug("Loading certificate with key from PEM")
	} else {
		log.Debug("Loading certificate with key from file")
	}

	return &cert, nil
}

// ReadPEMFile reads a PEM formatted string either from disk or passed as a plain text starting with a "-"
// and decrypt it with the provided password and return the raw content.
// Encrypted PKCS#1 PEM blocks are decrypted with a deprecation warning. To treat them as an error,
// use LoadCertificate with DisableLegacyPEMSupport set on the CertificateConfig.
func ReadPEMFile(log *logp.Logger, s, passphrase string) ([]byte, error) {
	return readPEMFile(log, s, passphrase, false)
}

func readPEMFile(log *logp.Logger, s, passphrase string, disableLegacy bool) ([]byte, error) {
	pass := []byte(passphrase)
	var blocks []*pem.Block

	r, err := NewPEMReader(s)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	content, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	var errs error
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
		case x509.IsEncryptedPEMBlock(block): //nolint:staticcheck // deprecated PKCS#1 PEM encryption
			if disableLegacy {
				return nil, fmt.Errorf("encrypted PKCS#1 PEM keys are not supported; convert to PKCS#8")
			}
			log.Warnf("Encrypted PKCS#1 PEM keys are deprecated and insecure, support will be removed in a future release; please convert the key to encrypted PKCS#8")
			decBlock, decErr := decryptPKCS1Key(*block, pass)
			if decErr != nil {
				log.Errorf("Dropping encrypted pem block with private key, block type '%s', could not decrypt PKCS#1: %s", block.Type, decErr)
				errs = errors.Join(errs, decErr)
				continue
			}
			blocks = append(blocks, &decBlock)
		case block.Type == "ENCRYPTED PRIVATE KEY":
			block, err := decryptPKCS8Key(*block, pass)
			if err != nil {
				log.Errorf("Dropping encrypted pem block with private key, block type '%s', could not decrypt as PKCS8: %s", block.Type, err)
				errs = errors.Join(errs, err)
				continue
			}
			blocks = append(blocks, &block)
		default:
			blocks = append(blocks, block)
		}
	}

	if len(blocks) == 0 {
		return nil, errors.Join(errors.New("no PEM blocks"), errs)
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

// LoadCertificateAuthorities read the slice of CAcert and return a Certpool.
func LoadCertificateAuthorities(CAs []string, logger *logp.Logger) (*x509.CertPool, []error) {
	errors := []error{}

	if len(CAs) == 0 {
		return nil, nil
	}

	log := logger.Named("tls")
	roots := x509.NewCertPool()
	for _, s := range CAs {
		r, err := NewPEMReader(s)
		if err != nil {
			log.Errorf("Failed reading CA certificate: %+v", err)
			errors = append(errors, fmt.Errorf("%w reading %s", err, pemSource(s)))
			continue
		}
		defer r.Close()

		pemData, err := io.ReadAll(r)
		if err != nil {
			log.Errorf("Failed reading CA certificate: %+v", err)
			errors = append(errors, fmt.Errorf("%w reading %s", err, pemSource(s)))
			continue
		}

		if ok := roots.AppendCertsFromPEM(pemData); !ok {
			log.Error("Failed to add CA to the cert pool, CA is not a valid PEM document")
			errors = append(errors, fmt.Errorf("%w adding %s to the list of known CAs", ErrNotACertificate, pemSource(s)))
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
	if isInlinePEM(certificate) {
		return &PEMReader{reader: io.NopCloser(strings.NewReader(certificate)), debugStr: "inline"}, nil
	}

	r, err := os.Open(certificate) //nolint:gosec // certificate is an operator-provided cert/key file path; opening it is the intended behavior
	if err != nil {
		// os.Open records the name it tried to open in *fs.PathError.Path
		// verbatim. Because a malformed inline PEM can be indistinguishable
		// from a file path (isInlinePEM is only a heuristic), that name may
		// actually be private key material, so it must never be echoed.
		// Preserve the *fs.PathError type -- so callers matching on it with
		// errors.As/errors.Is (e.g. fs.ErrNotExist) keep working -- but drop
		// the Path component.
		var pathErr *fs.PathError
		if errors.As(err, &pathErr) {
			return nil, &fs.PathError{Op: pathErr.Op, Err: pathErr.Err}
		}
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

// base64Alphabet is the standard base64 alphabet (RFC 4648) including the '='
// padding character. A PEM body stripped of its armor is a run of these
// characters.
const base64Alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/="

// minInlinePEMBodyLen is the length at or above which a bare, single-line,
// pure-base64 string with no path separators is treated as inline PEM content
// rather than a file path. The smallest realistic private-key encoding (an
// Ed25519 32-byte seed) is 44 base64 characters; RSA/EC keys are far longer. A
// legitimate file path this long that is also pure base64 with no separator or
// extension is vanishingly rare.
const minInlinePEMBodyLen = 44

// isInlinePEM reports whether s is (possibly malformed) inline PEM content
// rather than a filesystem path. A genuine path is a single line and contains
// no PEM armor; inline PEM is multi-line and/or contains "-----...-----"
// delimiters. Surrounding whitespace is trimmed first (consistent with
// IsPEMString) so a path with a trailing newline -- e.g. from a YAML block
// scalar -- is not misclassified as inline content. Used to keep key material
// out of os.Open errors and log lines: a malformed inline key that lost its
// leading dashes must not be mistaken for a path and handed to os.Open, which
// would echo the whole blob.
//
// Note this cannot be perfectly accurate: a single-line base64 blob containing
// a '/' is indistinguishable from a long file path, so such a malformed inline
// key is still classified as a path. Fully removing the ambiguity would require
// separate configuration options for inline PEM vs. file path, which would be a
// breaking change.
func isInlinePEM(s string) bool {
	trimmed := strings.TrimSpace(s)
	if IsPEMString(s) ||
		strings.ContainsAny(trimmed, "\r\n") ||
		strings.Contains(trimmed, "-----") {
		return true
	}

	// A malformed inline PEM can lose all of its armor and line breaks (e.g. a
	// copy-paste that drops the -----BEGIN/END----- lines), leaving a single
	// line of base64 that the checks above cannot tell apart from a file path.
	// Treat a long, pure-base64 string with no path-typical separator
	// (/, \, ., :) as inline content so it is redacted rather than handed to
	// os.Open, which would echo the whole blob.
	return len(trimmed) >= minInlinePEMBodyLen &&
		strings.Trim(trimmed, base64Alphabet) == "" &&
		!strings.ContainsAny(trimmed, `/\.:`)
}

// pemSource returns a log-safe description of s: the file path when s is a
// path, or a redacted placeholder when s is inline PEM content, so a private
// key provided inline in the configuration is never written to logs or errors.
func pemSource(s string) string {
	if isInlinePEM(s) {
		return "PEM REDACTED"
	}
	return s
}
