// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package v2

import (
	"testing"

	"github.com/stretchr/testify/assert"
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

	var enabled = false
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
	var expected = map[string][]string{"foo": {"bar"}}
	oauth2 := oAuth2Config{EndpointParams: map[string][]string{"foo": {"bar"}}}
	assert.Equal(t, expected, oauth2.getEndpointParams())
}

func TestGetEndpointParamsWithAzure(t *testing.T) {
	var expectedWithoutResource = map[string][]string{"foo": {"bar"}}
	oauth2 := oAuth2Config{Provider: "azure", EndpointParams: map[string][]string{"foo": {"bar"}}}

	assert.Equal(t, expectedWithoutResource, oauth2.getEndpointParams())

	oauth2.AzureResource = "baz"
	var expectedWithResource = map[string][]string{"foo": {"bar"}, "resource": {"baz"}}

	assert.Equal(t, expectedWithResource, oauth2.getEndpointParams())
}
