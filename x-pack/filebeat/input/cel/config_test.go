// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cel

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"golang.org/x/oauth2/google"

	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/transport/httpcommon"
)

func TestProviderCanonical(t *testing.T) {
	const (
		a oAuth2Provider = "gOoGle"
		b oAuth2Provider = "google"
	)
	if a.canonical() != b.canonical() {
		t.Errorf("%s and %s do not canonicalise to the same provider: %s != %s", a, b, a.canonical(), b.canonical())
	}
}

func TestGetProviderIsCanonical(t *testing.T) {
	const want oAuth2Provider = "google"
	got := oAuth2Config{Provider: "GOogle"}.getProvider()
	if got != want {
		t.Errorf("unexpected provider from getProvider: got:%s want:%s", got, want)
	}
}

func TestRegexpConfig(t *testing.T) {
	cfg := config{
		Interval: time.Minute,
		Program:  `{}`,
		Resource: &ResourceConfig{URL: &urlConfig{URL: &url.URL{}}},
		Regexps:  map[string]string{"regex_cve": `[Cc][Vv][Ee]-[0-9]{4}-[0-9]{4,7}`},
	}
	err := cfg.Validate()
	if err != nil {
		t.Errorf("failed to validate config with regexps: %v", err)
	}
}

func TestIsEnabled(t *testing.T) {
	type enabler interface {
		isEnabled() bool
		take(*bool)
	}
	for _, test := range []struct {
		name string
		auth enabler
	}{
		{name: "basic", auth: &basicAuthConfig{}},
		{name: "digest", auth: &digestAuthConfig{}},
		{name: "OAuth2", auth: &oAuth2Config{}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if !test.auth.isEnabled() {
				t.Errorf("auth not enabled by default")
			}

			var enabled bool
			for i := 0; i < 4; i++ {
				test.auth.take(&enabled)
				if got := test.auth.isEnabled(); got != enabled {
					t.Errorf("unexpected auth enabled state on iteration %d: got:%t want:%t", i, got, enabled)
				}
				enabled = !enabled
			}

			test.auth.take(nil)
			if !test.auth.isEnabled() {
				t.Errorf("auth not enabled if nilled")
			}
		})
	}
}

// take methods are for testing only.
func (b *basicAuthConfig) take(on *bool)  { b.Enabled = on }
func (d *digestAuthConfig) take(on *bool) { d.Enabled = on }
func (o *oAuth2Config) take(on *bool)     { o.Enabled = on }

func TestOAuth2GetTokenURL(t *testing.T) {
	const host = "http://localhost"
	for _, test := range []struct {
		tokenURL string
		provider oAuth2Provider
		tenentID string
		want     string
	}{
		{tokenURL: host, want: host},
		{tokenURL: host, provider: "azure", want: host},
		{provider: "azure", tenentID: "a_tenant_id", want: "https://login.microsoftonline.com/a_tenant_id/oauth2/v2.0/token"},
	} {
		oauth2 := oAuth2Config{TokenURL: test.tokenURL, Provider: test.provider, AzureTenantID: test.tenentID}
		got := oauth2.getTokenURL()
		if got != test.want {
			t.Errorf("unexpected token URL for %+v: got:%s want:%s", test, got, test.want)
		}
	}
}

func TestOAuth2GetEndpointParams(t *testing.T) {
	for _, test := range []struct {
		provider oAuth2Provider
		resource string
		params   url.Values
		want     url.Values
	}{
		{params: url.Values{"foo": {"bar"}}, want: url.Values{"foo": {"bar"}}},
		{provider: "azure", params: url.Values{"foo": {"bar"}}, want: url.Values{"foo": {"bar"}}},
		{provider: "azure", resource: "baz", params: url.Values{"foo": {"bar"}}, want: url.Values{"foo": {"bar"}, "resource": {"baz"}}},
	} {
		oauth2 := oAuth2Config{Provider: test.provider, EndpointParams: test.params, AzureResource: test.resource}
		got := oauth2.getEndpointParams()
		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("unexpected end point parameters for %+v: got:- want:+\n%s", test, cmp.Diff(got, test.want))
		}
	}
}

func TestConfigMustFailWithInvalidResource(t *testing.T) {
	for _, test := range []struct {
		val  string
		want error
	}{
		{val: ""},
		{val: "path/to/file"},
		{val: "::invalid::", want: errors.New(`parse "::invalid::": missing protocol scheme accessing 'resource.url'`)},
	} {
		m := map[string]interface{}{
			"resource.url": test.val,
		}
		cfg := conf.MustNewConfigFrom(m)
		conf := defaultConfig()
		conf.Program = "{}"     // Provide an empty program to avoid validation error from that.
		conf.Redact = &redact{} // Make sure we pass the redact requirement.
		err := cfg.Unpack(&conf)
		if fmt.Sprint(err) != fmt.Sprint(test.want) {
			t.Errorf("unexpected error return from Unpack: got:%v want:%v", err, test.want)
		}
	}
}

var oAuth2ValidationTests = []struct {
	name     string
	wantErr  error
	input    map[string]interface{}
	setup    func()
	teardown func()
}{
	{
		name:    "can't_set_oauth2_and_basic_auth_together",
		wantErr: errors.New("only one kind of auth can be enabled accessing 'auth'"),
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
		name:    "can't_set_oauth2_and_digest_auth_together",
		wantErr: errors.New("only one kind of auth can be enabled accessing 'auth'"),
		input: map[string]interface{}{
			"auth.digest.user":     "user",
			"auth.digest.password": "pass",
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
		name:    "can't_set_basic_and_digest_auth_together",
		wantErr: errors.New("only one kind of auth can be enabled accessing 'auth'"),
		input: map[string]interface{}{
			"auth.basic.user":      "user",
			"auth.basic.password":  "pass",
			"auth.digest.user":     "user",
			"auth.digest.password": "pass",
		},
	},
	{
		name: "can_set_oauth2_and_basic_auth_together_if_oauth2_is_disabled",
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
		name:    "token_url_and_client_credentials_must_be_set",
		wantErr: errors.New("both token_url and client credentials must be provided accessing 'auth.oauth2'"),
		input: map[string]interface{}{
			"auth.oauth2": map[string]interface{}{},
		},
	},
	{
		name: "client_credential_secret_may_be_empty",
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
		name:    "client_credential_secret_may_not_be_missing",
		wantErr: errors.New("both token_url and client credentials must be provided accessing 'auth.oauth2'"),
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
		name: "if_user_and_password_is_set_oauth2_must_use_user-password_authentication",
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
		name:    "if_user_is_set_password_credentials_must_be_set_for_user-password_authentication",
		wantErr: errors.New("both user and password credentials must be provided accessing 'auth.oauth2'"),
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
		name:    "if_password_is_set_user_credentials_must_be_set_for_user-password_authentication",
		wantErr: errors.New("both user and password credentials must be provided accessing 'auth.oauth2'"),
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
		name:    "must_fail_with_an_unknown_provider",
		wantErr: errors.New("unknown provider \"unknown\" accessing 'auth.oauth2'"),
		input: map[string]interface{}{
			"auth.oauth2": map[string]interface{}{
				"provider": "unknown",
			},
		},
	},
	{
		name:    "azure_must_have_either_tenant_id_or_token_url",
		wantErr: errors.New("at least one of token_url or tenant_id must be provided accessing 'auth.oauth2'"),
		input: map[string]interface{}{
			"auth.oauth2": map[string]interface{}{
				"provider": "azure",
			},
		},
	},
	{
		name:    "azure_must_have_only_one_of_token_url_and_tenant_id",
		wantErr: errors.New("only one of token_url and tenant_id can be used accessing 'auth.oauth2'"),
		input: map[string]interface{}{
			"auth.oauth2": map[string]interface{}{
				"provider":        "azure",
				"azure.tenant_id": "a_tenant_id",
				"token_url":       "localhost",
			},
		},
	},
	{
		name:    "azure_must_have_client_credentials_set",
		wantErr: errors.New("client credentials must be provided accessing 'auth.oauth2'"),
		input: map[string]interface{}{
			"auth.oauth2": map[string]interface{}{
				"provider":        "azure",
				"azure.tenant_id": "a_tenant_id",
			},
		},
	},
	{
		name: "azure_config_is_valid",
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
		name:    "google_can't_have_token_url_or_client_credentials_set",
		wantErr: errors.New("none of token_url and client credentials can be used, use google.credentials_file, google.jwt_file, google.credentials_json or ADC instead accessing 'auth.oauth2'"),
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
		name:    "google_must_fail_if_no_ADC_available",
		wantErr: errors.New("no authentication credentials were configured or detected (ADC) accessing 'auth.oauth2'"),
		input: map[string]interface{}{
			"auth.oauth2": map[string]interface{}{
				"provider": "google",
			},
		},
		setup: func() {
			// we change the default function to force a failure
			findDefaultGoogleCredentials = func(context.Context, ...string) (*google.Credentials, error) {
				return nil, errors.New("failed")
			}
		},
		teardown: func() { findDefaultGoogleCredentials = google.FindDefaultCredentials },
	},
	{
		name:    "google_must_fail_if_credentials_file_not_found",
		wantErr: errors.New("the file \"./wrong\" cannot be found accessing 'auth.oauth2'"),
		input: map[string]interface{}{
			"auth.oauth2": map[string]interface{}{
				"provider":                "google",
				"google.credentials_file": "./wrong",
			},
		},
	},
	{
		name:    "google_must_fail_if_ADC_is_wrongly_set",
		wantErr: errors.New("no authentication credentials were configured or detected (ADC) accessing 'auth.oauth2'"),
		input: map[string]interface{}{
			"auth.oauth2": map[string]interface{}{
				"provider": "google",
			},
		},
		setup: func() { os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "./wrong") },
	},
	{
		name: "google_must_work_if_ADC_is_set_up",
		input: map[string]interface{}{
			"auth.oauth2": map[string]interface{}{
				"provider": "google",
			},
		},
		setup: func() { os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "./testdata/credentials.json") },
	},
	{
		name: "google_must_work_if_credentials_file_is_correct",
		input: map[string]interface{}{
			"auth.oauth2": map[string]interface{}{
				"provider":                "google",
				"google.credentials_file": "./testdata/credentials.json",
			},
		},
	},
	{
		name: "google_must_work_if_jwt_file_is_correct",
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
		name: "google_must_work_if_credentials_json_is_correct",
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
		name:    "google_must_fail_if_credentials_json_is_not_a_valid_JSON",
		wantErr: errors.New("the field can't be converted to valid JSON accessing 'auth.oauth2.google.credentials_json'"),
		input: map[string]interface{}{
			"auth.oauth2": map[string]interface{}{
				"provider":                "google",
				"google.credentials_json": `invalid`,
			},
		},
	},
	{
		name:    "google must fail if jwt_json is not a valid JSON",
		wantErr: errors.New("the field can't be converted to valid JSON accessing 'auth.oauth2.google.jwt_json'"),
		input: map[string]interface{}{
			"auth.oauth2": map[string]interface{}{
				"provider":        "google",
				"google.jwt_json": `invalid`,
			},
		},
	},
	{
		name:    "google_must_fail_if_the_provided_credentials_file_is_not_a_valid_JSON",
		wantErr: errors.New("the file \"./testdata/invalid_credentials.json\" does not contain valid JSON accessing 'auth.oauth2'"),
		input: map[string]interface{}{
			"auth.oauth2": map[string]interface{}{
				"provider":                "google",
				"google.credentials_file": "./testdata/invalid_credentials.json",
			},
		},
	},
	{
		name:    "google_must_fail_if_the_delegated_account_is_set_without_jwt_file",
		wantErr: errors.New("google.delegated_account can only be provided with a jwt_file accessing 'auth.oauth2'"),
		input: map[string]interface{}{
			"auth.oauth2": map[string]interface{}{
				"provider":                 "google",
				"google.credentials_file":  "./testdata/credentials.json",
				"google.delegated_account": "delegated@account.com",
			},
		},
	},
	{
		name: "google_must_work_with_delegated_account_and_a_valid_jwt_file",
		input: map[string]interface{}{
			"auth.oauth2": map[string]interface{}{
				"provider":                 "google",
				"google.jwt_file":          "./testdata/credentials.json",
				"google.delegated_account": "delegated@account.com",
			},
		},
	},
	{
		name:    "unique_okta_jwk_token",
		wantErr: errors.New("okta validation error: one of okta.jwk_json, okta.jwk_file or okta.jwk_pem must be provided accessing 'auth.oauth2'"),
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
		name:    "invalid_okta_jwk_json",
		wantErr: errors.New("the field can't be converted to valid JSON accessing 'auth.oauth2.okta.jwk_json'"),
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
		name: "okta_successful_oauth2_validation",
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
		name: "okta_successful_pem_oauth2_validation",
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

func TestConfigOauth2Validation(t *testing.T) {
	for _, test := range oAuth2ValidationTests {
		t.Run(test.name, func(t *testing.T) {
			if test.setup != nil {
				test.setup()
			}
			if test.teardown != nil {
				defer test.teardown()
			}

			test.input["resource.url"] = "localhost"
			cfg := conf.MustNewConfigFrom(test.input)
			conf := defaultConfig()
			conf.Program = "{}"     // Provide an empty program to avoid validation error from that.
			conf.Redact = &redact{} // Make sure we pass the redact requirement.
			err := cfg.Unpack(&conf)

			if fmt.Sprint(err) != fmt.Sprint(test.wantErr) {
				t.Errorf("unexpected error return from Unpack: got:%v want:%v", err, test.wantErr)
			}
		})
	}
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
			"resource.keep_alive.disable": true,
		},
		want: httpcommon.WithKeepaliveSettings{Disable: true},
	},
	{
		name: "keep_alive_false",
		input: map[string]interface{}{
			"resource.keep_alive.disable": false,
		},
		want: httpcommon.WithKeepaliveSettings{Disable: false},
	},
	{
		name: "keep_alive_invalid_max",
		input: map[string]interface{}{
			"resource.keep_alive.disable":              false,
			"resource.keep_alive.max_idle_connections": -1,
		},
		wantErr: errors.New("max_idle_connections must not be negative accessing 'resource.keep_alive'"),
	},
}

func TestKeepAliveSetting(t *testing.T) {
	for _, test := range keepAliveTests {
		t.Run(test.name, func(t *testing.T) {
			test.input["resource.url"] = "localhost"
			cfg := conf.MustNewConfigFrom(test.input)
			conf := defaultConfig()
			conf.Program = "{}"     // Provide an empty program to avoid validation error from that.
			conf.Redact = &redact{} // Make sure we pass the redact requirement.
			err := cfg.Unpack(&conf)
			if fmt.Sprint(err) != fmt.Sprint(test.wantErr) {
				t.Errorf("unexpected error return from Unpack: got: %v want: %v", err, test.wantErr)
			}
			if err != nil {
				return
			}
			got := conf.Resource.KeepAlive.settings()
			if got != test.want {
				t.Errorf("unexpected setting for %s: got: %#v\nwant:%#v", test.name, got, test.want)
			}
		})
	}
}
