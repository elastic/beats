// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix

package azureeventhub

import (
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/stretchr/testify/assert"
)

func TestGetAzureCloud(t *testing.T) {
	tests := []struct {
		name          string
		authorityHost string
		expectedCloud cloud.Configuration
	}{
		{
			name:          "Azure Government",
			authorityHost: "https://login.microsoftonline.us",
			expectedCloud: cloud.AzureGovernment,
		},
		{
			name:          "Azure China",
			authorityHost: "https://login.chinacloudapi.cn",
			expectedCloud: cloud.AzureChina,
		},
		{
			name:          "Azure Public Cloud (default)",
			authorityHost: "https://login.microsoftonline.com",
			expectedCloud: cloud.AzurePublic,
		},
		{
			name:          "Empty authority host defaults to Public Cloud",
			authorityHost: "",
			expectedCloud: cloud.AzurePublic,
		},
		{
			name:          "Unknown authority host defaults to Public Cloud",
			authorityHost: "https://login.unknown.cloud",
			expectedCloud: cloud.AzurePublic,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getAzureCloud(tt.authorityHost)
			// Verify that we get a valid cloud configuration with a non-empty authority host
			assert.NotEmpty(t, result.ActiveDirectoryAuthorityHost, "ActiveDirectoryAuthorityHost should not be empty")
			// Verify it matches the expected cloud configuration
			assert.Equal(t, tt.expectedCloud.ActiveDirectoryAuthorityHost, result.ActiveDirectoryAuthorityHost,
				"Authority host should match expected cloud configuration")
		})
	}
}

func TestGetStorageEndpointSuffix(t *testing.T) {
	tests := []struct {
		name          string
		authorityHost string
		expected      string
	}{
		{
			name:          "Azure GovCloud",
			authorityHost: "https://login.microsoftonline.us",
			expected:      "core.usgovcloudapi.net",
		},
		{
			name:          "Azure China",
			authorityHost: "https://login.chinacloudapi.cn",
			expected:      "core.chinacloudapi.cn",
		},
		{
			name:          "Azure Germany",
			authorityHost: "https://login.microsoftonline.de",
			expected:      "core.cloudapi.de",
		},
		{
			name:          "Azure Public Cloud (default)",
			authorityHost: "https://login.microsoftonline.com",
			expected:      "core.windows.net",
		},
		{
			name:          "Empty authority host defaults to Public Cloud",
			authorityHost: "",
			expected:      "core.windows.net",
		},
		{
			name:          "Unknown authority host defaults to Public Cloud",
			authorityHost: "https://login.unknown.cloud",
			expected:      "core.windows.net",
		},
		{
			name:          "Nil/empty string edge case",
			authorityHost: "",
			expected:      "core.windows.net",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getStorageEndpointSuffix(tt.authorityHost)
			assert.Equal(t, tt.expected, result, "Storage endpoint suffix should match expected value")
		})
	}
}
