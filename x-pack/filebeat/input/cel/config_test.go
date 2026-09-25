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
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"golang.org/x/oauth2/google"

	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/transport/httpcommon"
)

// testBoxPEMForConfig is the plain testBoxPlainKey reformatted for use in
// oAuth2ValidationTests table entries, where it appears as a raw string.
const testBoxPEMForConfig = `-----BEGIN PRIVATE KEY-----
MIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQCzy2cU1+L8Tpht
Lg9q9nCNqy3pXQQAiuWhGRtBS9pmRo/xl+tlVUeJxN8AVdBjJGjra6pZZ6g8x+NW
tUfuOXNgRa8NUVQIRNkspSQVlb00DD8LQetehnSRQV6I0CY2g4xmobw9246HFIlO
64/KhBF0BxnYx45rJ7CHEuo61p97dzGbvq0yf3aUMKo/NkqxgSLi5dKeOBuolHJb
k/fbRGaXG0YDqpz0EjxnSRZ8PNw35JSNjEpiRhxcLEJo002V+x/SrS/w7peGjdFN
axe47SoKSF2SErtLYxciGEY5Cn3E1A/8DGFXVsIQx987+nzfZJU2Yos/2iHwA/+N
O+u2zGYlAgMBAAECggEACK1pdTUSLHEypBpT/iqUthGr7pZhqhEKEiNfEGCz0rnX
GqblYoeiI0EQLjj2DMLmGW6h0xzQntZa34VySkoVinDyiOcC8j84aBI0UqJedlOc
+1McI/zDRXttL5c0MO9aaF2n8yhUkappEhkGYJTNLtdk5PSEqCFLQMml6l8PZWr/
j8tbbIuNudzpBR8H4/agc0UFZwLK1YpCF9vD9oRZ+HZSum5sM7Hg4UV4UqZUmyaE
Fqdy82P1bw9W2mJ171yYsTP7UzQo6eJ7H0Mdo5AYmReNXLIgE0rPtc1S3KIxWvVl
0k6kp1P0ria11qzjpIuzYZkzE1cLZfuuaVinoznWoQKBgQDRPazPohVCdbc3frT1
a9hBotAAAKg0Mc7eRIb9JqPPj2p5eUanpZgWaQ1To+I/uN+i7EWiCHAAeZhzj8wm
O8H+QEo3jUntIam7TNdEW/TC5Uex4GKshA2EaNekoFudzAcmgrzi3CLsF3e9Ks3a
iOE/1io3NwcAvYb/z+m9tvJrPQKBgQDb+SY+Xwu6wzVPTK6TRPhoBFEDasyzElKt
5i1Jg6K8W0YHe1L9y5jwPVXGPBhN3loRqrO1zJxVwK8CbR13VNagvPw4UEKkkCjh
fuYlNx0ifukAK8fKWqWJm/NyFy5uPqsEh8KrPbkONzKC6oJRxYd3lmZoOYnSr+6u
jwU2BY81CQKBgC7xRkbi1yAs5qjlnVV+F2tKSp3lh9cF4aJN/3bl51RWmY2dHrPX
29ITSXEdUFH5ePrFRS3/9Ji2rvQmK6fcOj5/T+c8pHw11C14JMdqVfQvmjEW5SxN
B/dPyild7I/vSR9jr1q6Bn+vGCbxZnODx/0ZYCk5CDIrUxErJQZx99sFAoGBALGW
co6WExUjNa2gnavdWaI4IeNdXIcROtiT5GneMQpZsa6mnHiy3vTMv6u7pm9vHE34
/v69glUkquWNi+VkA6ZfDEy2VyceDzMFTO4skYPg62Cs963hApWW5rJsDpsIUu7k
X3/546WbYFca1j0H+HbOYDyyfxct28bnRfC4CkZpAoGAeEZ4ZPh2ufpMzKgwWK/U
5lXA5QVH167KYRJR5ptH3GjNjBc6jmsHqb24Dnx9y+7jmVDGSWhRvjhbpj9mT19m
yzbNC9wNkT3IGwoEu1C3SbsstDnPmDILXZXoGODn5jYWYKXwUPRmakkkFpqdkQ4f
A7JN9GvuuK/jTvNqCQDmIb0=
-----END PRIVATE KEY-----
`

// boxTestKeyOneLiner is testBoxPEMForConfig with newlines escaped as \n so
// the key can be embedded directly in a JSON string literal.
var boxTestKeyOneLiner = strings.ReplaceAll(strings.TrimSpace(testBoxPEMForConfig), "\n", `\n`)

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

func TestSecretStateUnpack(t *testing.T) {
	t.Run("map", func(t *testing.T) {
		var s secretState
		err := s.Unpack(map[string]any{"api_key": "secret"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if s.m["api_key"] != "secret" {
			t.Fatalf("unexpected value: got %v, want %q", s.m["api_key"], "secret")
		}
	})

	t.Run("string", func(t *testing.T) {
		var s secretState
		err := s.Unpack("api_key: secret")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if s.m["api_key"] != "secret" {
			t.Fatalf("unexpected value: got %v, want %q", s.m["api_key"], "secret")
		}
	})

	t.Run("nested_string", func(t *testing.T) {
		var s secretState
		err := s.Unpack("headers:\n  auth: token\n  key: secret")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		headers, ok := s.m["headers"].(map[string]any)
		if !ok {
			t.Fatalf("expected map[string]interface{} for nested map, got %T", s.m["headers"])
		}
		if headers["auth"] != "token" {
			t.Fatalf("unexpected value: got %v, want %q", headers["auth"], "token")
		}
	})

	t.Run("invalid_type", func(t *testing.T) {
		var s secretState
		err := s.Unpack(42)
		if err == nil {
			t.Fatal("expected error for int value")
		}
	})

	t.Run("invalid_yaml", func(t *testing.T) {
		var s secretState
		err := s.Unpack(":\ninvalid:\n  :\n")
		if err == nil {
			t.Fatal("expected error for invalid YAML")
		}
	})
}

func TestSecretStateValidation(t *testing.T) {
	base := config{
		Interval: time.Minute,
		Program:  `{}`,
		Resource: &ResourceConfig{URL: &urlConfig{URL: &url.URL{}}},
	}

	t.Run("state_with_secret_key_rejected", func(t *testing.T) {
		cfg := base
		cfg.State = map[string]any{
			"secret": map[string]any{"api_key": "hidden"},
		}
		err := cfg.Validate()
		if err == nil {
			t.Fatal("expected error for state containing \"secret\" key")
		}
		const want = `state must not contain a "secret" key`
		if !strings.HasPrefix(err.Error(), want) {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("state_with_secret_key_rejected_without_secret_state", func(t *testing.T) {
		cfg := base
		cfg.State = map[string]any{
			"secret": "value",
		}
		err := cfg.Validate()
		if err == nil {
			t.Fatal("expected error for state containing \"secret\" key even without secret_state")
		}
	})

	t.Run("state_without_secret_key_accepted", func(t *testing.T) {
		cfg := base
		cfg.State = map[string]any{
			"token": "not_actually_secret",
		}
		err := cfg.Validate()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("nil_state_accepted", func(t *testing.T) {
		cfg := base
		err := cfg.Validate()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
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
		{name: "file", auth: &fileAuthConfig{}},
		{name: "OAuth2", auth: &oAuth2Config{}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if !test.auth.isEnabled() {
				t.Errorf("auth not enabled by default")
			}

			var enabled bool
			for i := range 4 {
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
func (f *fileAuthConfig) take(on *bool)   { f.Enabled = on }
func (o *oAuth2Config) take(on *bool)     { o.Enabled = on }

func TestFileAuthConfigValidate(t *testing.T) {
	t.Run("requires path", func(t *testing.T) {
		cfg := &fileAuthConfig{}
		if err := cfg.Validate(); err == nil || err.Error() != "path must be set" {
			t.Fatalf("expected path requirement error, got: %v", err)
		}
	})

	t.Run("requires positive refresh interval", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "token")
		if err := os.WriteFile(path, []byte("secret"), 0o600); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}
		zero := time.Duration(0)
		cfg := &fileAuthConfig{Path: path, RefreshInterval: &zero}
		if err := cfg.Validate(); err == nil || err.Error() != "refresh_interval must be greater than 0" {
			t.Fatalf("expected refresh interval error, got: %v", err)
		}
	})

	t.Run("valid configuration", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "token")
		if err := os.WriteFile(path, []byte("secret"), 0o600); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}
		refresh := time.Second
		cfg := &fileAuthConfig{Path: path, RefreshInterval: &refresh}
		if err := cfg.Validate(); err != nil {
			t.Fatalf("unexpected validation error: %v", err)
		}
	})
}

func TestFileAuthConfigDefaults(t *testing.T) {
	cfg := &fileAuthConfig{}
	if got := cfg.headerName(); got != defaultFileAuthHeader {
		t.Fatalf("unexpected default header: got %q want %q", got, defaultFileAuthHeader)
	}
	if got := cfg.refreshInterval(); got != defaultFileAuthRefreshInterval {
		t.Fatalf("unexpected default refresh interval: got %v want %v", got, defaultFileAuthRefreshInterval)
	}

	header := "X-Api-Key"
	cfg.Header = header
	if got := cfg.headerName(); got != header {
		t.Fatalf("unexpected header override: got %q want %q", got, header)
	}

	refresh := 42 * time.Second
	cfg.RefreshInterval = &refresh
	if got := cfg.refreshInterval(); got != refresh {
		t.Fatalf("unexpected refresh interval override: got %v want %v", got, refresh)
	}
}

func TestConfigFileAuthMutualExclusion(t *testing.T) {
	path := filepath.Join(t.TempDir(), "secret")
	if err := os.WriteFile(path, []byte("secret"), 0o600); err != nil {
		t.Fatalf("failed to write secret file: %v", err)
	}

	cfg := conf.MustNewConfigFrom(map[string]any{
		"resource.url":        "localhost",
		"auth.file.path":      path,
		"auth.basic.user":     "user",
		"auth.basic.password": "pass",
	})
	conf := defaultConfig()
	conf.Program = "{}"
	conf.Redact = &redact{}
	err := cfg.Unpack(&conf)
	wantErr := errors.New("only one kind of auth can be enabled accessing 'auth'")
	if fmt.Sprint(err) != fmt.Sprint(wantErr) {
		t.Fatalf("unexpected error: got %v want %v", err, wantErr)
	}
}

func TestConfigFileAuthDisabledAllowsOther(t *testing.T) {
	path := filepath.Join(t.TempDir(), "secret")
	if err := os.WriteFile(path, []byte("secret"), 0o600); err != nil {
		t.Fatalf("failed to write secret file: %v", err)
	}

	cfg := conf.MustNewConfigFrom(map[string]any{
		"resource.url":        "localhost",
		"auth.file.enabled":   false,
		"auth.file.path":      path,
		"auth.basic.user":     "user",
		"auth.basic.password": "pass",
	})
	conf := defaultConfig()
	conf.Program = "{}"
	conf.Redact = &redact{}
	if err := cfg.Unpack(&conf); err != nil {
		t.Fatalf("unexpected error unpacking config: %v", err)
	}
}

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
		m := map[string]any{
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
	input    map[string]any
	setup    func()
	teardown func()
}{
	{
		name:    "can't_set_oauth2_and_basic_auth_together",
		wantErr: errors.New("only one kind of auth can be enabled accessing 'auth'"),
		input: map[string]any{
			"auth.basic.user":     "user",
			"auth.basic.password": "pass",
			"auth.oauth2": map[string]any{
				"token_url": "localhost",
				"client": map[string]any{
					"id":     "a_client_id",
					"secret": "a_client_secret",
				},
			},
		},
	},
	{
		name:    "can't_set_oauth2_and_digest_auth_together",
		wantErr: errors.New("only one kind of auth can be enabled accessing 'auth'"),
		input: map[string]any{
			"auth.digest.user":     "user",
			"auth.digest.password": "pass",
			"auth.oauth2": map[string]any{
				"token_url": "localhost",
				"client": map[string]any{
					"id":     "a_client_id",
					"secret": "a_client_secret",
				},
			},
		},
	},
	{
		name:    "can't_set_basic_and_digest_auth_together",
		wantErr: errors.New("only one kind of auth can be enabled accessing 'auth'"),
		input: map[string]any{
			"auth.basic.user":      "user",
			"auth.basic.password":  "pass",
			"auth.digest.user":     "user",
			"auth.digest.password": "pass",
		},
	},
	{
		name: "can_set_oauth2_and_basic_auth_together_if_oauth2_is_disabled",
		input: map[string]any{
			"auth.basic.user":     "user",
			"auth.basic.password": "pass",
			"auth.oauth2": map[string]any{
				"enabled":   false,
				"token_url": "localhost",
				"client": map[string]any{
					"id":     "a_client_id",
					"secret": "a_client_secret",
				},
			},
		},
	},
	{
		name:    "token_url_and_client_credentials_must_be_set",
		wantErr: errors.New("both token_url and client credentials must be provided accessing 'auth.oauth2'"),
		input: map[string]any{
			"auth.oauth2": map[string]any{},
		},
	},
	{
		name: "client_credential_secret_may_be_empty",
		input: map[string]any{
			"auth.oauth2": map[string]any{
				"enabled":   true,
				"token_url": "localhost",
				"client": map[string]any{
					"id":     "a_client_id",
					"secret": "",
				},
			},
		},
	},
	{
		name:    "client_credential_secret_may_not_be_missing",
		wantErr: errors.New("both token_url and client credentials must be provided accessing 'auth.oauth2'"),
		input: map[string]any{
			"auth.oauth2": map[string]any{
				"enabled":   true,
				"token_url": "localhost",
				"client": map[string]any{
					"id": "a_client_id",
				},
			},
		},
	},
	{
		name: "if_user_and_password_is_set_oauth2_must_use_user-password_authentication",
		input: map[string]any{
			"auth.oauth2": map[string]any{
				"user":      "a_client_user",
				"password":  "a_client_password",
				"token_url": "localhost",
				"client": map[string]any{
					"id":     "a_client_id",
					"secret": "a_client_secret",
				},
			},
		},
	},
	{
		name:    "if_user_is_set_password_credentials_must_be_set_for_user-password_authentication",
		wantErr: errors.New("both user and password credentials must be provided accessing 'auth.oauth2'"),
		input: map[string]any{
			"auth.oauth2": map[string]any{
				"user":      "a_client_user",
				"token_url": "localhost",
				"client": map[string]any{
					"id":     "a_client_id",
					"secret": "a_client_secret",
				},
			},
		},
	},
	{
		name:    "if_password_is_set_user_credentials_must_be_set_for_user-password_authentication",
		wantErr: errors.New("both user and password credentials must be provided accessing 'auth.oauth2'"),
		input: map[string]any{
			"auth.oauth2": map[string]any{
				"password":  "a_client_password",
				"token_url": "localhost",
				"client": map[string]any{
					"id":     "a_client_id",
					"secret": "a_client_secret",
				},
			},
		},
	},
	{
		name: "if_password_is_set_credentials_may_be_missing_for_user-password_authentication",
		input: map[string]any{
			"auth.oauth2": map[string]any{
				"user":      "a_client_user",
				"password":  "a_client_password",
				"token_url": "localhost",
			},
		},
	},
	{
		name:    "must_fail_with_an_unknown_provider",
		wantErr: errors.New("unknown provider \"unknown\" accessing 'auth.oauth2'"),
		input: map[string]any{
			"auth.oauth2": map[string]any{
				"provider": "unknown",
			},
		},
	},
	{
		name:    "azure_must_have_either_tenant_id_or_token_url",
		wantErr: errors.New("at least one of token_url or tenant_id must be provided accessing 'auth.oauth2'"),
		input: map[string]any{
			"auth.oauth2": map[string]any{
				"provider": "azure",
			},
		},
	},
	{
		name:    "azure_must_have_only_one_of_token_url_and_tenant_id",
		wantErr: errors.New("only one of token_url and tenant_id can be used accessing 'auth.oauth2'"),
		input: map[string]any{
			"auth.oauth2": map[string]any{
				"provider":        "azure",
				"azure.tenant_id": "a_tenant_id",
				"token_url":       "localhost",
			},
		},
	},
	{
		name:    "azure_must_have_client_credentials_set",
		wantErr: errors.New("client credentials must be provided accessing 'auth.oauth2'"),
		input: map[string]any{
			"auth.oauth2": map[string]any{
				"provider":        "azure",
				"azure.tenant_id": "a_tenant_id",
			},
		},
	},
	{
		name: "azure_config_is_valid",
		input: map[string]any{
			"auth.oauth2": map[string]any{
				"provider": "azure",
				"azure": map[string]any{
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
		input: map[string]any{
			"auth.oauth2": map[string]any{
				"provider": "google",
				"azure": map[string]any{
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
		input: map[string]any{
			"auth.oauth2": map[string]any{
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
		input: map[string]any{
			"auth.oauth2": map[string]any{
				"provider":                "google",
				"google.credentials_file": "./wrong",
			},
		},
	},
	{
		name:    "google_must_fail_if_ADC_is_wrongly_set",
		wantErr: errors.New("no authentication credentials were configured or detected (ADC) accessing 'auth.oauth2'"),
		input: map[string]any{
			"auth.oauth2": map[string]any{
				"provider": "google",
			},
		},
		setup: func() { os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "./wrong") },
	},
	{
		name: "google_must_work_if_ADC_is_set_up",
		input: map[string]any{
			"auth.oauth2": map[string]any{
				"provider": "google",
			},
		},
		setup: func() { os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "./testdata/credentials.json") },
	},
	{
		name: "google_must_work_if_credentials_file_is_correct",
		input: map[string]any{
			"auth.oauth2": map[string]any{
				"provider":                "google",
				"google.credentials_file": "./testdata/credentials.json",
			},
		},
	},
	{
		name: "google_must_work_if_jwt_file_is_correct",
		input: map[string]any{
			"auth.oauth2": map[string]any{
				"provider":        "google",
				"google.jwt_file": "./testdata/credentials.json",
			},
		},
	},
	{
		name: "google must work if jwt_json is correct",
		input: map[string]any{
			"auth.oauth2": map[string]any{
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
		input: map[string]any{
			"auth.oauth2": map[string]any{
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
		input: map[string]any{
			"auth.oauth2": map[string]any{
				"provider":                "google",
				"google.credentials_json": `invalid`,
			},
		},
	},
	{
		name:    "google must fail if jwt_json is not a valid JSON",
		wantErr: errors.New("the field can't be converted to valid JSON accessing 'auth.oauth2.google.jwt_json'"),
		input: map[string]any{
			"auth.oauth2": map[string]any{
				"provider":        "google",
				"google.jwt_json": `invalid`,
			},
		},
	},
	{
		name:    "google_must_fail_if_the_provided_credentials_file_is_not_a_valid_JSON",
		wantErr: errors.New("the file \"./testdata/invalid_credentials.json\" does not contain valid JSON accessing 'auth.oauth2'"),
		input: map[string]any{
			"auth.oauth2": map[string]any{
				"provider":                "google",
				"google.credentials_file": "./testdata/invalid_credentials.json",
			},
		},
	},
	{
		name:    "google_must_fail_if_the_delegated_account_is_set_without_jwt_file",
		wantErr: errors.New("google.delegated_account can only be provided with a jwt_file accessing 'auth.oauth2'"),
		input: map[string]any{
			"auth.oauth2": map[string]any{
				"provider":                 "google",
				"google.credentials_file":  "./testdata/credentials.json",
				"google.delegated_account": "delegated@account.com",
			},
		},
	},
	{
		name: "google_must_work_with_delegated_account_and_a_valid_jwt_file",
		input: map[string]any{
			"auth.oauth2": map[string]any{
				"provider":                 "google",
				"google.jwt_file":          "./testdata/credentials.json",
				"google.delegated_account": "delegated@account.com",
			},
		},
	},
	{
		name:    "unique_okta_jwk_token",
		wantErr: errors.New("okta validation error: one of okta.jwk_json, okta.jwk_file or okta.jwk_pem must be provided accessing 'auth.oauth2'"),
		input: map[string]any{
			"auth.oauth2": map[string]any{
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
		input: map[string]any{
			"auth.oauth2": map[string]any{
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
		input: map[string]any{
			"auth.oauth2": map[string]any{
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
		input: map[string]any{
			"auth.oauth2": map[string]any{
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

	{
		name:    "box_must_have_key_material",
		wantErr: errors.New("box validation error: one of box.config_file, box.config_json or discrete credentials (box.private_key) must be provided accessing 'auth.oauth2'"),
		input: map[string]any{
			"auth.oauth2": map[string]any{
				"provider": "box",
			},
		},
	},
	{
		name:    "box_scopes_rejected",
		wantErr: errors.New("box validation error: scopes must not be set for the Box JWT grant accessing 'auth.oauth2'"),
		input: map[string]any{
			"auth.oauth2": map[string]any{
				"provider":        "box",
				"box.config_json": `{"boxAppSettings":{"clientID":"x","clientSecret":"y","appAuth":{"publicKeyID":"k","privateKey":"` + boxTestKeyOneLiner + `","passphrase":""}},"enterpriseID":"e"}`,
				"scopes":          []string{"foo"},
			},
		},
	},
	{
		name:    "box_config_json_and_private_key_mutually_exclusive",
		wantErr: errors.New("box validation error: box.config_file/box.config_json and discrete fields are mutually exclusive accessing 'auth.oauth2'"),
		input: map[string]any{
			"auth.oauth2": map[string]any{
				"provider":        "box",
				"box.config_json": `{"boxAppSettings":{"clientID":"x","clientSecret":"y","appAuth":{"publicKeyID":"k","privateKey":"` + boxTestKeyOneLiner + `","passphrase":""}},"enterpriseID":"e"}`,
				"box.private_key": testBoxPEMForConfig,
			},
		},
	},
	{
		name:    "box_discrete_requires_client_id_and_secret",
		wantErr: errors.New("box validation error: client.id and client.secret must be provided when using discrete credentials accessing 'auth.oauth2'"),
		input: map[string]any{
			"auth.oauth2": map[string]any{
				"provider":        "box",
				"box.private_key": testBoxPEMForConfig,
			},
		},
	},
	{
		name:    "box_invalid_subject_type",
		wantErr: errors.New(`box validation error: box.subject_type must be enterprise or user, got "invalid" accessing 'auth.oauth2'`),
		input: map[string]any{
			"auth.oauth2": map[string]any{
				"provider":         "box",
				"box.config_json":  `{"boxAppSettings":{"clientID":"x","clientSecret":"y","appAuth":{"publicKeyID":"k","privateKey":"` + boxTestKeyOneLiner + `","passphrase":""}},"enterpriseID":"e"}`,
				"box.subject_type": "invalid",
			},
		},
	},
	{
		name:    "box_user_subject_requires_subject_id",
		wantErr: errors.New("box validation error: box.subject_id must be provided when box.subject_type is user accessing 'auth.oauth2'"),
		input: map[string]any{
			"auth.oauth2": map[string]any{
				"provider":         "box",
				"box.config_json":  `{"boxAppSettings":{"clientID":"x","clientSecret":"y","appAuth":{"publicKeyID":"k","privateKey":"` + boxTestKeyOneLiner + `","passphrase":""}},"enterpriseID":"e"}`,
				"box.subject_type": "user",
			},
		},
	},
	{
		name: "box_successful_config_json_validation",
		input: map[string]any{
			"auth.oauth2": map[string]any{
				"provider":        "box",
				"box.config_json": `{"boxAppSettings":{"clientID":"x","clientSecret":"y","appAuth":{"publicKeyID":"k","privateKey":"` + boxTestKeyOneLiner + `","passphrase":""}},"enterpriseID":"e"}`,
			},
		},
	},
	{
		name: "box_successful_discrete_validation",
		input: map[string]any{
			"auth.oauth2": map[string]any{
				"provider":          "box",
				"client.id":         "x",
				"client.secret":     "y",
				"box.private_key":   testBoxPEMForConfig,
				"box.enterprise_id": "eid123",
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
	input   map[string]any
	want    httpcommon.WithKeepaliveSettings
	wantErr error
}{
	{
		name:  "keep_alive_none", // Default to the old behaviour of true.
		input: map[string]any{},
		want:  httpcommon.WithKeepaliveSettings{Disable: true},
	},
	{
		name: "keep_alive_true",
		input: map[string]any{
			"resource.keep_alive.disable": true,
		},
		want: httpcommon.WithKeepaliveSettings{Disable: true},
	},
	{
		name: "keep_alive_false",
		input: map[string]any{
			"resource.keep_alive.disable": false,
		},
		want: httpcommon.WithKeepaliveSettings{Disable: false},
	},
	{
		name: "keep_alive_invalid_max",
		input: map[string]any{
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
