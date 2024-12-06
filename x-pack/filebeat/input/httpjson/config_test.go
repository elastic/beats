// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpjson

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2/google"

	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/transport/httpcommon"
)

func TestProviderCanonical(t *testing.T) {
	const (
		a oAuth2Provider = "gOoGle"
		b oAuth2Provider = "google"
	)

	assert.Equal(t, a.canonical(), b.canonical())
}

func TestGetProviderIsCanonical(t *testing.T) {
	const expected oAuth2Provider = "google"

	oauth2 := oAuth2Config{Provider: "GOogle"}
	assert.Equal(t, expected, oauth2.getProvider())
}

func TestIsEnabled(t *testing.T) {
	oauth2 := oAuth2Config{}
	if !oauth2.isEnabled() {
		t.Fatal("OAuth2 should be enabled by default")
	}

	enabled := false
	oauth2.Enabled = &enabled

	assert.False(t, oauth2.isEnabled())

	enabled = true

	assert.True(t, oauth2.isEnabled())
}

func TestGetTokenURL(t *testing.T) {
	const expected = "http://localhost"
	oauth2 := oAuth2Config{TokenURL: "http://localhost"}
	assert.Equal(t, expected, oauth2.getTokenURL())
}

func TestGetTokenURLWithAzure(t *testing.T) {
	const expectedWithoutTenantID = "http://localhost"
	oauth2 := oAuth2Config{TokenURL: "http://localhost", Provider: "azure"}

	assert.Equal(t, expectedWithoutTenantID, oauth2.getTokenURL())

	oauth2.TokenURL = ""
	oauth2.AzureTenantID = "a_tenant_id"
	const expectedWithTenantID = "https://login.microsoftonline.com/a_tenant_id/oauth2/v2.0/token"

	assert.Equal(t, expectedWithTenantID, oauth2.getTokenURL())
}

func TestGetEndpointParams(t *testing.T) {
	expected := map[string][]string{"foo": {"bar"}}
	oauth2 := oAuth2Config{EndpointParams: map[string][]string{"foo": {"bar"}}}
	assert.Equal(t, expected, oauth2.getEndpointParams())
}

func TestGetEndpointParamsWithAzure(t *testing.T) {
	expectedWithoutResource := map[string][]string{"foo": {"bar"}}
	oauth2 := oAuth2Config{Provider: "azure", EndpointParams: map[string][]string{"foo": {"bar"}}}

	assert.Equal(t, expectedWithoutResource, oauth2.getEndpointParams())

	oauth2.AzureResource = "baz"
	expectedWithResource := map[string][]string{"foo": {"bar"}, "resource": {"baz"}}

	assert.Equal(t, expectedWithResource, oauth2.getEndpointParams())
}

func TestConfigFailsWithInvalidMethod(t *testing.T) {
	m := map[string]interface{}{
		"request.method": "DELETE",
	}
	cfg := conf.MustNewConfigFrom(m)
	conf := defaultConfig()
	if err := cfg.Unpack(&conf); err == nil {
		t.Fatal("Configuration validation failed. http_method DELETE is not allowed.")
	}
}

func TestConfigMustFailWithInvalidURL(t *testing.T) {
	m := map[string]interface{}{
		"request.url": "::invalid::",
	}
	cfg := conf.MustNewConfigFrom(m)
	conf := defaultConfig()
	err := cfg.Unpack(&conf)
	assert.EqualError(t, err, `parse "::invalid::": missing protocol scheme accessing 'request.url'`)
}

func TestConfigOauth2Validation(t *testing.T) {
	cases := []struct {
		name        string
		expectedErr string
		input       map[string]interface{}
		setup       func()
		teardown    func()
	}{
		{
			name:        "can't set oauth2 and basic auth together",
			expectedErr: "only one kind of auth can be enabled accessing 'auth'",
			input: map[string]interface{}{
				"auth.basic.user":     "user",
				"auth.basic.password": "pass",
				"auth.oauth2": map[string]interface{}{
					"token_url": "localhost",
					"client": map[string]interface{}{
						"id":     "a_client_id",
						"secret": "a_client_secret",
					},
				},
			},
		},
		{
			name: "can set oauth2 and basic auth together if oauth2 is disabled",
			input: map[string]interface{}{
				"auth.basic.user":     "user",
				"auth.basic.password": "pass",
				"auth.oauth2": map[string]interface{}{
					"enabled":   false,
					"token_url": "localhost",
					"client": map[string]interface{}{
						"id":     "a_client_id",
						"secret": "a_client_secret",
					},
				},
			},
		},
		{
			name:        "token_url and client credentials must be set",
			expectedErr: "both token_url and client credentials must be provided accessing 'auth.oauth2'",
			input: map[string]interface{}{
				"auth.oauth2": map[string]interface{}{},
			},
		},
		{
			name: "client credential secret may be empty",
			input: map[string]interface{}{
				"auth.oauth2": map[string]interface{}{
					"enabled":   true,
					"token_url": "localhost",
					"client": map[string]interface{}{
						"id":     "a_client_id",
						"secret": "",
					},
				},
			},
		},
		{
			name:        "client credential secret may not be missing",
			expectedErr: "both token_url and client credentials must be provided accessing 'auth.oauth2'",
			input: map[string]interface{}{
				"auth.oauth2": map[string]interface{}{
					"enabled":   true,
					"token_url": "localhost",
					"client": map[string]interface{}{
						"id": "a_client_id",
					},
				},
			},
		},
		{
			name: "if user and password is set oauth2 must use user-password authentication",
			input: map[string]interface{}{
				"auth.oauth2": map[string]interface{}{
					"user":      "a_client_user",
					"password":  "a_client_password",
					"token_url": "localhost",
					"client": map[string]interface{}{
						"id":     "a_client_id",
						"secret": "a_client_secret",
					},
				},
			},
		},
		{
			name:        "if user is set password credentials must be set for user-password authentication",
			expectedErr: "both user and password credentials must be provided accessing 'auth.oauth2'",
			input: map[string]interface{}{
				"auth.oauth2": map[string]interface{}{
					"user":      "a_client_user",
					"token_url": "localhost",
					"client": map[string]interface{}{
						"id":     "a_client_id",
						"secret": "a_client_secret",
					},
				},
			},
		},
		{
			name:        "if password is set user credentials must be set for user-password authentication",
			expectedErr: "both user and password credentials must be provided accessing 'auth.oauth2'",
			input: map[string]interface{}{
				"auth.oauth2": map[string]interface{}{
					"password":  "a_client_password",
					"token_url": "localhost",
					"client": map[string]interface{}{
						"id":     "a_client_id",
						"secret": "a_client_secret",
					},
				},
			},
		},
		{
			name: "if password is set credentials may be missing for user-password authentication",
			input: map[string]interface{}{
				"auth.oauth2": map[string]interface{}{
					"user":      "a_client_user",
					"password":  "a_client_password",
					"token_url": "localhost",
				},
			},
		},
		{
			name:        "must fail with an unknown provider",
			expectedErr: "unknown provider \"unknown\" accessing 'auth.oauth2'",
			input: map[string]interface{}{
				"auth.oauth2": map[string]interface{}{
					"provider": "unknown",
				},
			},
		},
		{
			name:        "azure must have either tenant_id or token_url",
			expectedErr: "at least one of token_url or tenant_id must be provided accessing 'auth.oauth2'",
			input: map[string]interface{}{
				"auth.oauth2": map[string]interface{}{
					"provider": "azure",
				},
			},
		},
		{
			name:        "azure must have only one of token_url and tenant_id",
			expectedErr: "only one of token_url and tenant_id can be used accessing 'auth.oauth2'",
			input: map[string]interface{}{
				"auth.oauth2": map[string]interface{}{
					"provider":        "azure",
					"azure.tenant_id": "a_tenant_id",
					"token_url":       "localhost",
				},
			},
		},
		{
			name:        "azure must have client credentials set",
			expectedErr: "client credentials must be provided accessing 'auth.oauth2'",
			input: map[string]interface{}{
				"auth.oauth2": map[string]interface{}{
					"provider":        "azure",
					"azure.tenant_id": "a_tenant_id",
				},
			},
		},
		{
			name: "azure config is valid",
			input: map[string]interface{}{
				"auth.oauth2": map[string]interface{}{
					"provider": "azure",
					"azure": map[string]interface{}{
						"tenant_id": "a_tenant_id",
					},
					"client.id":     "a_client_id",
					"client.secret": "a_client_secret",
				},
			},
		},
		{
			name:        "google can't have token_url or client credentials set",
			expectedErr: "none of token_url and client credentials can be used, use google.credentials_file, google.jwt_file, google.credentials_json or ADC instead accessing 'auth.oauth2'",
			input: map[string]interface{}{
				"auth.oauth2": map[string]interface{}{
					"provider": "google",
					"azure": map[string]interface{}{
						"tenant_id": "a_tenant_id",
					},
					"client.id":     "a_client_id",
					"client.secret": "a_client_secret",
					"token_url":     "localhost",
				},
			},
		},
		{
			name:        "google must fail if no ADC available",
			expectedErr: "no authentication credentials were configured or detected (ADC) accessing 'auth.oauth2'",
			input: map[string]interface{}{
				"auth.oauth2": map[string]interface{}{
					"provider": "google",
				},
			},
			setup: func() {
				// we change the default function to force a failure
				findDefaultGoogleCredentials = func(context.Context, google.CredentialsParams) (*google.Credentials, error) {
					return nil, errors.New("failed")
				}
			},
			teardown: func() { findDefaultGoogleCredentials = google.FindDefaultCredentialsWithParams },
		},
		{
			name: "google must send scopes and delegated_account if ADC available",
			input: map[string]interface{}{
				"auth.oauth2": map[string]interface{}{
					"provider":                 "google",
					"google.delegated_account": "delegated@account.com",
					"scopes":                   []string{"foo"},
				},
			},
			setup: func() {
				findDefaultGoogleCredentials = func(_ context.Context, p google.CredentialsParams) (*google.Credentials, error) {
					if len(p.Scopes) != 1 || p.Scopes[0] != "foo" || p.Subject != "delegated@account.com" {
						return nil, errors.New("failed")
					}
					return &google.Credentials{}, nil
				}
			},
			teardown: func() { findDefaultGoogleCredentials = google.FindDefaultCredentialsWithParams },
		},
		{
			name:        "google must fail if credentials file not found",
			expectedErr: "the file \"./wrong\" cannot be found accessing 'auth.oauth2'",
			input: map[string]interface{}{
				"auth.oauth2": map[string]interface{}{
					"provider":                "google",
					"google.credentials_file": "./wrong",
				},
			},
		},
		{
			name:        "google must fail if ADC is wrongly set",
			expectedErr: "no authentication credentials were configured or detected (ADC) accessing 'auth.oauth2'",
			input: map[string]interface{}{
				"auth.oauth2": map[string]interface{}{
					"provider": "google",
				},
			},
			setup: func() { os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "./wrong") },
		},
		{
			name: "google must work if ADC is set up",
			input: map[string]interface{}{
				"auth.oauth2": map[string]interface{}{
					"provider": "google",
				},
			},
			setup: func() { os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "./testdata/credentials.json") },
		},
		{
			name: "google must work if credentials_file is correct",
			input: map[string]interface{}{
				"auth.oauth2": map[string]interface{}{
					"provider":                "google",
					"google.credentials_file": "./testdata/credentials.json",
				},
			},
		},
		{
			name: "google must work if jwt_file is correct",
			input: map[string]interface{}{
				"auth.oauth2": map[string]interface{}{
					"provider":        "google",
					"google.jwt_file": "./testdata/credentials.json",
				},
			},
		},
		{
			name: "google must work if jwt_json is correct",
			input: map[string]interface{}{
				"auth.oauth2": map[string]interface{}{
					"provider": "google",
					"google.jwt_json": `{
						"type":           "service_account",
						"project_id":     "foo",
						"private_key_id": "x",
						"client_email":   "foo@bar.com",
						"client_id":      "0"
					}`,
				},
			},
		},
		{
			name: "google must work if credentials_json is correct",
			input: map[string]interface{}{
				"auth.oauth2": map[string]interface{}{
					"provider": "google",
					"google.credentials_json": `{
						"type":           "service_account",
						"project_id":     "foo",
						"private_key_id": "x",
						"client_email":   "foo@bar.com",
						"client_id":      "0"
					}`,
				},
			},
		},
		{
			name:        "google must fail if credentials_json is not a valid JSON",
			expectedErr: "the field can't be converted to valid JSON accessing 'auth.oauth2.google.credentials_json'",
			input: map[string]interface{}{
				"auth.oauth2": map[string]interface{}{
					"provider":                "google",
					"google.credentials_json": `invalid`,
				},
			},
		},
		{
			name:        "google must fail if jwt_json is not a valid JSON",
			expectedErr: "the field can't be converted to valid JSON accessing 'auth.oauth2.google.jwt_json'",
			input: map[string]interface{}{
				"auth.oauth2": map[string]interface{}{
					"provider":        "google",
					"google.jwt_json": `invalid`,
				},
			},
		},
		{
			name:        "google must fail if the provided credentials file is not a valid JSON",
			expectedErr: "the file \"./testdata/invalid_credentials.json\" does not contain valid JSON accessing 'auth.oauth2'",
			input: map[string]interface{}{
				"auth.oauth2": map[string]interface{}{
					"provider":                "google",
					"google.credentials_file": "./testdata/invalid_credentials.json",
				},
			},
		},
		{
			name:        "google must fail if the delegated_account is set without jwt_file",
			expectedErr: "google.delegated_account can only be provided with a jwt_file accessing 'auth.oauth2'",
			input: map[string]interface{}{
				"auth.oauth2": map[string]interface{}{
					"provider":                 "google",
					"google.credentials_file":  "./testdata/credentials.json",
					"google.delegated_account": "delegated@account.com",
				},
			},
		},
		{
			name: "google must work with delegated_account and a valid jwt_file",
			input: map[string]interface{}{
				"auth.oauth2": map[string]interface{}{
					"provider":                 "google",
					"google.jwt_file":          "./testdata/credentials.json",
					"google.delegated_account": "delegated@account.com",
				},
			},
		},
		{
			name: "google must work with delegated_account and ADC set up",
			input: map[string]interface{}{
				"auth.oauth2": map[string]interface{}{
					"provider":                 "google",
					"google.delegated_account": "delegated@account.com",
				},
			},
			setup: func() { os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "./testdata/credentials.json") },
		},
		{
			name:        "okta requires token_url, client_id, scopes and at least one of okta.jwk_json or okta.jwk_file to be provided",
			expectedErr: "okta validation error: one of okta.jwk_json, okta.jwk_file or okta.jwk_pem must be provided accessing 'auth.oauth2'",
			input: map[string]interface{}{
				"auth.oauth2": map[string]interface{}{
					"provider":  "okta",
					"client.id": "a_client_id",
					"token_url": "localhost",
					"scopes":    []string{"foo"},
				},
			},
		},
		{
			name:        "okta oauth2 validation fails if jwk_json is not a valid JSON",
			expectedErr: "the field can't be converted to valid JSON accessing 'auth.oauth2.okta.jwk_json'",
			input: map[string]interface{}{
				"auth.oauth2": map[string]interface{}{
					"provider":      "okta",
					"client.id":     "a_client_id",
					"token_url":     "localhost",
					"scopes":        []string{"foo"},
					"okta.jwk_json": `"p":"x","kty":"RSA","q":"x","d":"x","e":"x","use":"x","kid":"x","qi":"x","dp":"x","alg":"x","dq":"x","n":"x"}`,
				},
			},
		},
		{
			name: "okta successful oauth2 validation",
			input: map[string]interface{}{
				"auth.oauth2": map[string]interface{}{
					"provider":      "okta",
					"client.id":     "a_client_id",
					"token_url":     "localhost",
					"scopes":        []string{"foo"},
					"okta.jwk_json": `{"p":"x","kty":"RSA","q":"x","d":"x","e":"x","use":"x","kid":"x","qi":"x","dp":"x","alg":"x","dq":"x","n":"x"}`,
				},
			},
		},
		{
			name: "okta successful pem oauth2 validation",
			input: map[string]interface{}{
				"auth.oauth2": map[string]interface{}{
					"provider":  "okta",
					"client.id": "a_client_id",
					"token_url": "localhost",
					"scopes":    []string{"foo"},
					"okta.jwk_pem": `
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
			},
		},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			if c.setup != nil {
				c.setup()
			}

			if c.teardown != nil {
				defer c.teardown()
			}

			c.input["request.url"] = "localhost"
			cfg := conf.MustNewConfigFrom(c.input)
			conf := defaultConfig()
			err := cfg.Unpack(&conf)

			switch {
			case c.expectedErr == "":
				if err != nil {
					t.Fatalf("Configuration validation failed. no error expected but got %q", err)
				}

			case c.expectedErr != "":
				if err == nil || err.Error() != c.expectedErr {
					t.Fatalf("Configuration validation failed. expecting %q error but got %q", c.expectedErr, err)
				}
			}
		})
	}
}

func TestCursorEntryConfig(t *testing.T) {
	in := map[string]interface{}{
		"entry1": map[string]interface{}{
			"ignore_empty_value": true,
		},
		"entry2": map[string]interface{}{
			"ignore_empty_value": false,
		},
		"entry3": map[string]interface{}{
			"ignore_empty_value": nil,
		},
		"entry4": map[string]interface{}{},
	}
	cfg := conf.MustNewConfigFrom(in)
	conf := cursorConfig{}
	require.NoError(t, cfg.Unpack(&conf))
	assert.True(t, conf["entry1"].mustIgnoreEmptyValue())
	assert.False(t, conf["entry2"].mustIgnoreEmptyValue())
	assert.True(t, conf["entry3"].mustIgnoreEmptyValue())
	assert.True(t, conf["entry4"].mustIgnoreEmptyValue())
}

var keepAliveTests = []struct {
	name    string
	input   map[string]interface{}
	want    httpcommon.WithKeepaliveSettings
	wantErr error
}{
	{
		name:  "keep_alive_none", // Default to the old behaviour of true.
		input: map[string]interface{}{},
		want:  httpcommon.WithKeepaliveSettings{Disable: true},
	},
	{
		name: "keep_alive_true",
		input: map[string]interface{}{
			"request.keep_alive.disable": true,
		},
		want: httpcommon.WithKeepaliveSettings{Disable: true},
	},
	{
		name: "keep_alive_false",
		input: map[string]interface{}{
			"request.keep_alive.disable": false,
		},
		want: httpcommon.WithKeepaliveSettings{Disable: false},
	},
	{
		name: "keep_alive_invalid_max",
		input: map[string]interface{}{
			"request.keep_alive.disable":              false,
			"request.keep_alive.max_idle_connections": -1,
		},
		wantErr: errors.New("max_idle_connections must not be negative accessing 'request.keep_alive'"),
	},
}

func TestKeepAliveSetting(t *testing.T) {
	for _, test := range keepAliveTests {
		t.Run(test.name, func(t *testing.T) {
			test.input["request.url"] = "localhost"
			cfg := conf.MustNewConfigFrom(test.input)
			conf := defaultConfig()
			err := cfg.Unpack(&conf)
			if fmt.Sprint(err) != fmt.Sprint(test.wantErr) {
				t.Errorf("unexpected error return from Unpack: got: %q want: %q", err, test.wantErr)
			}
			if err != nil {
				return
			}
			got := conf.Request.KeepAlive.settings()
			if got != test.want {
				t.Errorf("unexpected setting for %s: got: %#v\nwant:%#v", test.name, got, test.want)
			}
		})
	}
}
