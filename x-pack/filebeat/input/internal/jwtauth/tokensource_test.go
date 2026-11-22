// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package jwtauth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"
)

func testSigner(t *testing.T, life time.Duration, claims map[string]any) *Signer {
	t.Helper()
	s, err := NewSigner(SignerConfig{Key: mustKey(t), Lifetime: life, Claims: claims})
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func verify(t *testing.T, signed string, now time.Time) jwt.MapClaims {
	t.Helper()
	claims := jwt.MapClaims{}
	_, err := jwt.ParseWithClaims(signed, claims, func(*jwt.Token) (any, error) {
		return mustKey(t).Public(), nil
	}, jwt.WithTimeFunc(func() time.Time { return now }))
	if err != nil {
		t.Fatalf("verifying jwt: %v", err)
	}
	return claims
}

func TestDirect(t *testing.T) {
	s := testSigner(t, time.Hour, map[string]any{"iss": "me"})
	before := time.Now()
	tok, err := Direct(s).Token()
	if err != nil {
		t.Fatal(err)
	}
	after := time.Now()

	if tok.TokenType != "Bearer" {
		t.Errorf("TokenType = %q; want Bearer", tok.TokenType)
	}
	claims := verify(t, tok.AccessToken, after)
	if claims["iss"] != "me" {
		t.Errorf("iss = %v; want me", claims["iss"])
	}
	// Expiry is the JWT lifetime less a margin.
	lo, hi := before.Add(time.Hour-time.Minute), after.Add(time.Hour-time.Minute)
	if tok.Expiry.Before(lo) || tok.Expiry.After(hi) {
		t.Errorf("Expiry = %v; want in [%v, %v]", tok.Expiry, lo, hi)
	}
	if !tok.Valid() {
		t.Error("token is not valid")
	}
}

func TestDirect_shortLifetime(t *testing.T) {
	s := testSigner(t, 30*time.Second, map[string]any{"iss": "me"})
	tok, err := Direct(s).Token()
	if err != nil {
		t.Fatal(err)
	}
	// The margin must not make a short-lived token expire at once.
	if remain := time.Until(tok.Expiry); remain < 20*time.Second {
		t.Errorf("token expires in %v; want more than 20s", remain)
	}
}

// tokenServer is a token endpoint that records its requests and verifies
// the assertion in each.
type tokenServer struct {
	*httptest.Server

	mu    sync.Mutex
	forms []url.Values
}

func newTokenServer(t *testing.T, handle func(w http.ResponseWriter, form url.Values, n int)) *tokenServer {
	t.Helper()
	ts := &tokenServer{}
	ts.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		ts.mu.Lock()
		ts.forms = append(ts.forms, r.PostForm)
		n := len(ts.forms)
		ts.mu.Unlock()
		handle(w, r.PostForm, n)
	}))
	t.Cleanup(ts.Close)
	return ts
}

//nolint:errcheck // No point checking errors in test server.
func okToken(w http.ResponseWriter, _ url.Values, _ int) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"access_token": "tok", "token_type": "Bearer", "expires_in": 3600})
}

func TestGrant(t *testing.T) {
	srv := newTokenServer(t, okToken)
	s := testSigner(t, 45*time.Second, map[string]any{"iss": "key", "sub": "user", "aud": "example.com", "scope": "signature impersonation"})

	src := Grant(context.Background(), s, GrantConfig{
		TokenURL:     srv.URL,
		ClientID:     "cid",
		ClientSecret: "secret",
		Params:       url.Values{"extra": {"x"}, "grant_type": {"ignored"}},
		// Scopes are not sent for the jwt-bearer grant.
		Scopes: []string{"ignored"},
	})
	tok, err := src.Token()
	if err != nil {
		t.Fatal(err)
	}
	if tok.AccessToken != "tok" {
		t.Errorf("AccessToken = %q; want tok", tok.AccessToken)
	}

	form := srv.forms[0]
	want := map[string]string{
		"grant_type":    "urn:ietf:params:oauth:grant-type:jwt-bearer",
		"client_id":     "cid",
		"client_secret": "secret",
		"extra":         "x",
	}
	for k, v := range want {
		if got := form.Get(k); got != v {
			t.Errorf("form %q = %q; want %q", k, got, v)
		}
	}
	if form.Has("scope") {
		t.Errorf("unexpected scope parameter: %q", form.Get("scope"))
	}
	claims := verify(t, form.Get("assertion"), time.Now())
	if claims["iss"] != "key" || claims["sub"] != "user" || claims["aud"] != "example.com" || claims["scope"] != "signature impersonation" {
		t.Errorf("unexpected claims: %v", claims)
	}
}

func TestGrant_noClientCredentials(t *testing.T) {
	srv := newTokenServer(t, okToken)
	s := testSigner(t, 45*time.Second, map[string]any{"iss": "key"})

	_, err := Grant(context.Background(), s, GrantConfig{TokenURL: srv.URL}).Token()
	if err != nil {
		t.Fatal(err)
	}
	form := srv.forms[0]
	if form.Has("client_id") || form.Has("client_secret") {
		t.Errorf("client credentials sent when none are configured: %v", form)
	}
}

//nolint:errcheck // No point checking errors in test server.
func TestGrant_clockSkewRetry(t *testing.T) {
	// The server's clock is an hour ahead of ours.
	serverTime := time.Now().Add(time.Hour)
	srv := newTokenServer(t, func(w http.ResponseWriter, form url.Values, n int) {
		if n == 1 {
			w.Header().Set("Date", serverTime.UTC().Format(http.TimeFormat))
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"invalid_grant","error_description":"exp is too far in the future"}`))
			return
		}
		okToken(w, form, n)
	})
	s := testSigner(t, 45*time.Second, map[string]any{"iss": "key"})

	tok, err := Grant(context.Background(), s, GrantConfig{TokenURL: srv.URL}).Token()
	if err != nil {
		t.Fatal(err)
	}
	if tok.AccessToken != "tok" {
		t.Errorf("AccessToken = %q; want tok", tok.AccessToken)
	}
	if len(srv.forms) != 2 {
		t.Fatalf("got %d requests; want 2", len(srv.forms))
	}
	first := verify(t, srv.forms[0].Get("assertion"), time.Now())
	second := verify(t, srv.forms[1].Get("assertion"), serverTime)
	if first["iat"] == second["iat"] {
		t.Error("retry reused the original iat; want the server's time")
	}
	if got, want := int64(second["iat"].(float64)), serverTime.Unix(); got < want-1 || got > want+1 {
		t.Errorf("retry iat = %d; want about %d", got, want)
	}
}

//nolint:errcheck // No point checking errors in test server.
func TestGrant_noRetryOtherErrors(t *testing.T) {
	srv := newTokenServer(t, func(w http.ResponseWriter, _ url.Values, _ int) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"invalid_client"}`))
	})
	s := testSigner(t, 45*time.Second, map[string]any{"iss": "key"})

	_, err := Grant(context.Background(), s, GrantConfig{TokenURL: srv.URL}).Token()
	if err == nil {
		t.Fatal("got nil error; want error")
	}
	if len(srv.forms) != 1 {
		t.Errorf("got %d requests; want 1", len(srv.forms))
	}
}

func TestGrant_usesContextClient(t *testing.T) {
	srv := newTokenServer(t, okToken)
	s := testSigner(t, 45*time.Second, map[string]any{"iss": "key"})

	var used bool
	cli := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		used = true
		return http.DefaultTransport.RoundTrip(r)
	})}
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, cli)
	_, err := Grant(ctx, s, GrantConfig{TokenURL: srv.URL}).Token()
	if err != nil {
		t.Fatal(err)
	}
	if !used {
		t.Error("the HTTP client in the context was not used")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestClientAssertion(t *testing.T) {
	srv := newTokenServer(t, okToken)
	s := testSigner(t, time.Hour, map[string]any{"iss": "cid", "sub": "cid", "aud": "https://as.example.com/token"})

	_, err := ClientAssertion(context.Background(), s, GrantConfig{
		TokenURL: srv.URL,
		Scopes:   []string{"okta.logs.read", "okta.users.read"},
	}).Token()
	if err != nil {
		t.Fatal(err)
	}
	form := srv.forms[0]
	want := map[string]string{
		"grant_type":            "client_credentials",
		"scope":                 "okta.logs.read okta.users.read",
		"client_assertion_type": "urn:ietf:params:oauth:client-assertion-type:jwt-bearer",
	}
	for k, v := range want {
		if got := form.Get(k); got != v {
			t.Errorf("form %q = %q; want %q", k, got, v)
		}
	}
	claims := verify(t, form.Get("client_assertion"), time.Now())
	if claims["iss"] != "cid" || claims["sub"] != "cid" || claims["aud"] != "https://as.example.com/token" {
		t.Errorf("unexpected claims: %v", claims)
	}
}
