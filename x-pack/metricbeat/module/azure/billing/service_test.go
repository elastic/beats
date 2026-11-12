// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !requirefips

package billing

import (
	"maps"
	"os"
	"path/filepath"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/pkg/logp/logptest"
	"github.com/elastic/beats/v7/x-pack/libbeat/common/identityfederation"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/azure"
)

func TestBillingCloudConfiguration(t *testing.T) {
	tests := []struct {
		name         string
		config       azure.Config
		wantEndpoint string
		wantAudience string
		wantADHost   string
	}{
		{
			name: "public cloud defaults",
			config: azure.Config{
				ActiveDirectoryEndpoint: "https://login.microsoftonline.com/",
			},
			wantEndpoint: "https://management.azure.com",
			wantAudience: "https://management.core.windows.net/",
			wantADHost:   "https://login.microsoftonline.com/",
		},
		{
			name: "public cloud explicit configuration",
			config: azure.Config{
				ResourceManagerEndpoint: azure.DefaultBaseURI,
				ResourceManagerAudience: "https://management.core.windows.net/",
				ActiveDirectoryEndpoint: "https://login.microsoftonline.com/",
			},
			wantEndpoint: "https://management.azure.com",
			wantAudience: "https://management.core.windows.net/",
			wantADHost:   "https://login.microsoftonline.com/",
		},
		{
			name: "government explicit configuration",
			config: azure.Config{
				ResourceManagerEndpoint: azure.GovCloudBaseURI,
				ResourceManagerAudience: "https://management.core.usgovcloudapi.net/",
				ActiveDirectoryEndpoint: "https://login.microsoftonline.us/",
			},
			wantEndpoint: "https://management.usgovcloudapi.net/",
			wantAudience: "https://management.core.usgovcloudapi.net/",
			wantADHost:   "https://login.microsoftonline.us/",
		},
		{
			name: "custom cloud explicit configuration",
			config: azure.Config{
				ResourceManagerEndpoint: "https://management.local.azurestack.external/",
				ResourceManagerAudience: "https://management.adfs.azurestack.local/tenant",
				ActiveDirectoryEndpoint: "https://adfs.local.azurestack.external/",
			},
			wantEndpoint: "https://management.local.azurestack.external/",
			wantAudience: "https://management.adfs.azurestack.local/tenant",
			wantADHost:   "https://adfs.local.azurestack.external/",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			configuration := azure.BuildCloudConfig(test.config)
			resourceManager := configuration.Services[cloud.ResourceManager]

			assert.Equal(t, test.wantEndpoint, resourceManager.Endpoint, "billing resource manager endpoint must preserve existing behavior")
			assert.Equal(t, test.wantAudience, resourceManager.Audience, "billing resource manager audience must preserve existing behavior")
			assert.Equal(t, test.wantADHost, configuration.ActiveDirectoryAuthorityHost, "billing AD authority must preserve existing behavior")
		})
	}
}

func TestNewServiceWithCloudConnectors(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")
	config := azure.Config{
		TenantId:                "00000000-0000-0000-0000-000000000000",
		ClientId:                "00000000-0000-0000-0000-000000000001",
		ActiveDirectoryEndpoint: "https://login.microsoftonline.com/",
		UseCloudConnectors:      true,
	}

	t.Run("creates the service", func(t *testing.T) {
		p := filepath.Join(t.TempDir(), "id-token")
		require.NoError(t, os.WriteFile(p, []byte("eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJ0ZXN0In0.signature"), 0o600), "expected the token file to be written")
		t.Setenv(identityfederation.WIIIssuerURLEnvVar, "")
		t.Setenv(identityfederation.CloudConnectorsIDTokenFileEnvVar, p)

		service, err := NewService(config, logger)
		require.NoError(t, err, "billing service must be created with identity federation")
		require.NotNil(t, service.usageDetailsClient, "billing usage details client must be constructed")
		require.NotNil(t, service.forecastClient, "billing forecast client must be constructed")
	})

	t.Run("fails without an identity token source", func(t *testing.T) {
		t.Setenv(identityfederation.WIIIssuerURLEnvVar, "")
		t.Setenv(identityfederation.CloudConnectorsIDTokenFileEnvVar, "")

		_, err := NewService(config, logger)
		require.ErrorContains(t, err, "no identity token source configured", "billing service must report the identity federation error")
	})
}

func TestNewServiceDoesNotMutateSDKCloudConfigurations(t *testing.T) {
	publicBefore := maps.Clone(cloud.AzurePublic.Services)
	governmentBefore := maps.Clone(cloud.AzureGovernment.Services)
	chinaBefore := maps.Clone(cloud.AzureChina.Services)

	configs := []azure.Config{
		{
			TenantId:                "00000000-0000-0000-0000-000000000000",
			ClientId:                "00000000-0000-0000-0000-000000000001",
			ClientSecret:            "test-client-secret",
			ActiveDirectoryEndpoint: "https://login.microsoftonline.com/",
		},
		{
			TenantId:                "00000000-0000-0000-0000-000000000000",
			ClientId:                "00000000-0000-0000-0000-000000000001",
			ClientSecret:            "test-client-secret",
			ResourceManagerEndpoint: azure.GovCloudBaseURI,
			ResourceManagerAudience: "https://management.core.usgovcloudapi.net/",
			ActiveDirectoryEndpoint: "https://login.microsoftonline.us/",
		},
	}

	for _, config := range configs {
		service, err := NewService(config, logptest.NewTestingLogger(t, ""))
		require.NoError(t, err, "billing service must construct both SDK clients with an existing cloud configuration")
		require.NotNil(t, service.usageDetailsClient, "billing usage details client must be constructed")
		require.NotNil(t, service.forecastClient, "billing forecast client must be constructed")
	}

	assert.Equal(t, publicBefore, cloud.AzurePublic.Services, "billing service construction must not mutate the SDK Public cloud")
	assert.Equal(t, governmentBefore, cloud.AzureGovernment.Services, "billing service construction must not mutate the SDK Government cloud")
	assert.Equal(t, chinaBefore, cloud.AzureChina.Services, "billing service construction must not mutate the SDK China cloud")
}
