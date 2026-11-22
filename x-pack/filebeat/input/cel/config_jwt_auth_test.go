// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cel

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"maps"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	conf "github.com/elastic/beats/v7/pkg/config"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/internal/jwtauth"
)

func TestConfigJWTValidation(t *testing.T) {
	// jwt builds an auth.jwt config from the base fields and overrides.
	jwt := func(mode string, override map[string]any) map[string]any {
		m := map[string]any{
			"mode":        mode,
			"private_key": testBoxPEMForConfig,
			"claims":      map[string]any{"iss": "me"},
		}
		if mode != jwtModeDirect {
			m["token_url"] = "https://example.com/token"
		}
		for k, v := range override {
			if v == nil {
				delete(m, k)
				continue
			}
			m[k] = v
		}
		return m
	}

	// DocuSign generates keys in the PKCS#1 form, "BEGIN RSA PRIVATE KEY".
	signer, err := jwtauth.ParsePrivateKey([]byte(testBoxPEMForConfig), "")
	if err != nil {
		t.Fatal(err)
	}
	rsaKey, ok := signer.(*rsa.PrivateKey)
	if !ok {
		t.Fatalf("test key is %T; want *rsa.PrivateKey", signer)
	}
	pkcs1 := string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(rsaKey)}))

	tests := []struct {
		name    string
		input   map[string]any
		wantErr string
	}{
		{name: "direct", input: map[string]any{"auth.jwt": jwt("direct", nil)}},
		{name: "grant", input: map[string]any{"auth.jwt": jwt("grant", nil)}},
		{name: "pkcs1_key", input: map[string]any{"auth.jwt": jwt("grant", map[string]any{"private_key": pkcs1})}},
		{name: "client_assertion", input: map[string]any{"auth.jwt": jwt("client_assertion", map[string]any{"scopes": []string{"a"}})}},
		{
			name: "encrypted_key",
			input: map[string]any{"auth.jwt": jwt("direct", map[string]any{
				"private_key": testBoxEncryptedKey,
				"passphrase":  "testpassphrase",
			})},
		},
		{
			name: "snowflake_placeholder",
			input: map[string]any{"auth.jwt": jwt("direct", map[string]any{
				"claims": map[string]any{"iss": "A.U.%{public_key_sha256}", "sub": "A.U"},
			})},
		},
		{
			name: "disabled_is_not_validated",
			input: map[string]any{"auth.jwt": map[string]any{
				"enabled": false,
				"mode":    "nonsense",
			}},
		},
		{
			name:    "no_mode",
			input:   map[string]any{"auth.jwt": jwt("", nil)},
			wantErr: "mode must be provided",
		},
		{
			name:    "unknown_mode",
			input:   map[string]any{"auth.jwt": jwt("bogus", nil)},
			wantErr: `unknown mode "bogus"`,
		},
		{
			name:    "direct_token_url",
			input:   map[string]any{"auth.jwt": jwt("direct", map[string]any{"token_url": "https://example.com/token"})}, //nolint:gosec // G101: not a credential
			wantErr: "token_url must not be set when mode is direct",
		},
		{
			name:    "direct_client_id",
			input:   map[string]any{"auth.jwt": jwt("direct", map[string]any{"client.id": "x"})},
			wantErr: "client.id and client.secret must not be set when mode is direct",
		},
		{
			name:    "direct_scopes",
			input:   map[string]any{"auth.jwt": jwt("direct", map[string]any{"scopes": []string{"a"}})},
			wantErr: "scopes must not be set when mode is direct",
		},
		{
			name:    "direct_endpoint_params",
			input:   map[string]any{"auth.jwt": jwt("direct", map[string]any{"endpoint_params": map[string]any{"a": "b"}})},
			wantErr: "endpoint_params must not be set when mode is direct",
		},
		{
			name:    "grant_no_token_url",
			input:   map[string]any{"auth.jwt": jwt("grant", map[string]any{"token_url": nil})},
			wantErr: "token_url must be provided when mode is grant",
		},
		{
			name:    "grant_scopes",
			input:   map[string]any{"auth.jwt": jwt("grant", map[string]any{"scopes": []string{"a"}})},
			wantErr: "scopes must not be set when mode is grant",
		},
		{
			name:    "grant_reserved_param",
			input:   map[string]any{"auth.jwt": jwt("grant", map[string]any{"endpoint_params": map[string]any{"assertion": "x"}})},
			wantErr: `endpoint_params must not include reserved key "assertion"`,
		},
		{
			name:    "client_assertion_reserved_param",
			input:   map[string]any{"auth.jwt": jwt("client_assertion", map[string]any{"endpoint_params": map[string]any{"scope": "x"}})},
			wantErr: `endpoint_params must not include reserved key "scope"`,
		},
		{
			name:    "no_key",
			input:   map[string]any{"auth.jwt": jwt("direct", map[string]any{"private_key": nil})},
			wantErr: "private_key must be provided",
		},
		{
			name:    "bad_key",
			input:   map[string]any{"auth.jwt": jwt("direct", map[string]any{"private_key": "not pem"})},
			wantErr: "private_key: no PEM data",
		},
		{
			name: "wrong_passphrase",
			input: map[string]any{"auth.jwt": jwt("direct", map[string]any{
				"private_key": testBoxEncryptedKey,
				"passphrase":  "bad",
			})},
			wantErr: "decrypting private key",
		},
		{
			name:    "iss_not_string",
			input:   map[string]any{"auth.jwt": jwt("direct", map[string]any{"claims": map[string]any{"iss": 42}})},
			wantErr: "claims.iss must be provided as a non-empty string",
		},
		{
			name:    "iss_empty",
			input:   map[string]any{"auth.jwt": jwt("direct", map[string]any{"claims": map[string]any{"iss": ""}})},
			wantErr: "claims.iss must be provided as a non-empty string",
		},
		{
			name:    "grant_reserved_client_id_param",
			input:   map[string]any{"auth.jwt": jwt("grant", map[string]any{"endpoint_params": map[string]any{"client_id": "x"}})},
			wantErr: `endpoint_params must not include reserved key "client_id"`,
		},
		{
			name:    "client_assertion_reserved_client_secret_param",
			input:   map[string]any{"auth.jwt": jwt("client_assertion", map[string]any{"endpoint_params": map[string]any{"client_secret": "x"}})},
			wantErr: `endpoint_params must not include reserved key "client_secret"`,
		},
		{
			name:    "no_iss",
			input:   map[string]any{"auth.jwt": jwt("direct", map[string]any{"claims": map[string]any{"sub": "x"}})},
			wantErr: "claims.iss must be provided",
		},
		{
			name:    "unknown_placeholder",
			input:   map[string]any{"auth.jwt": jwt("direct", map[string]any{"claims": map[string]any{"iss": "%{nope}"}})},
			wantErr: `unknown placeholder "%{nope}"`,
		},
		{
			name:    "bad_algorithm",
			input:   map[string]any{"auth.jwt": jwt("direct", map[string]any{"algorithm": "none"})},
			wantErr: `unsupported algorithm "none"`,
		},
		{
			name:    "algorithm_key_mismatch",
			input:   map[string]any{"auth.jwt": jwt("direct", map[string]any{"algorithm": "ES256"})},
			wantErr: "signing jwt",
		},
		{
			name:    "reserved_claim",
			input:   map[string]any{"auth.jwt": jwt("direct", map[string]any{"claims": map[string]any{"iss": "x", "exp": 1}})},
			wantErr: `"exp" is set automatically`,
		},
		{
			name:    "negative_lifetime",
			input:   map[string]any{"auth.jwt": jwt("direct", map[string]any{"lifetime": "-1s"})},
			wantErr: "lifetime must not be negative",
		},
		{
			name: "jwt_and_oauth2",
			input: map[string]any{
				"auth.jwt":    jwt("direct", nil),
				"auth.oauth2": map[string]any{"client.id": "a", "client.secret": "b", "token_url": "https://example.com"}, //nolint:gosec // G101: not a credential
			},
			wantErr: "only one kind of auth can be enabled",
		},
		{
			name: "jwt_and_basic",
			input: map[string]any{
				"auth.jwt":            jwt("direct", nil),
				"auth.basic.user":     "u",
				"auth.basic.password": "p",
			},
			wantErr: "only one kind of auth can be enabled",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.input["resource.url"] = "localhost"
			cfg := conf.MustNewConfigFrom(test.input)
			c := defaultConfig()
			c.Program = "{}"
			c.Redact = &redact{}
			err := cfg.Unpack(&c)
			switch {
			case test.wantErr == "" && err != nil:
				t.Errorf("unexpected error: %v", err)
			case test.wantErr != "" && err == nil:
				t.Errorf("got nil error; want error containing %q", test.wantErr)
			case test.wantErr != "" && !strings.Contains(err.Error(), test.wantErr):
				t.Errorf("error %q does not contain %q", err, test.wantErr)
			}
		})
	}
}

// testTokenURL is a token endpoint URL for tests that do not make requests.
const testTokenURL = "https://example.com/token" //nolint:gosec // G101: not a credential

func TestJWTAuthSignerDefaults(t *testing.T) {
	tests := []struct {
		name         string
		cfg          jwtAuthConfig
		wantAud      any
		wantLifetime time.Duration
	}{
		{
			name:         "direct_has_no_aud",
			cfg:          jwtAuthConfig{Mode: jwtModeDirect},
			wantAud:      nil,
			wantLifetime: 59 * time.Minute,
		},
		{
			name:         "grant_aud_is_token_url",
			cfg:          jwtAuthConfig{Mode: jwtModeGrant, TokenURL: testTokenURL},
			wantAud:      testTokenURL,
			wantLifetime: 45 * time.Second,
		},
		{
			name:         "client_assertion_aud_is_token_url",
			cfg:          jwtAuthConfig{Mode: jwtModeClientAssertion, TokenURL: testTokenURL},
			wantAud:      testTokenURL,
			wantLifetime: 45 * time.Second,
		},
		{
			name:         "explicit_aud_and_lifetime",
			cfg:          jwtAuthConfig{Mode: jwtModeGrant, TokenURL: testTokenURL, Claims: map[string]any{"aud": "example.com"}, Lifetime: 10 * time.Second},
			wantAud:      "example.com",
			wantLifetime: 10 * time.Second,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := test.cfg
			cfg.PrivateKey = testBoxPEMForConfig
			claims := map[string]any{"iss": "me"}
			maps.Copy(claims, cfg.Claims)
			cfg.Claims = claims

			s, err := cfg.signer()
			if err != nil {
				t.Fatal(err)
			}
			signed, err := s.Sign(time.Now())
			if err != nil {
				t.Fatal(err)
			}
			got := jwt.MapClaims{}
			_, _, err = jwt.NewParser().ParseUnverified(signed, got)
			if err != nil {
				t.Fatal(err)
			}
			if got["aud"] != test.wantAud {
				t.Errorf("aud = %v; want %v", got["aud"], test.wantAud)
			}
			exp, err := got.GetExpirationTime()
			if err != nil {
				t.Fatal(err)
			}
			iat, err := got.GetIssuedAt()
			if err != nil {
				t.Fatal(err)
			}
			life := exp.Sub(iat.Time)
			if life != test.wantLifetime {
				t.Errorf("lifetime = %v; want %v", life, test.wantLifetime)
			}
		})
	}
}

func TestJWTAuthClientInitialTokenFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"invalid_client"}`)) //nolint:errcheck // No point checking errors in test server.
	}))
	defer srv.Close()

	cfg := jwtAuthConfig{
		Mode:       jwtModeGrant,
		PrivateKey: testBoxPEMForConfig,
		Claims:     map[string]any{"iss": "me"},
		TokenURL:   srv.URL,
	}
	_, err := cfg.client(context.Background(), http.DefaultClient)
	if err == nil || !strings.Contains(err.Error(), "initial token") {
		t.Errorf("client() error = %v; want error containing %q", err, "initial token")
	}
}
