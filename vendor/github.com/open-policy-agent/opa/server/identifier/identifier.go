// Copyright 2017 The OPA Authors.  All rights reserved.
// Use of this source code is governed by an Apache2
// license that can be found in the LICENSE file.

// Package identifier provides handlers for associating an identity with incoming requests.
package identifier

import (
	"context"
	"net/http"
	"regexp"
)

// Identity returns the identity of the caller associated with ctx.
func Identity(r *http.Request) (string, bool) {
	ctx := r.Context()
	v, ok := ctx.Value(identity).(string)
	if ok {
		return v, true
	}
	return "", false
}

// SetIdentity returns a new http.Request with the identity set to v.
func SetIdentity(r *http.Request, v string) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), identity, v))
}

type identityKey string

const identity = identityKey("org.openpolicyagent/identity")

// TokenBased extracts Bearer tokens from the request.
type TokenBased struct {
	inner http.Handler
}

// NewTokenBased returns a new TokenBased object.
func NewTokenBased(inner http.Handler) *TokenBased {
	return &TokenBased{
		inner: inner,
	}
}

var bearerTokenRegexp = regexp.MustCompile(`^Bearer\s+(\S+)$`)

func (h *TokenBased) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	value := r.Header.Get("Authorization")
	if len(value) > 0 {
		match := bearerTokenRegexp.FindStringSubmatch(value)
		if len(match) > 0 {
			r = SetIdentity(r, match[1])
		}
	}

	h.inner.ServeHTTP(w, r)
}
