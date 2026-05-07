// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package okta

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/common"
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
				OktaJWKJSON: common.JSONBlob(`{
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
			name: "valid oauth2 config with jwk_pem",
			config: &oAuth2Config{
				Enabled:  boolPtr(true),
				ClientID: "test-client",
				Scopes:   []string{"okta.users.read"},
				TokenURL: "https://test.okta.com/oauth2/v1/token",
				OktaJWKPEM: `
-----BEGIN PRIVATE KEY-----
MIIEvAIBADANBgkqhkiG9w0BAQEFAASCBKYwggSiAgEAAoIBAQCOuef3HMRhohVT
5kSoAJgV+atpDjkwTwkOq+ImnbBlv75GaApG90w8VpjXjhqN/1KJmwfyrKiquiMq
OPu+o/672Dys5rUAaWSbT7wRF1GjLDDZrM0GHRdV4DGxM/LKI8I5yE1Mx3EzV+D5
ZLmcRc5U4oEoMwtGpr0zRZ7uUr6a28UQwcUsVIPItc1/9rERlo1WTv8dcaj4ECC3
2Sc0y/F+9XqwJvLd4Uv6ckzP0Sv4tbDA+7jpD9MneAIUiZ4LVj2cwbBd+YRY6jXx
MkevcCSmSX60clBY1cIFkw1DYHqtdHEwAQcQHLGMoi72xRP2qrdzIPsaTKVYoHVo
WA9vADdHAgMBAAECggEAIlx7jjCsztyYyeQsL05FTzUWoWo9NnYwtgmHnshkCXsK
MiUmJEOxZO1sSqj5l6oakupyFWigCspZYPbrFNCiqVK7+NxqQzkccY/WtT6p9uDS
ufUyPwCN96zMCd952lSVlBe3FH8Hr9a+YQxw60CbFjCZ67WuR0opTsi6JKJjJSDb
TQQZ4qJR97D05I1TgfmO+VO7G/0/dDaNHnnlYz0AnOgZPSyvrU2G5cYye4842EMB
ng81xjHD+xp55JNui/xYkhmYspYhrB2KlEjkKb08OInUjBeaLEAgA1r9yOHsfV/3
DQzDPRO9iuqx5BfJhdIqUB1aifrye+sbxt9uMBtUgQKBgQDVdfO3GYT+ZycOQG9P
QtdMn6uiSddchVCGFpk331u6M6yafCKjI/MlJDl29B+8R5sVsttwo8/qnV/xd3cn
pY14HpKAsE4l6/Ciagzoj+0NqfPEDhEzbo8CyArcd7pSxt3XxECAfZe2+xivEPHe
gFO60vSFjFtvlLRMDMOmqX3kYQKBgQCrK1DISyQTnD6/axsgh2/ESOmT7n+JRMx/
YzA7Lxu3zGzUC8/sRDa1C41t054nf5ZXJueYLDSc4kEAPddzISuCLxFiTD2FQ75P
lHWMgsEzQObDm4GPE9cdKOjoAvtAJwbvZcjDa029CDx7aCaDzbNvdmplZ7EUrznR
55U8Wsm8pwKBgBytxTmzZwfbCgdDJvFKNKzpwuCB9TpL+v6Y6Kr2Clfg+26iAPFU
MiWqUUInGGBuamqm5g6jI5sM28gQWeTsvC4IRXyes1Eq+uCHSQax15J/Y+3SSgNT
9kjUYYkvWMwoRcPobRYWSZze7XkP2L8hFJ7EGvAaZGqAWxzgliS9HtnhAoGAONZ/
UqMw7Zoac/Ga5mhSwrj7ZvXxP6Gqzjofj+eKqrOlB5yMhIX6LJATfH6iq7cAMxxm
Fu/G4Ll4oB3o5wACtI3wldV/MDtYfJBtoCTjBqPsfNOsZ9hMvBATlsc2qwzKjsAb
tFhzTevoOYpSD75EcSS/G8Ec2iN9bagatBnpl00CgYBVqAOFZelNfP7dj//lpk8y
EUAw7ABOq0S9wkpFWTXIVPoBQUipm3iAUqGNPmvr/9ShdZC9xeu5AwKram4caMWJ
ExRhcDP1hFM6CdmSkIYEgBKvN9N0O4Lx1ba34gk74Hm65KXxokjJHOC0plO7c7ok
LNV/bIgMHOMoxiGrwyjAhg==
-----END PRIVATE KEY-----
`,
			},
			wantErr: false,
		},
		{
			name: "invalid - bad jwk_pem data",
			config: &oAuth2Config{
				Enabled:    boolPtr(true),
				ClientID:   "test-client",
				Scopes:     []string{"okta.users.read"},
				TokenURL:   "https://test.okta.com/oauth2/v1/token",
				OktaJWKPEM: "not-valid-pem-data",
			},
			wantErr: true,
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
				OktaJWKJSON: common.JSONBlob(`{"kty": "RSA"}`),
			},
			wantErr: true,
		},
		{
			name: "invalid - missing scopes",
			config: &oAuth2Config{
				Enabled:     boolPtr(true),
				ClientID:    "test-client",
				TokenURL:    "https://test.okta.com/oauth2/v1/token",
				OktaJWKJSON: common.JSONBlob(`{"kty": "RSA"}`),
			},
			wantErr: true,
		},
		{
			name: "invalid - missing token_url",
			config: &oAuth2Config{
				Enabled:     boolPtr(true),
				ClientID:    "test-client",
				Scopes:      []string{"okta.users.read"},
				OktaJWKJSON: common.JSONBlob(`{"kty": "RSA"}`),
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
				OktaJWKJSON:  common.JSONBlob(`{"kty": "RSA"}`),
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
					OktaJWKJSON: common.JSONBlob(`{
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
					OktaJWKJSON: common.JSONBlob(`{
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
