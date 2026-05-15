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
	"crypto/x509"
	"fmt"
	"sync"
	"time"

	"github.com/elastic/elastic-agent-libs/logp"
)

// CAReloader periodically reloads CA certificate files from disk.
// On each call to GetCertPool, it checks whether the reload interval has
// elapsed and, if so, re-reads the CA files from disk. Invalid or unreadable
// files are skipped, preserving the last successfully loaded pool.
type CAReloader struct {
	caPaths        []string
	reloadInterval time.Duration
	log            *logp.Logger

	mu         sync.RWMutex
	pool       *x509.CertPool
	nextReload time.Time
	// trustedFingerprintCerts holds CA certificates discovered at runtime via
	// CATrustedFingerprint. Unlike the CAs loaded from caPaths on disk, these
	// certs come from the peer's certificate chain during a TLS handshake and
	// must be explicitly re-added to the pool after every disk reload.
	trustedFingerprintCerts []*x509.Certificate
}

// NewCAReloader creates a CAReloader for the given CA file paths. Inline PEM
// strings are also accepted and will be included on every reload (they never
// change, but are needed to build a complete pool). An initial load is
// performed; an error is returned only if no CA could be loaded at all.
func NewCAReloader(caPaths []string, reloadInterval time.Duration) (*CAReloader, error) {
	if len(caPaths) == 0 {
		return nil, fmt.Errorf("at least one CA path must be provided")
	}

	if reloadInterval <= 0 {
		reloadInterval = defaultReloadInterval
	}

	r := &CAReloader{
		caPaths:        caPaths,
		reloadInterval: reloadInterval,
		log:            logp.NewLogger("tls"),
	}

	pool, errs := LoadCertificateAuthorities(r.caPaths)
	if len(errs) == len(r.caPaths) {
		return nil, fmt.Errorf("initial CA load failed, none of the %d CA(s) could be loaded: %v", len(r.caPaths), errs)
	}
	r.pool = pool
	r.nextReload = time.Now().Add(r.reloadInterval)

	return r, nil
}

// GetCertPool returns the current CA certificate pool, reloading from disk if
// the reload interval has elapsed. The returned pool is a snapshot: callers
// should not cache it across handshakes but instead call GetCertPool each time
// to pick up reloaded CAs. It is safe for concurrent use.
func (r *CAReloader) GetCertPool() *x509.CertPool {
	r.mu.RLock()
	if time.Now().Before(r.nextReload) {
		defer r.mu.RUnlock()
		return r.pool
	}
	r.mu.RUnlock()

	r.mu.Lock()
	defer r.mu.Unlock()

	if time.Now().Before(r.nextReload) {
		return r.pool
	}

	r.nextReload = time.Now().Add(r.reloadInterval)

	pool, errs := LoadCertificateAuthorities(r.caPaths)
	if len(errs) == len(r.caPaths) {
		r.log.Warnf("CA reload failed for all %d CA(s), keeping previous pool: %v", len(r.caPaths), errs)
		return r.pool
	}
	if len(errs) > 0 {
		r.log.Warnf("CA reload: %d/%d CA(s) failed to load, continuing with the ones that succeeded: %v", len(errs), len(r.caPaths), errs)
	}

	for _, cert := range r.trustedFingerprintCerts {
		pool.AddCert(cert)
	}

	r.pool = pool
	return r.pool
}

// IsDynamic reports that CAReloader provides a dynamically reloaded pool.
func (r *CAReloader) IsDynamic() bool { return true }

// AddTrustedCert registers a certificate so it is included in every pool
// returned by GetCertPool, surviving reloads from disk. Duplicate certs
// (by raw bytes) are ignored.
func (r *CAReloader) AddTrustedCert(cert *x509.Certificate) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, existing := range r.trustedFingerprintCerts {
		if bytes.Equal(existing.Raw, cert.Raw) {
			return
		}
	}

	r.trustedFingerprintCerts = append(r.trustedFingerprintCerts, cert)
	if r.pool != nil {
		r.pool.AddCert(cert)
	}
}
