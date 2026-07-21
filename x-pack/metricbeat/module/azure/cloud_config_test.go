// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !requirefips

package azure

import (
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/Azure/azure-sdk-for-go/sdk/monitor/query/azmetrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildCloudConfig(t *testing.T) {
	t.Run("empty endpoint defaults to public cloud", func(t *testing.T) {
		cfg := BuildCloudConfig(Config{ActiveDirectoryEndpoint: "https://login.microsoftonline.com/"})

		rm := cfg.Services[cloud.ResourceManager]
		assert.Equal(t, "https://management.azure.com", rm.Endpoint)
		assert.Equal(t, "https://management.core.windows.net/", rm.Audience)
		assert.Equal(t, "https://metrics.monitor.azure.com", cfg.Services[azmetrics.ServiceName].Audience)
		assert.Equal(t, "https://login.microsoftonline.com/", cfg.ActiveDirectoryAuthorityHost)
	})

	t.Run("default endpoint keeps public cloud values", func(t *testing.T) {
		cfg := BuildCloudConfig(Config{
			ResourceManagerEndpoint: DefaultBaseURI,
			ActiveDirectoryEndpoint: "https://login.microsoftonline.com/",
		})

		rm := cfg.Services[cloud.ResourceManager]
		assert.Equal(t, "https://management.azure.com", rm.Endpoint)
		assert.Equal(t, "https://management.core.windows.net/", rm.Audience)
	})

	t.Run("government endpoint selects government cloud", func(t *testing.T) {
		cfg := BuildCloudConfig(Config{
			ResourceManagerEndpoint: "https://management.usgovcloudapi.net/",
			ActiveDirectoryEndpoint: "https://login.microsoftonline.us/",
		})

		rm := cfg.Services[cloud.ResourceManager]
		assert.Equal(t, "https://management.usgovcloudapi.net/", rm.Endpoint)
		assert.Equal(t, "https://management.core.usgovcloudapi.net/", rm.Audience)
		assert.Equal(t, "https://metrics.monitor.azure.us", cfg.Services[azmetrics.ServiceName].Audience)
		assert.Equal(t, "https://login.microsoftonline.us/", cfg.ActiveDirectoryAuthorityHost)
	})

	t.Run("china endpoint selects china cloud", func(t *testing.T) {
		cfg := BuildCloudConfig(Config{
			ResourceManagerEndpoint: "https://management.chinacloudapi.cn/",
			ActiveDirectoryEndpoint: "https://login.chinacloudapi.cn/",
		})

		rm := cfg.Services[cloud.ResourceManager]
		assert.Equal(t, "https://management.chinacloudapi.cn/", rm.Endpoint)
		assert.Equal(t, "https://management.core.chinacloudapi.cn/", rm.Audience)
		assert.Equal(t, "https://metrics.monitor.azure.cn", cfg.Services[azmetrics.ServiceName].Audience)
	})

	t.Run("explicit audience overrides the cloud default", func(t *testing.T) {
		cfg := BuildCloudConfig(Config{
			ResourceManagerEndpoint: "https://management.local.azurestack.external/",
			ResourceManagerAudience: "https://management.adfs.azurestack.local/abc-123",
			ActiveDirectoryEndpoint: "https://adfs.local.azurestack.external/",
		})

		rm := cfg.Services[cloud.ResourceManager]
		assert.Equal(t, "https://management.local.azurestack.external/", rm.Endpoint)
		assert.Equal(t, "https://management.adfs.azurestack.local/abc-123", rm.Audience)
	})

	t.Run("does not mutate the SDK global cloud configurations", func(t *testing.T) {
		publicBefore := cloud.AzurePublic.Services[cloud.ResourceManager]
		govBefore := cloud.AzureGovernment.Services[cloud.ResourceManager]

		cfg := BuildCloudConfig(Config{
			ResourceManagerEndpoint: "https://management.usgovcloudapi.net/",
			ResourceManagerAudience: "https://management.core.usgovcloudapi.net/",
			ActiveDirectoryEndpoint: "https://login.microsoftonline.us/",
		})

		// mutating the returned map must not leak into the globals either
		cfg.Services[cloud.ResourceManager] = cloud.ServiceConfiguration{}

		assert.Equal(t, publicBefore, cloud.AzurePublic.Services[cloud.ResourceManager])
		assert.Equal(t, govBefore, cloud.AzureGovernment.Services[cloud.ResourceManager])
	})
}

func TestMetricsBatchEndpoint(t *testing.T) {
	t.Run("public cloud", func(t *testing.T) {
		for _, endpoint := range []string{"", DefaultBaseURI} {
			got, err := metricsBatchEndpoint(endpoint, "westeurope")
			require.NoError(t, err)
			assert.Equal(t, "https://westeurope.metrics.monitor.azure.com", got)
		}
	})

	t.Run("government cloud", func(t *testing.T) {
		got, err := metricsBatchEndpoint("https://management.usgovcloudapi.net/", "usgovvirginia")
		require.NoError(t, err)
		assert.Equal(t, "https://usgovvirginia.metrics.monitor.azure.us", got)
	})

	t.Run("china cloud", func(t *testing.T) {
		got, err := metricsBatchEndpoint("https://management.chinacloudapi.cn/", "chinaeast2")
		require.NoError(t, err)
		assert.Equal(t, "https://chinaeast2.metrics.monitor.azure.cn", got)
	})

	t.Run("unknown cloud fails loudly", func(t *testing.T) {
		_, err := metricsBatchEndpoint("https://management.local.azurestack.external/", "local")
		assert.Error(t, err)
	})
}
