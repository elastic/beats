// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//nolint:bodyclose,errcheck,noctx // These are redundant in the context of testing.
package dpop

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"
)

func TestBuildProofIncludesRequiredClaims(t *testing.T) {
	key, err := generateECDSAP256Key()
	if err != nil {
		t.Fatalf("unexpected error generating key: %v", err)
	}
	pg, err := NewProofGenerator(durationClaimer(0), key, jwt.GetSigningMethod("ES256"))
	if err != nil {
		t.Fatalf("unexpected error making proof generator: %v", err)
	}
	now := time.Now().Unix()
	proof, err := pg.BuildProof(context.Background(), http.MethodGet, "https://api.example.com/path?q=1#frag", ProofOptions{})
	if err != nil {
		t.Fatalf("unexpected error building proof: %v", err)
	}
	parts := strings.Split(proof, ".")
	if len(parts) != 3 {
		t.Fatalf("expected 3 parts, got %d", len(parts))
	}
	var header map[string]any
	decodePart(t, parts[0], &header)
	if header["typ"] != "dpop+jwt" {
		t.Errorf("wrong typ: %v", header["typ"])
	}
	if header["alg"] != "ES256" {
		t.Errorf("wrong alg: %v", header["alg"])
	}
	if _, ok := header["jwk"].(map[string]any); !ok {
		t.Errorf("missing jwk")
	}
	var claims ProofClaims
	decodePart(t, parts[1], &claims)
	if claims.Method != "GET" {
		t.Errorf("wrong htm: %v", claims.Method)
	}
	if claims.URL != "https://api.example.com/path?q=1" {
		t.Errorf("wrong htu: %v", claims.URL)
	}
	if claims.ID == "" {
		t.Errorf("missing jti")
	}
	iat, err := claims.GetIssuedAt()
	if err != nil {
		t.Fatalf("unexpected error getting iat: %v", err)
	}
	if iat.Unix() < now-5 || iat.Unix() > now+5 {
		t.Errorf("iat out of range: %v", iat)
	}
}

func TestResourceTransportSetsHeadersAndAth(t *testing.T) {
	key, err := generateECDSAP256Key()
	if err != nil {
		t.Fatalf("unexpected error generating key: %v", err)
	}
	accessToken := "test-token"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "DPoP "+accessToken {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		proof := r.Header.Get("DPoP")
		if proof == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		parts := strings.Split(proof, ".")
		var claims ProofClaims
		decodePart(t, parts[1], &claims)
		if claims.AccessTokenHash == nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	pg, err := NewProofGenerator(durationClaimer(0), key, jwt.GetSigningMethod("ES256"))
	if err != nil {
		t.Fatalf("unexpected error making proof generator: %v", err)
	}
	ts := staticTokenSource{token: &oauth2.Token{AccessToken: accessToken, TokenType: "DPoP"}}
	cl := &http.Client{Transport: &Transport{TokenSource: ts, ProofGen: pg}}
	req, err := http.NewRequest(http.MethodGet, srv.URL+"/resource", nil)
	if err != nil {
		t.Fatalf("unexpected error making request: %v", err)
	}
	res, err := cl.Do(req)
	if err != nil {
		t.Fatalf("unexpected error performing request: %v", err)
	}
	if res.StatusCode != http.StatusOK {
		t.Errorf("unexpected status: %d", res.StatusCode)
	}
}

type staticTokenSource struct{ token *oauth2.Token }

func (s staticTokenSource) Token() (*oauth2.Token, error) { return s.token, nil }

func TestTokenTransportRetriesWithNonce(t *testing.T) {
	key, err := generateECDSAP256Key()
	if err != nil {
		t.Fatalf("unexpected error generating key: %v", err)
	}
	first := true
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if first {
			first = false
			w.Header().Set("DPoP-Nonce", "abc123")
			w.WriteHeader(401)
			return
		}
		proof := r.Header.Get("DPoP")
		parts := strings.Split(proof, ".")
		var claims ProofClaims
		decodePart(t, parts[1], &claims)
		if claims.Nonce == nil || *claims.Nonce != "abc123" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	pg, err := NewProofGenerator(durationClaimer(0), key, jwt.GetSigningMethod("ES256"))
	if err != nil {
		t.Fatalf("unexpected error making proof generator: %v", err)
	}
	cl := &http.Client{Transport: &TokenTransport{ProofGen: pg}}
	req, err := http.NewRequest(http.MethodPost, srv.URL+"/token", nil)
	if err != nil {
		t.Fatalf("unexpected error making request: %v", err)
	}
	res, err := cl.Do(req)
	if err != nil {
		t.Fatalf("unexpected error performing request: %v", err)
	}
	if res.StatusCode != http.StatusOK {
		t.Errorf("unexpected status: %d", res.StatusCode)
	}
}

func TestFlow(t *testing.T) {
	key, err := generateECDSAP256Key()
	if err != nil {
		t.Fatalf("unexpected error generating key: %v", err)
	}
	signing := jwt.GetSigningMethod("ES256")
	claim := durationClaimer(5 * time.Minute)
	const (
		user     = "client"
		password = "secret"

		authCode    = "auth-code"
		accessToken = "resource-token"
		nonce       = "abc123"
	)

	mux := http.NewServeMux()
	var (
		tokenURL string
		first    = true
	)
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		gotUser, gotPassword, ok := r.BasicAuth()
		if !ok {
			t.Fatalf("did not get basic auth: %v", err)
		}
		if gotUser != user {
			t.Fatalf("unexpected users: got %q, want %q", gotUser, user)
		}
		if gotPassword != password {
			t.Fatalf("unexpected users: got %q, want %q", gotPassword, password)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method for token endpoint: %s", r.Method)
		}
		if first {
			first = false
			w.Header().Set("dpop-nonce", "abc123")
			w.WriteHeader(401)
			return
		}
		err := r.ParseForm()
		if err != nil {
			t.Fatalf("unexpected error parsing form: %v", err)
		}
		gotCode := r.Form.Get("code")
		if gotCode != authCode {
			t.Fatalf("unexpected code: %s", gotCode)
		}
		claims := decodeProofClaims(t, r.Header.Get("dpop"))
		if claims.Nonce == nil {
			t.Fatal("no nonce")
		}
		if *claims.Nonce != nonce {
			t.Fatalf("unexpected nonce: got %q, want %q", *claims.Nonce, nonce)
		}
		if claims.Method != http.MethodPost {
			t.Fatalf("unexpected proof method: %s", claims.Method)
		}
		if claims.URL != tokenURL {
			t.Fatalf("unexpected proof htu: %s", claims.URL)
		}
		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(map[string]any{
			"access_token": accessToken,
			"token_type":   "DPoP",
			"expires_in":   3600,
		})
		if err != nil {
			t.Fatalf("encode token response: %v", err)
		}
	})
	var resourceURL string
	mux.HandleFunc("/resource", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("unexpected method for resource endpoint: %s", r.Method)
		}
		gotAuth := r.Header.Get("authorization")
		if gotAuth != "DPoP "+accessToken {
			t.Fatalf("unexpected authorization header: %s", gotAuth)
		}
		claims := decodeProofClaims(t, r.Header.Get("dpop"))
		if claims.Method != http.MethodGet {
			t.Fatalf("unexpected resource proof method: %s", claims.Method)
		}
		if claims.URL != resourceURL {
			t.Fatalf("unexpected resource proof htu: %s", claims.URL)
		}
		if claims.AccessTokenHash == nil {
			t.Fatal("missing access token hash in resource proof")
		}
		wantAth, err := sha256Base64URL(accessToken)
		if err != nil {
			t.Fatalf("ath hash: %v", err)
		}
		if *claims.AccessTokenHash != wantAth {
			t.Fatalf("unexpected ath: %s", *claims.AccessTokenHash)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	tokenURL = srv.URL + "/token"
	resourceURL = srv.URL + "/resource"

	ctx := context.Background()
	tokenClient, err := NewTokenClient(claim, key, signing, nil)
	if err != nil {
		t.Fatalf("unexpected error creating token client: %v", err)
	}
	oauthCtx := context.WithValue(ctx, oauth2.HTTPClient, tokenClient)
	cfg := oauth2.Config{
		ClientID:     user,
		ClientSecret: password,
		Endpoint:     oauth2.Endpoint{TokenURL: tokenURL},
	}
	tok, err := cfg.Exchange(oauthCtx, authCode)
	if err != nil {
		t.Fatalf("exchange token: %v", err)
	}

	ts := cfg.TokenSource(oauthCtx, tok)
	src := oauth2.ReuseTokenSource(tok, ts)
	baseClient := oauth2.NewClient(ctx, src)
	resourceClient, err := NewResourceClient(claim, key, signing, src, baseClient)
	if err != nil {
		t.Fatalf("unexpected error creating resource client: %v", err)
	}

	res, err := resourceClient.Get(resourceURL)
	if err != nil {
		t.Fatalf("unexpected error performing resource request: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Errorf("unexpected resource status: %d", res.StatusCode)
	}
	var got bytes.Buffer
	_, err = io.Copy(&got, res.Body)
	if got.String() != "ok" {
		t.Errorf("unexpected resource response body: %q, want %q", &got, "ok")
	}
}

func generateECDSAP256Key() (*ecdsa.PrivateKey, error) {
	return ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
}

func decodeProofClaims(t *testing.T, proof string) ProofClaims {
	t.Helper()
	if proof == "" {
		t.Fatalf("missing dpop proof")
	}
	parts := strings.Split(proof, ".")
	if len(parts) != 3 {
		t.Fatalf("invalid proof parts: %d", len(parts))
	}
	var claims ProofClaims
	decodePart(t, parts[1], &claims)
	return claims
}

func decodePart(t *testing.T, part string, v any) {
	t.Helper()
	b, err := base64.RawURLEncoding.DecodeString(part)
	if err != nil {
		t.Fatalf("decode base64: %v", err)
	}
	if err := json.Unmarshal(b, v); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
}

// durationClaimer is an example Claimer that returns a *jwt.RegisteredClaims
// with a defined duration of validity.
type durationClaimer time.Duration

func (c durationClaimer) Claims() *jwt.RegisteredClaims {
	now := time.Now()
	claims := &jwt.RegisteredClaims{
		IssuedAt: jwt.NewNumericDate(now),
		ID:       RandomJTI(),
	}
	if c >= 0 {
		claims.ExpiresAt = jwt.NewNumericDate(now.Add(time.Duration(c)))
	}
	return claims
}
