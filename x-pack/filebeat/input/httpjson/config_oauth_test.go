// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpjson

import (
	"reflect"
	"testing"
)

func TestProviderCanonical(t *testing.T) {
	const (
		a oauth2Provider = "gOoGle"
		b oauth2Provider = "google"
	)

	if a.canonical() != b.canonical() {
		t.Fatal("Canonical provider should be equal")
	}
}

func TestGetProviderIsCanonical(t *testing.T) {
	const expected oauth2Provider = "google"

	oauth2 := oauth2Config{Provider: "GOogle"}
	if oauth2.getProvider() != expected {
		t.Fatal("GetProvider should return canonical provider")
	}
}

func TestIsEnabled(t *testing.T) {
	oauth2 := oauth2Config{}
	if !oauth2.isEnabled() {
		t.Fatal("OAuth2 should be enabled by default")
	}

	var enabled = false
	oauth2.Enabled = &enabled

	if oauth2.isEnabled() {
		t.Fatal("OAuth2 should be disabled")
	}

	enabled = true
	if !oauth2.isEnabled() {
		t.Fatal("OAuth2 should be enabled")
	}
}

func TestGetTokenURL(t *testing.T) {
	const expected = "http://localhost"
	oauth2 := oauth2Config{TokenURL: "http://localhost"}
	if got := oauth2.getTokenURL(); got != expected {
		t.Fatalf("GetTokenURL should return the provided TokenURL but got %q", got)
	}
}

func TestGetTokenURLWithAzure(t *testing.T) {
	const expectedWithoutTenantID = "http://localhost"
	oauth2 := oauth2Config{TokenURL: "http://localhost", Provider: "azure"}
	if got := oauth2.getTokenURL(); got != expectedWithoutTenantID {
		t.Fatalf("GetTokenURL should return the provided TokenURL but got %q", got)
	}

	oauth2.TokenURL = ""
	oauth2.AzureTenantID = "a_tenant_id"
	const expectedWithTenantID = "https://login.microsoftonline.com/a_tenant_id/oauth2/v2.0/token"
	if got := oauth2.getTokenURL(); got != expectedWithTenantID {
		t.Fatalf("GetTokenURL should return the generated TokenURL but got %q", got)
	}
}

func TestGetEndpointParams(t *testing.T) {
	var expected = map[string][]string{"foo": {"bar"}}
	oauth2 := oauth2Config{EndpointParams: map[string][]string{"foo": {"bar"}}}
	if got := oauth2.getEndpointParams(); !reflect.DeepEqual(got, expected) {
		t.Fatalf("GetEndpointParams should return the provided EndpointParams but got %q", got)
	}
}

func TestGetEndpointParamsWithAzure(t *testing.T) {
	var expectedWithoutResource = map[string][]string{"foo": {"bar"}}
	oauth2 := oauth2Config{Provider: "azure", EndpointParams: map[string][]string{"foo": {"bar"}}}
	if got := oauth2.getEndpointParams(); !reflect.DeepEqual(got, expectedWithoutResource) {
		t.Fatalf("GetEndpointParams should return the provided EndpointParams but got %q", got)
	}

	oauth2.AzureResource = "baz"
	var expectedWithResource = map[string][]string{"foo": {"bar"}, "resource": {"baz"}}
	if got := oauth2.getEndpointParams(); !reflect.DeepEqual(got, expectedWithResource) {
		t.Fatalf("GetEndpointParams should return the provided EndpointParams but got %q", got)
	}
}
