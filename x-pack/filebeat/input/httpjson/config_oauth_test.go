// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpjson

import "testing"

func TestProviderCanonical(t *testing.T) {
	const (
		a OAuth2Provider = "gOoGle"
		b OAuth2Provider = "google"
	)

	if a.canonical() != b.canonical() {
		t.Fatal("Canonical provider should be equal")
	}
}

func TestGetProviderIsCanonical(t *testing.T) {
	const expected OAuth2Provider = "google"

	oauth2 := OAuth2{Provider: "GOogle"}
	if oauth2.GetProvider() != expected {
		t.Fatal("GetProvider should return canonical provider")
	}
}

func TestIsEnabled(t *testing.T) {
	oauth2 := OAuth2{}
	if !oauth2.IsEnabled() {
		t.Fatal("OAuth2 should be enabled by default")
	}

	var enabled = false
	oauth2.Enabled = &enabled

	if oauth2.IsEnabled() {
		t.Fatal("OAuth2 should be disabled")
	}

	enabled = true
	if !oauth2.IsEnabled() {
		t.Fatal("OAuth2 should be enabled")
	}
}

func TestGetTokenURL(t *testing.T) {
	const expected = "http://localhost"
	oauth2 := OAuth2{TokenURL: "http://localhost"}
	if oauth2.GetTokenURL() != expected {
		t.Fatal("GetTokenURL should return the provided TokenURL")
	}
}

func TestGetTokenURLWithAzure(t *testing.T) {
	const expectedWithoutTenantID = "http://localhost"
	oauth2 := OAuth2{TokenURL: "http://localhost", Provider: "azure"}
	if got := oauth2.GetTokenURL(); got != expectedWithoutTenantID {
		t.Fatalf("GetTokenURL should return the provided TokenURL but got %q", got)
	}

	oauth2.TokenURL = ""
	oauth2.AzureTenantID = "a_tenant_id"
	const expectedWithTenantID = "https://login.microsoftonline.com/a_tenant_id/oauth2/v2.0/token"
	if got := oauth2.GetTokenURL(); got != expectedWithTenantID {
		t.Fatalf("GetTokenURL should return the generated TokenURL but got %q", got)
	}
}
