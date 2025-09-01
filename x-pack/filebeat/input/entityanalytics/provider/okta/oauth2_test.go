// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package okta

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestOAuth2ConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  *oAuth2Config
		wantErr bool
	}{
		{
			name: "valid oauth2 config with jwk_json",
			config: &oAuth2Config{
				Enabled:  boolPtr(true),
				ClientID: "test-client",
				Scopes:   []string{"okta.users.read"},
				TokenURL: "https://test.okta.com/oauth2/v1/token",
				OktaJWKJSON: []byte(`{
					"kty": "RSA",
					"n": "test-n",
					"e": "AQAB",
					"d": "test-d",
					"p": "test-p",
					"q": "test-q",
					"dp": "test-dp",
					"dq": "test-dq",
					"qi": "test-qi"
				}`),
			},
			wantErr: false,
		},
		{
			name: "valid oauth2 config with client secret",
			config: &oAuth2Config{
				Enabled:      boolPtr(true),
				ClientID:     "test-client",
				ClientSecret: "test-secret",
				Scopes:       []string{"okta.users.read"},
				TokenURL:     "https://test.okta.com/oauth2/v1/token",
			},
			wantErr: false,
		},
		{
			name: "invalid - missing client.id",
			config: &oAuth2Config{
				Enabled:     boolPtr(true),
				Scopes:      []string{"okta.users.read"},
				TokenURL:    "https://test.okta.com/oauth2/v1/token",
				OktaJWKJSON: []byte(`{"kty": "RSA"}`),
			},
			wantErr: true,
		},
		{
			name: "invalid - missing scopes",
			config: &oAuth2Config{
				Enabled:     boolPtr(true),
				ClientID:    "test-client",
				TokenURL:    "https://test.okta.com/oauth2/v1/token",
				OktaJWKJSON: []byte(`{"kty": "RSA"}`),
			},
			wantErr: true,
		},
		{
			name: "invalid - missing token_url",
			config: &oAuth2Config{
				Enabled:     boolPtr(true),
				ClientID:    "test-client",
				Scopes:      []string{"okta.users.read"},
				OktaJWKJSON: []byte(`{"kty": "RSA"}`),
			},
			wantErr: true,
		},
		{
			name: "invalid - mixing client secret and JWT keys",
			config: &oAuth2Config{
				Enabled:      boolPtr(true),
				ClientID:     "test-client",
				ClientSecret: "test-secret",
				Scopes:       []string{"okta.users.read"},
				TokenURL:     "https://test.okta.com/oauth2/v1/token",
				OktaJWKJSON:  []byte(`{"kty": "RSA"}`),
			},
			wantErr: true,
		},
		{
			name: "invalid - no authentication method provided",
			config: &oAuth2Config{
				Enabled:  boolPtr(true),
				ClientID: "test-client",
				Scopes:   []string{"okta.users.read"},
				TokenURL: "https://test.okta.com/oauth2/v1/token",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestOAuth2ConfigIsEnabled(t *testing.T) {
	tests := []struct {
		name   string
		config *oAuth2Config
		want   bool
	}{
		{
			name:   "nil config",
			config: nil,
			want:   false,
		},
		{
			name: "enabled explicitly",
			config: &oAuth2Config{
				Enabled: boolPtr(true),
			},
			want: true,
		},
		{
			name: "disabled explicitly",
			config: &oAuth2Config{
				Enabled: boolPtr(false),
			},
			want: false,
		},
		{
			name: "enabled by default",
			config: &oAuth2Config{
				Enabled: nil,
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.isEnabled()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestConfValidationWithOAuth2(t *testing.T) {
	tests := []struct {
		name    string
		config  conf
		wantErr bool
	}{
		{
			name: "valid oauth2 config",
			config: conf{
				OktaDomain: "test.okta.com",
				OAuth2: &oAuth2Config{
					Enabled:  boolPtr(true),
					ClientID: "test-client",
					Scopes:   []string{"okta.users.read"},
					TokenURL: "https://test.okta.com/oauth2/v1/token",
					OktaJWKJSON: []byte(`{
						"kty": "RSA",
						"kid": "test-key",
						"use": "sig",
						"alg": "RS256",
						"n": "test-n",
						"e": "AQAB",
						"d": "test-d",
						"p": "test-p",
						"q": "test-q",
						"dp": "test-dp",
						"dq": "test-dq",
						"qi": "test-qi"
					}`),
				},
				SyncInterval:   24 * time.Hour,
				UpdateInterval: 15 * time.Minute,
			},
			wantErr: false,
		},
		{
			name: "valid api token config",
			config: conf{
				OktaDomain:     "test.okta.com",
				OktaToken:      "test-token",
				SyncInterval:   24 * time.Hour,
				UpdateInterval: 15 * time.Minute,
			},
			wantErr: false,
		},
		{
			name: "no authentication method",
			config: conf{
				OktaDomain:     "test.okta.com",
				SyncInterval:   24 * time.Hour,
				UpdateInterval: 15 * time.Minute,
			},
			wantErr: true,
		},
		{
			name: "both authentication methods",
			config: conf{
				OktaDomain: "test.okta.com",
				OktaToken:  "test-token",
				OAuth2: &oAuth2Config{
					Enabled:  boolPtr(true),
					ClientID: "test-client",
					Scopes:   []string{"okta.users.read"},
					TokenURL: "https://test.okta.com/oauth2/v1/token",
					OktaJWKJSON: []byte(`{
						"kty": "RSA",
						"kid": "test-key",
						"use": "sig",
						"alg": "RS256",
						"n": "test-n",
						"e": "AQAB",
						"d": "test-d",
						"p": "test-p",
						"q": "test-q",
						"dp": "test-dp",
						"dq": "test-dq",
						"qi": "test-qi"
					}`),
				},
				SyncInterval:   24 * time.Hour,
				UpdateInterval: 15 * time.Minute,
			},
			wantErr: false, // OAuth2 takes precedence
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetAuthToken(t *testing.T) {
	tests := []struct {
		name     string
		config   conf
		expected string
	}{
		{
			name: "oauth2 enabled",
			config: conf{
				OAuth2: &oAuth2Config{
					Enabled: boolPtr(true),
				},
			},
			expected: "",
		},
		{
			name: "oauth2 disabled",
			config: conf{
				OktaToken: "test-token",
				OAuth2: &oAuth2Config{
					Enabled: boolPtr(false),
				},
			},
			expected: "test-token",
		},
		{
			name: "no oauth2 config",
			config: conf{
				OktaToken: "test-token",
			},
			expected: "test-token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &oktaInput{cfg: tt.config}
			got := p.getAuthToken()
			assert.Equal(t, tt.expected, got)
		})
	}
}

func boolPtr(b bool) *bool {
	return &b
}
