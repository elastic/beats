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
	"crypto/tls"
	"fmt"
	"sync"
	"time"

	"github.com/elastic/elastic-agent-libs/logp"
)

const defaultReloadInterval = 5 * time.Second

// CertReloader periodically reloads TLS certificate and key files from disk.
// On each call to GetCertificate (i.e., on each TLS handshake), it checks
// whether the reload interval has elapsed and, if so, re-reads the files from
// disk. Invalid cert/key pairs are logged and skipped, preserving the last
// successfully loaded certificate.
//
// This design follows the OpenTelemetry Collector's configtls approach: no file
// watchers or extra goroutines — just a time check on the handshake hot path.
type CertReloader struct {
	certPath         string
	keyPath          string
	passphrase       string
	disableLegacyPEM bool
	reloadInterval   time.Duration
	log              *logp.Logger

	mu         sync.RWMutex
	cert       *tls.Certificate
	nextReload time.Time
}

// CertReloaderOption is a functional option for configuring a CertReloader.
type CertReloaderOption func(*CertReloader)

// WithReloadInterval sets how often the certificate files are re-read from disk.
// If not specified, a default of 5 seconds is used.
func WithReloadInterval(d time.Duration) CertReloaderOption {
	return func(r *CertReloader) {
		r.reloadInterval = d
	}
}

// WithPassphrase sets the passphrase used to decrypt encrypted private keys.
func WithPassphrase(passphrase string) CertReloaderOption {
	return func(r *CertReloader) {
		r.passphrase = passphrase
	}
}

// WithDisableLegacyPEMSupport configures whether encrypted PKCS#1 PEM keys
// should be treated as an error instead of a deprecation warning.
func WithDisableLegacyPEMSupport(disable bool) CertReloaderOption {
	return func(r *CertReloader) {
		r.disableLegacyPEM = disable
	}
}

// NewCertReloader creates a CertReloader for the given cert and key file paths.
// It performs an initial load of the certificate pair, returning an error if the
// initial load fails.
func NewCertReloader(certPath, keyPath string, opts ...CertReloaderOption) (*CertReloader, error) {
	if certPath == "" || keyPath == "" {
		return nil, fmt.Errorf("certificate and key paths must be non-empty")
	}
	if IsPEMString(certPath) {
		return nil, fmt.Errorf("certificate must be a file path, not an inline PEM")
	}
	if IsPEMString(keyPath) {
		return nil, fmt.Errorf("key must be a file path, not an inline PEM")
	}

	r := &CertReloader{
		certPath:       certPath,
		keyPath:        keyPath,
		reloadInterval: defaultReloadInterval,
		log:            logp.NewLogger("tls"),
	}
	for _, opt := range opts {
		opt(r)
	}

	cert, err := r.loadKeyPair()
	if err != nil {
		return nil, fmt.Errorf("initial certificate load failed: %w", err)
	}
	r.cert = &cert
	r.nextReload = time.Now().Add(r.reloadInterval)

	return r, nil
}

// loadKeyPair mirrors the static cert-loading path in LoadCertificate (tls.go).
func (r *CertReloader) loadKeyPair() (tls.Certificate, error) {
	certPEM, err := readPEMFile(r.log, r.certPath, r.passphrase, r.disableLegacyPEM)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("reading certificate file: %w", err)
	}
	keyPEM, err := readPEMFile(r.log, r.keyPath, r.passphrase, r.disableLegacyPEM)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("reading key file: %w", err)
	}
	return tls.X509KeyPair(certPEM, keyPEM)
}

// GetCertificate returns the current certificate, reloading from disk if the
// reload interval has elapsed. It is safe for concurrent use and is intended
// to be used with tls.Config.GetCertificate.
func (r *CertReloader) GetCertificate(_ *tls.ClientHelloInfo) (*tls.Certificate, error) {
	return r.getCertificate()
}

// GetClientCertificate returns the current certificate, reloading from disk if
// the reload interval has elapsed. It is safe for concurrent use and is
// intended to be used with tls.Config.GetClientCertificate.
func (r *CertReloader) GetClientCertificate(_ *tls.CertificateRequestInfo) (*tls.Certificate, error) {
	return r.getCertificate()
}

func (r *CertReloader) getCertificate() (*tls.Certificate, error) {
	r.mu.RLock()
	if time.Now().Before(r.nextReload) {
		defer r.mu.RUnlock()
		return r.cert, nil
	}
	r.mu.RUnlock()

	r.mu.Lock()
	defer r.mu.Unlock()

	// Another goroutine may have reloaded while we waited for the write lock.
	if time.Now().Before(r.nextReload) {
		return r.cert, nil
	}

	r.nextReload = time.Now().Add(r.reloadInterval)

	cert, err := r.loadKeyPair()
	if err != nil {
		r.log.Errorf("Failed to reload TLS certificate from %s and %s, continuing with current certificate: %v", r.certPath, r.keyPath, err)
		return r.cert, nil
	}
	r.cert = &cert
	return r.cert, nil
}
