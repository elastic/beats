// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpjson

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2/google"

	"github.com/elastic/beats/v7/libbeat/common"
)

func TestConfigValidationCase1(t *testing.T) {
	m := map[string]interface{}{
		"http_method":       "GET",
		"http_request_body": map[string]interface{}{"test": "abc"},
		"no_http_body":      true,
		"url":               "localhost",
	}
	cfg := common.MustNewConfigFrom(m)
	conf := defaultConfig()
	if err := cfg.Unpack(&conf); err == nil {
		t.Fatal("Configuration validation failed. no_http_body and http_request_body cannot coexist.")
	}
}

func TestConfigValidationCase2(t *testing.T) {
	m := map[string]interface{}{
		"http_method":  "GET",
		"no_http_body": true,
		"pagination":   map[string]interface{}{"extra_body_content": map[string]interface{}{"test": "abc"}},
		"url":          "localhost",
	}
	cfg := common.MustNewConfigFrom(m)
	conf := defaultConfig()
	if err := cfg.Unpack(&conf); err == nil {
		t.Fatal("Configuration validation failed. no_http_body and pagination.extra_body_content cannot coexist.")
	}
}

func TestConfigValidationCase3(t *testing.T) {
	m := map[string]interface{}{
		"http_method":  "GET",
		"no_http_body": true,
		"pagination":   map[string]interface{}{"req_field": "abc"},
		"url":          "localhost",
	}
	cfg := common.MustNewConfigFrom(m)
	conf := defaultConfig()
	if err := cfg.Unpack(&conf); err == nil {
		t.Fatal("Configuration validation failed. no_http_body and pagination.req_field cannot coexist.")
	}
}

func TestConfigValidationCase4(t *testing.T) {
	m := map[string]interface{}{
		"http_method": "GET",
		"pagination":  map[string]interface{}{"header": map[string]interface{}{"field_name": "Link", "regex_pattern": "<([^>]+)>; *rel=\"next\"(?:,|$)"}, "req_field": "abc"},
		"url":         "localhost",
	}
	cfg := common.MustNewConfigFrom(m)
	conf := defaultConfig()
	if err := cfg.Unpack(&conf); err == nil {
		t.Fatal("Configuration validation failed. pagination.header and pagination.req_field cannot coexist.")
	}
}

func TestConfigValidationCase5(t *testing.T) {
	m := map[string]interface{}{
		"http_method": "GET",
		"pagination":  map[string]interface{}{"header": map[string]interface{}{"field_name": "Link", "regex_pattern": "<([^>]+)>; *rel=\"next\"(?:,|$)"}, "id_field": "abc"},
		"url":         "localhost",
	}
	cfg := common.MustNewConfigFrom(m)
	conf := defaultConfig()
	if err := cfg.Unpack(&conf); err == nil {
		t.Fatal("Configuration validation failed. pagination.header and pagination.id_field cannot coexist.")
	}
}

func TestConfigValidationCase6(t *testing.T) {
	m := map[string]interface{}{
		"http_method": "GET",
		"pagination":  map[string]interface{}{"header": map[string]interface{}{"field_name": "Link", "regex_pattern": "<([^>]+)>; *rel=\"next\"(?:,|$)"}, "extra_body_content": map[string]interface{}{"test": "abc"}},
		"url":         "localhost",
	}
	cfg := common.MustNewConfigFrom(m)
	conf := defaultConfig()
	if err := cfg.Unpack(&conf); err == nil {
		t.Fatal("Configuration validation failed. pagination.header and extra_body_content cannot coexist.")
	}
}

func TestConfigValidationCase7(t *testing.T) {
	m := map[string]interface{}{
		"http_method":  "DELETE",
		"no_http_body": true,
		"url":          "localhost",
	}
	cfg := common.MustNewConfigFrom(m)
	conf := defaultConfig()
	if err := cfg.Unpack(&conf); err == nil {
		t.Fatal("Configuration validation failed. http_method DELETE is not allowed.")
	}
}

func TestConfigMustFailWithInvalidURL(t *testing.T) {
	m := map[string]interface{}{
		"url": "::invalid::",
	}
	cfg := common.MustNewConfigFrom(m)
	conf := defaultConfig()
	err := cfg.Unpack(&conf)
	assert.EqualError(t, err, `parse "::invalid::": missing protocol scheme accessing 'url'`)
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
			name:        "can't set oauth2 and api_key together",
			expectedErr: "invalid configuration: oauth2 and api_key or authentication_scheme cannot be set simultaneously accessing config",
			input: map[string]interface{}{
				"api_key": "an_api_key",
				"oauth2": map[string]interface{}{
					"token_url": "localhost",
					"client": map[string]interface{}{
						"id":     "a_client_id",
						"secret": "a_client_secret",
					},
				},
				"url": "localhost",
			},
		},
		{
			name: "can set oauth2 and api_key together if oauth2 is disabled",
			input: map[string]interface{}{
				"api_key": "an_api_key",
				"oauth2": map[string]interface{}{
					"enabled":   false,
					"token_url": "localhost",
					"client": map[string]interface{}{
						"id":     "a_client_id",
						"secret": "a_client_secret",
					},
				},
				"url": "localhost",
			},
		},
		{
			name:        "can't set oauth2 and authentication_scheme",
			expectedErr: "invalid configuration: oauth2 and api_key or authentication_scheme cannot be set simultaneously accessing config",
			input: map[string]interface{}{
				"authentication_scheme": "a_scheme",
				"oauth2": map[string]interface{}{
					"token_url": "localhost",
					"client": map[string]interface{}{
						"id":     "a_client_id",
						"secret": "a_client_secret",
					},
				},
				"url": "localhost",
			},
		},
		{
			name:        "token_url and client credentials must be set",
			expectedErr: "invalid configuration: both token_url and client credentials must be provided accessing 'oauth2'",
			input: map[string]interface{}{
				"oauth2": map[string]interface{}{},
				"url":    "localhost",
			},
		},
		{
			name:        "must fail with an unknown provider",
			expectedErr: "invalid configuration: unknown provider \"unknown\" accessing 'oauth2'",
			input: map[string]interface{}{
				"oauth2": map[string]interface{}{
					"provider": "unknown",
				},
				"url": "localhost",
			},
		},
		{
			name:        "azure must have either tenant_id or token_url",
			expectedErr: "invalid configuration: at least one of token_url or tenant_id must be provided accessing 'oauth2'",
			input: map[string]interface{}{
				"oauth2": map[string]interface{}{
					"provider": "azure",
				},
				"url": "localhost",
			},
		},
		{
			name:        "azure must have only one of token_url and tenant_id",
			expectedErr: "invalid configuration: only one of token_url and tenant_id can be used accessing 'oauth2'",
			input: map[string]interface{}{
				"oauth2": map[string]interface{}{
					"provider":        "azure",
					"azure.tenant_id": "a_tenant_id",
					"token_url":       "localhost",
				},
				"url": "localhost",
			},
		},
		{
			name:        "azure must have client credentials set",
			expectedErr: "invalid configuration: client credentials must be provided accessing 'oauth2'",
			input: map[string]interface{}{
				"oauth2": map[string]interface{}{
					"provider":        "azure",
					"azure.tenant_id": "a_tenant_id",
				},
				"url": "localhost",
			},
		},
		{
			name: "azure config is valid",
			input: map[string]interface{}{
				"oauth2": map[string]interface{}{
					"provider": "azure",
					"azure": map[string]interface{}{
						"tenant_id": "a_tenant_id",
					},
					"client.id":     "a_client_id",
					"client.secret": "a_client_secret",
				},
				"url": "localhost",
			},
		},
		{
			name:        "google can't have token_url or client credentials set",
			expectedErr: "invalid configuration: none of token_url and client credentials can be used, use google.credentials_file, google.jwt_file, google.credentials_json or ADC instead accessing 'oauth2'",
			input: map[string]interface{}{
				"oauth2": map[string]interface{}{
					"provider": "google",
					"azure": map[string]interface{}{
						"tenant_id": "a_tenant_id",
					},
					"client.id":     "a_client_id",
					"client.secret": "a_client_secret",
					"token_url":     "localhost",
				},
				"url": "localhost",
			},
		},
		{
			name:        "google must fail if no ADC available",
			expectedErr: "invalid configuration: no authentication credentials were configured or detected (ADC) accessing 'oauth2'",
			input: map[string]interface{}{
				"oauth2": map[string]interface{}{
					"provider": "google",
				},
				"url": "localhost",
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
			name:        "google must fail if credentials file not found",
			expectedErr: "invalid configuration: the file \"./wrong\" cannot be found accessing 'oauth2'",
			input: map[string]interface{}{
				"oauth2": map[string]interface{}{
					"provider":                "google",
					"google.credentials_file": "./wrong",
				},
				"url": "localhost",
			},
		},
		{
			name:        "google must fail if ADC is wrongly set",
			expectedErr: "invalid configuration: no authentication credentials were configured or detected (ADC) accessing 'oauth2'",
			input: map[string]interface{}{
				"oauth2": map[string]interface{}{
					"provider": "google",
				},
				"url": "localhost",
			},
			setup: func() { os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "./wrong") },
		},
		{
			name: "google must work if ADC is set up",
			input: map[string]interface{}{
				"oauth2": map[string]interface{}{
					"provider": "google",
				},
				"url": "localhost",
			},
			setup: func() { os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "./testdata/credentials.json") },
		},
		{
			name: "google must work if credentials_file is correct",
			input: map[string]interface{}{
				"oauth2": map[string]interface{}{
					"provider":                "google",
					"google.credentials_file": "./testdata/credentials.json",
				},
				"url": "localhost",
			},
		},
		{
			name: "google must work if jwt_file is correct",
			input: map[string]interface{}{
				"oauth2": map[string]interface{}{
					"provider":        "google",
					"google.jwt_file": "./testdata/credentials.json",
				},
				"url": "localhost",
			},
		},
		{
			name: "google must work if credentials_json is correct",
			input: map[string]interface{}{
				"oauth2": map[string]interface{}{
					"provider": "google",
					"google.credentials_json": []byte(`{
						"type":           "service_account",
						"project_id":     "foo",
						"private_key_id": "x",
						"client_email":   "foo@bar.com",
						"client_id":      "0"
					}`),
				},
				"url": "localhost",
			},
		},
		{
			name:        "google must fail if credentials_json is not a valid JSON",
			expectedErr: "invalid configuration: google.credentials_json must be valid JSON accessing 'oauth2'",
			input: map[string]interface{}{
				"oauth2": map[string]interface{}{
					"provider":                "google",
					"google.credentials_json": []byte(`invalid`),
				},
				"url": "localhost",
			},
		},
		{
			name:        "google must fail if the provided credentials file is not a valid JSON",
			expectedErr: "invalid configuration: the file \"./testdata/invalid_credentials.json\" does not contain valid JSON accessing 'oauth2'",
			input: map[string]interface{}{
				"oauth2": map[string]interface{}{
					"provider":                "google",
					"google.credentials_file": "./testdata/invalid_credentials.json",
				},
				"url": "localhost",
			},
		},
		{
			name:        "date_cursor.date_format will fail if invalid",
			expectedErr: "invalid configuration: date_format is not a valid date layout accessing 'date_cursor'",
			input: map[string]interface{}{
				"date_cursor": map[string]interface{}{"field": "foo", "url_field": "foo", "date_format": "1234"},
				"url":         "localhost",
			},
		},
		{
			name: "date_cursor must work with a valid date_format",
			input: map[string]interface{}{
				"date_cursor": map[string]interface{}{"field": "foo", "url_field": "foo", "date_format": time.RFC3339},
				"url":         "localhost",
			},
		},
		{
			name:        "google must fail if the delegated_account is set without jwt_file",
			expectedErr: "invalid configuration: google.delegated_account can only be provided with a jwt_file accessing 'oauth2'",
			input: map[string]interface{}{
				"oauth2": map[string]interface{}{
					"provider":                 "google",
					"google.credentials_file":  "./testdata/credentials.json",
					"google.delegated_account": "delegated@account.com",
				},
				"url": "localhost",
			},
		},
		{
			name: "google must work with delegated_account and a valid jwt_file",
			input: map[string]interface{}{
				"oauth2": map[string]interface{}{
					"provider":                 "google",
					"google.jwt_file":          "./testdata/credentials.json",
					"google.delegated_account": "delegated@account.com",
				},
				"url": "localhost",
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

			cfg := common.MustNewConfigFrom(c.input)
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
