// Copyright 2019 The OPA Authors.  All rights reserved.
// Use of this source code is governed by an Apache2
// license that can be found in the LICENSE file.

package identifier

import (
	"net/http"
)

// TLSBased extracts the CN of the client's TLS ceritificate
type TLSBased struct {
	inner http.Handler
}

// NewTLSBased returns a new TLSBased object.
func NewTLSBased(inner http.Handler) *TLSBased {
	return &TLSBased{
		inner: inner,
	}
}

func (h *TLSBased) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if tls := r.TLS; tls != nil {
		if certs := tls.PeerCertificates; len(certs) > 0 {
			r = SetIdentity(r, certs[0].Subject.ToRDNSequence().String())
		}
	}

	h.inner.ServeHTTP(w, r)
}
