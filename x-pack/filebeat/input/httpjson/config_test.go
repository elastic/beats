// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpjson

import (
	"os"
	"testing"

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

func TestConfigValidationCase8(t *testing.T) {
	const expectedErr = "invalid configuration: oauth2 and api_key or authentication_scheme cannot be set simultaneously accessing config"
	m := map[string]interface{}{
		"api_key": "an_api_key",
		"oauth2": map[string]interface{}{
			"token_url": "localhost",
			"client": map[string]interface{}{
				"id":     "a_client_id",
				"secret": "a_client_secret",
			},
		},
		"url": "localhost",
	}
	cfg := common.MustNewConfigFrom(m)
	conf := defaultConfig()
	if err := cfg.Unpack(&conf); err.Error() != expectedErr {
		t.Fatalf("Configuration validation failed. expecting %q error but got %q", expectedErr, err)
	}
}

func TestConfigValidationCase9(t *testing.T) {
	const expectedErr = "invalid configuration: oauth2 and api_key or authentication_scheme cannot be set simultaneously accessing config"
	m := map[string]interface{}{
		"authentication_scheme": "an_api_key",
		"oauth2": map[string]interface{}{
			"token_url": "localhost",
			"client": map[string]interface{}{
				"id":     "a_client_id",
				"secret": "a_client_secret",
			},
		},
		"url": "localhost",
	}
	cfg := common.MustNewConfigFrom(m)
	conf := defaultConfig()
	if err := cfg.Unpack(&conf); err.Error() != expectedErr {
		t.Fatalf("Configuration validation failed. expecting %q error but got %q", expectedErr, err)
	}
}

func TestConfigValidationCase10(t *testing.T) {
	const expectedErr = "invalid configuration: both token_url and client credentials must be provided accessing 'oauth2'"
	m := map[string]interface{}{
		"oauth2": map[string]interface{}{},
		"url":    "localhost",
	}
	cfg := common.MustNewConfigFrom(m)
	conf := defaultConfig()
	if err := cfg.Unpack(&conf); err.Error() != expectedErr {
		t.Fatalf("Configuration validation failed. expecting %q error but got %q", expectedErr, err)
	}
}

func TestConfigValidationCase11(t *testing.T) {
	const expectedErr = "invalid configuration: unknown provider \"unknown\" accessing 'oauth2'"
	m := map[string]interface{}{
		"oauth2": map[string]interface{}{
			"provider": "unknown",
		},
		"url": "localhost",
	}
	cfg := common.MustNewConfigFrom(m)
	conf := defaultConfig()
	if err := cfg.Unpack(&conf); err == nil || err.Error() != expectedErr {
		t.Fatalf("Configuration validation failed. expecting %q error but got %q", expectedErr, err)
	}
}

func TestConfigValidationCase12(t *testing.T) {
	const expectedErr = "invalid configuration: at least one of token_url or tenant_id must be provided accessing 'oauth2'"
	m := map[string]interface{}{
		"oauth2": map[string]interface{}{
			"provider": "azure",
		},
		"url": "localhost",
	}
	cfg := common.MustNewConfigFrom(m)
	conf := defaultConfig()
	if err := cfg.Unpack(&conf); err == nil || err.Error() != expectedErr {
		t.Fatalf("Configuration validation failed. expecting %q error but got %q", expectedErr, err)
	}
}

func TestConfigValidationCase13(t *testing.T) {
	const expectedErr = "invalid configuration: only one of token_url and tenant_id can be used accessing 'oauth2'"
	m := map[string]interface{}{
		"oauth2": map[string]interface{}{
			"provider":        "azure",
			"azure.tenant_id": "a_tenant_id",
			"token_url":       "localhost",
		},
		"url": "localhost",
	}
	cfg := common.MustNewConfigFrom(m)
	conf := defaultConfig()
	if err := cfg.Unpack(&conf); err == nil || err.Error() != expectedErr {
		t.Fatalf("Configuration validation failed. expecting %q error but got %q", expectedErr, err)
	}
}

func TestConfigValidationCase14(t *testing.T) {
	const expectedErr = "invalid configuration: client credentials must be provided accessing 'oauth2'"
	m := map[string]interface{}{
		"oauth2": map[string]interface{}{
			"provider":        "azure",
			"azure.tenant_id": "a_tenant_id",
		},
		"url": "localhost",
	}
	cfg := common.MustNewConfigFrom(m)
	conf := defaultConfig()
	if err := cfg.Unpack(&conf); err == nil || err.Error() != expectedErr {
		t.Fatalf("Configuration validation failed. expecting %q error but got %q", expectedErr, err)
	}
}

func TestConfigValidationCase15(t *testing.T) {
	m := map[string]interface{}{
		"oauth2": map[string]interface{}{
			"provider": "azure",
			"azure": map[string]interface{}{
				"tenant_id": "a_tenant_id",
			},
			"client.id":     "a_client_id",
			"client.secret": "a_client_secret",
		},
		"url": "localhost",
	}
	cfg := common.MustNewConfigFrom(m)
	conf := defaultConfig()
	if err := cfg.Unpack(&conf); err != nil {
		t.Fatalf("Configuration validation failed. no error expected but got %q", err)
	}
}

func TestConfigValidationCase16(t *testing.T) {
	const expectedErr = "invalid configuration: none of token_url and client credentials can be used, use google.credentials_file, google.jwt_file, google.credentials_json or ADC instead accessing 'oauth2'"
	m := map[string]interface{}{
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
	}
	cfg := common.MustNewConfigFrom(m)
	conf := defaultConfig()
	if err := cfg.Unpack(&conf); err == nil || err.Error() != expectedErr {
		t.Fatalf("Configuration validation failed. expecting %q error but got %q", expectedErr, err)
	}
}

func TestConfigValidationCase17(t *testing.T) {
	const expectedErr = "invalid configuration: no authentication credentials were configured or detected (ADC) accessing 'oauth2'"
	m := map[string]interface{}{
		"oauth2": map[string]interface{}{
			"provider": "google",
		},
		"url": "localhost",
	}
	cfg := common.MustNewConfigFrom(m)
	conf := defaultConfig()
	if err := cfg.Unpack(&conf); err == nil || err.Error() != expectedErr {
		t.Fatalf("Configuration validation failed. expecting %q error but got %q", expectedErr, err)
	}
}

func TestConfigValidationCase18(t *testing.T) {
	const expectedErr = "invalid configuration: the file \"./wrong\" cannot be found accessing 'oauth2'"
	m := map[string]interface{}{
		"oauth2": map[string]interface{}{
			"provider":                "google",
			"google.credentials_file": "./wrong",
		},
		"url": "localhost",
	}
	cfg := common.MustNewConfigFrom(m)
	conf := defaultConfig()
	if err := cfg.Unpack(&conf); err == nil || err.Error() != expectedErr {
		t.Fatalf("Configuration validation failed. expecting %q error but got %q", expectedErr, err)
	}
}

func TestConfigValidationCase19(t *testing.T) {
	os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "./wrong")
	const expectedErr = "invalid configuration: no authentication credentials were configured or detected (ADC) accessing 'oauth2'"
	m := map[string]interface{}{
		"oauth2": map[string]interface{}{
			"provider": "google",
		},
		"url": "localhost",
	}
	cfg := common.MustNewConfigFrom(m)
	conf := defaultConfig()
	if err := cfg.Unpack(&conf); err == nil || err.Error() != expectedErr {
		t.Fatalf("Configuration validation failed. expecting %q error but got %q", expectedErr, err)
	}
}

func TestConfigValidationCase20(t *testing.T) {
	os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "./testdata/credentials.json")
	m := map[string]interface{}{
		"oauth2": map[string]interface{}{
			"provider": "google",
		},
		"url": "localhost",
	}
	cfg := common.MustNewConfigFrom(m)
	conf := defaultConfig()
	if err := cfg.Unpack(&conf); err != nil {
		t.Fatalf("Configuration validation failed. no error expected but got %q", err)
	}
}

func TestConfigValidationCase21(t *testing.T) {
	m := map[string]interface{}{
		"oauth2": map[string]interface{}{
			"provider":                "google",
			"google.credentials_file": "./testdata/credentials.json",
		},
		"url": "localhost",
	}
	cfg := common.MustNewConfigFrom(m)
	conf := defaultConfig()
	if err := cfg.Unpack(&conf); err != nil {
		t.Fatalf("Configuration validation failed. no error expected but got %q", err)
	}
}

func TestConfigValidationCase22(t *testing.T) {
	m := map[string]interface{}{
		"oauth2": map[string]interface{}{
			"provider":        "google",
			"google.jwt_file": "./testdata/credentials.json",
		},
		"url": "localhost",
	}
	cfg := common.MustNewConfigFrom(m)
	conf := defaultConfig()
	if err := cfg.Unpack(&conf); err != nil {
		t.Fatalf("Configuration validation failed. no error expected but got %q", err)
	}
}

func TestConfigValidationCase23(t *testing.T) {
	m := map[string]interface{}{
		"oauth2": map[string]interface{}{
			"provider": "google",
			"google.credentials_json": []byte(`{
				"type":           "service_account",
				"project_id":     "foo",
				"private_key_id": "x",
				"client_email":   "foo@bar.com",
				"client_id":      "0",
			}`),
		},
		"url": "localhost",
	}
	cfg := common.MustNewConfigFrom(m)
	conf := defaultConfig()
	if err := cfg.Unpack(&conf); err != nil {
		t.Fatalf("Configuration validation failed. no error expected but got %q", err)
	}
}
