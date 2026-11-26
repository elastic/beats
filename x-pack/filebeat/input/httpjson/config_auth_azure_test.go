// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpjson

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/pkg/logp/logptest"
	"github.com/elastic/beats/v7/x-pack/libbeat/common/identityfederation"
)

func TestOAuth2AzureUseCloudConnectorsWithoutWIIEnvironment(t *testing.T) {
	t.Setenv(identityfederation.WIIIssuerURLEnvVar, "")
	cfg := oAuth2Config{
		Provider:                oAuth2ProviderAzure,
		ClientID:                "a_client_id",
		AzureTenantID:           "a_tenant_id",
		AzureUseCloudConnectors: true,
	}

	_, err := cfg.client(t.Context(), http.DefaultClient, logptest.NewTestingLogger(t, ""))
	assert.ErrorContains(t, err, identityfederation.WIIIssuerURLEnvVar, "the input must stop with the WII error, not fall back to a client secret")
}
