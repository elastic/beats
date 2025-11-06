// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// Package dpop implements OAuth 2.0 Demonstrating Proof of Possession client behaviour
// as described in [RFC 9449].
//
// [RFC 9449]: https://datatracker.ietf.org/doc/html/rfc9449
package dpop

import (
	"crypto"
	"errors"
	"io"
	"net/http"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"
)

// NewTokenClient builds an [http.Client] to be used by [oauth2.Config] or [clientcredentials.Config]
// when exchanging code/client_credentials to get an access token.
// This client sends DPoP proofs to the token endpoint.
//
// [clientcredentials.Config]: https://pkg.go.dev/golang.org/x/oauth2/clientcredentials#Config
func NewTokenClient(claim Claimer, key crypto.Signer, signing jwt.SigningMethod, base *http.Client) (*http.Client, error) {
	pg, err := NewProofGenerator(claim, key, signing)
	if err != nil {
		return nil, err
	}
	tr := &TokenTransport{ProofGen: pg}
	if base != nil && base.Transport != nil {
		tr.Base = base.Transport
	}
	client := &http.Client{Transport: tr}
	return client, nil
}

// TokenTransport adds a DPoP proof to token endpoint HTTP requests.
// It retries once on DPoP-Nonce challenges (401/400/429 with DPoP-Nonce header).
// This transport should be installed on the [http.Client] used by oauth2 when fetching tokens.
type TokenTransport struct {
	Base     http.RoundTripper
	ProofGen *ProofGenerator
}

// RoundTrip implements [http.RoundTripper], injecting a DPoP proof into token
// endpoint requests and handling one retry on a nonce challenge.
func (t *TokenTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.Base
	if base == nil {
		base = http.DefaultTransport
	}
	if t.ProofGen == nil {
		return nil, errors.New("token dpop transport requires ProofGenerator")
	}

	r := req.Clone(req.Context())
	proof, err := t.ProofGen.BuildProof(req.Context(), req.Method, req.URL.String(), ProofOptions{})
	if err != nil {
		return nil, err
	}
	r.Header.Set("DPoP", proof)
	resp, err := base.RoundTrip(r)
	if err != nil {
		return resp, err
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusTooManyRequests {
		nonce := resp.Header.Get("DPoP-Nonce")
		if nonce != "" {
			discard(resp.Body)
			proof, err = t.ProofGen.BuildProof(req.Context(), req.Method, req.URL.String(), ProofOptions{Nonce: nonce})
			if err != nil {
				return nil, err
			}
			r2 := req.Clone(req.Context())
			r2.Header.Set("DPoP", proof)
			return base.RoundTrip(r2)
		}
	}
	return resp, nil
}

// NewResourceClient builds an [http.Client] that wraps [oauth2.TokenSource] and sends DPoP proofs
// and Authorization: DPoP «access_token» to protected resource endpoints.
func NewResourceClient(claim Claimer, key crypto.Signer, signing jwt.SigningMethod, ts oauth2.TokenSource, base *http.Client) (*http.Client, error) {
	if ts == nil {
		return nil, errors.New("token source is required")
	}
	pg, err := NewProofGenerator(claim, key, signing)
	if err != nil {
		return nil, err
	}
	tr := &Transport{TokenSource: ts, ProofGen: pg}
	if base != nil && base.Transport != nil {
		tr.Base = base.Transport
	}
	client := &http.Client{Transport: tr}
	return client, nil
}

// Transport decorates an underlying RoundTripper to add DPoP proofs and bearer auth.
// It uses the provided TokenSource for access tokens and adds both Authorization and DPoP headers.

// Transport is an http.RoundTripper that adds DPoP proofs and Authorization
// headers (Authorization: DPoP «access_token») to outgoing requests using the
// provided [oauth2.TokenSource]. It retries once on a DPoP-Nonce challenge.
type Transport struct {
	Base        http.RoundTripper
	TokenSource oauth2.TokenSource
	ProofGen    *ProofGenerator
}

func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.Base
	if base == nil {
		base = http.DefaultTransport
	}
	if t.TokenSource == nil || t.ProofGen == nil {
		return nil, errors.New("dpop transport requires TokenSource and ProofGenerator")
	}
	tok, err := t.TokenSource.Token()
	if err != nil {
		return nil, err
	}
	// clone the request to avoid mutating the original
	r := req.Clone(req.Context())
	if tok.AccessToken != "" {
		r.Header.Set("Authorization", "DPoP "+tok.AccessToken)
	}
	proof, err := t.ProofGen.BuildProof(req.Context(), req.Method, req.URL.String(), ProofOptions{AccessToken: tok.AccessToken})
	if err != nil {
		return nil, err
	}
	r.Header.Set("DPoP", proof)
	resp, err := base.RoundTrip(r)
	if err != nil {
		return resp, err
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusTooManyRequests {
		// Retry once if DPoP-Nonce provided
		nonce := resp.Header.Get("DPoP-Nonce")
		if nonce != "" {
			discard(resp.Body)
			proof, err = t.ProofGen.BuildProof(req.Context(), req.Method, req.URL.String(), ProofOptions{AccessToken: tok.AccessToken, Nonce: nonce})
			if err != nil {
				return nil, err
			}
			r2 := req.Clone(req.Context())
			if tok.AccessToken != "" {
				r2.Header.Set("Authorization", "DPoP "+tok.AccessToken)
			}
			r2.Header.Set("DPoP", proof)
			return base.RoundTrip(r2)
		}
	}
	return resp, nil
}

func discard(r io.ReadCloser) {
	io.Copy(io.Discard, r) //nolint:errcheck // ¯\_(ツ)_/¯
	r.Close()
}
