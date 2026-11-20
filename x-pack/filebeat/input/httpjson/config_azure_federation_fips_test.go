// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build requirefips

package httpjson

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	conf "github.com/elastic/beats/v7/pkg/config"
	"github.com/elastic/beats/v7/pkg/logp/logptest"
)

func TestOAuth2AzureCloudConnectorsFIPS(t *testing.T) {
	cfg := conf.MustNewConfigFrom(map[string]any{
		"request.url": "https://localhost",
		"auth.oauth2": map[string]any{
			"provider":                   "azure",
			"azure.use_cloud_connectors": true,
			"azure.tenant_id":            "a_tenant_id",
			"azure.resource":             "https://management.azure.com/",
			"client.id":                  "a_client_id",
		},
	})
	c := defaultConfig()
	err := cfg.Unpack(&c)
	require.EqualError(t, err, "azure.use_cloud_connectors is not supported in FIPS builds accessing 'auth.oauth2'")

	o := &oAuth2Config{}
	_, err = o.azureCloudConnectorsClient(t.Context(), http.DefaultClient, logptest.NewTestingLogger(t, ""))
	require.EqualError(t, err, "azure.use_cloud_connectors is not supported in FIPS builds")
}
