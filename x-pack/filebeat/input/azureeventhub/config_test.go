// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix

package azureeventhub

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStorageContainerValidate(t *testing.T) {
	tests := []struct {
		input    string
		errIsNil bool
	}{
		{"a-valid-name", true},
		{"a", false},
		{"a-name-that-is-really-too-long-to-be-valid-and-should-never-be-used-no-matter-what", false},
		{"-not-valid", false},
		{"not-valid-", false},
		{"not--valid", false},
		{"capital-A-not-valid", false},
		{"no_underscores_either", false},
	}
	for _, test := range tests {
		err := storageContainerValidate(test.input)
		if (err == nil) != test.errIsNil {
			t.Errorf("storageContainerValidate(%s) = %v", test.input, err)
		}
	}
}

func TestValidate(t *testing.T) {
	t.Run("Sanitize storage account containers with underscores", func(t *testing.T) {
		config := defaultConfig()
		config.ConnectionString = "Endpoint=sb://test-ns.servicebus.windows.net/;SharedAccessKeyName=RootManageSharedAccessKey;SharedAccessKey=SECRET"
		config.EventHubName = "event_hub_00"
		config.SAName = "teststorageaccount"
		config.SAKey = "secret"
		config.SAContainer = "filebeat-activitylogs-event_hub_00"

		require.NoError(t, config.Validate())

		assert.Equal(
			t,
			"filebeat-activitylogs-event-hub-00",
			config.SAContainer,
			"underscores (_) not replaced with hyphens (-)",
		)
	})
}

func TestValidateConnectionStringV1(t *testing.T) {
	t.Run("Connection string contains entity path", func(t *testing.T) {
		// Check the Validate() function
		config := defaultConfig()
		config.ProcessorVersion = "v1"
		config.ConnectionString = "Endpoint=sb://my-namespace.servicebus.windows.net/;SharedAccessKeyName=my-key;SharedAccessKey=my-secret;EntityPath=my-event-hub;"
		config.EventHubName = "my-event-hub"
		config.SAName = "teststorageaccount"
		config.SAKey = "my-secret"
		config.SAContainer = "filebeat-activitylogs-event_hub_00"
		require.NoError(t, config.Validate())

		// Check the parseConnectionString() function
		connectionStringProperties, err := parseConnectionString(config.ConnectionString)
		require.NoError(t, err)
		require.NotNil(t, connectionStringProperties.EntityPath)
		assert.Equal(t, config.EventHubName, *connectionStringProperties.EntityPath)
	})

	t.Run("Connection string does not contain entity path", func(t *testing.T) {
		// Check the Validate() function
		config := defaultConfig()
		config.ProcessorVersion = "v1"
		config.ConnectionString = "Endpoint=sb://my-namespace.servicebus.windows.net/;SharedAccessKeyName=my-key;SharedAccessKey=my-secret"
		config.EventHubName = "my-event-hub"
		config.SAName = "teststorageaccount"
		config.SAKey = "my-secret"
		config.SAContainer = "filebeat-activitylogs-event_hub_00"
		require.NoError(t, config.Validate())

		// Check the parseConnectionString() function
		connectionStringProperties, err := parseConnectionString(config.ConnectionString)
		require.NoError(t, err)
		require.Nil(t, connectionStringProperties.EntityPath)
	})

	t.Run("Connection string contains entity path but does not match event hub name", func(t *testing.T) {
		config := defaultConfig()
		config.ProcessorVersion = "v1"
		config.ConnectionString = "Endpoint=sb://my-namespace.servicebus.windows.net/;SharedAccessKeyName=my-key;SharedAccessKey=my-secret;EntityPath=my-event-hub"
		config.EventHubName = "not-my-event-hub"
		config.SAName = "teststorageaccount"
		config.SAKey = "my-secret"
		config.SAContainer = "filebeat-activitylogs-event_hub_00"

		err := config.Validate()
		assert.ErrorContains(t, err, "invalid config: the entity path (my-event-hub) in the connection string does not match event hub name (not-my-event-hub)")
	})
}

func TestValidateConnectionStringV2(t *testing.T) {
	t.Run("Connection string contains entity path", func(t *testing.T) {
		// Check the Validate() function
		config := defaultConfig()
		config.ProcessorVersion = "v2"
		config.ConnectionString = "Endpoint=sb://my-namespace.servicebus.windows.net/;SharedAccessKeyName=my-key;SharedAccessKey=my-secret;EntityPath=my-event-hub"
		config.EventHubName = "my-event-hub"
		config.SAName = "teststorageaccount"
		config.SAConnectionString = "DefaultEndpointsProtocol=https;AccountName=teststorageaccount;AccountKey=my-secret;EndpointSuffix=core.windows.net"
		config.SAContainer = "filebeat-activitylogs-event_hub_00"
		require.NoError(t, config.Validate())

		// Check the parseConnectionString() function
		connectionStringProperties, err := parseConnectionString(config.ConnectionString)
		require.NoError(t, err)
		require.NotNil(t, connectionStringProperties.EntityPath)
		require.Equal(t, config.EventHubName, *connectionStringProperties.EntityPath)
	})

	t.Run("Connection string does not contain entity path", func(t *testing.T) {
		// Check the Validate() function
		config := defaultConfig()
		config.ProcessorVersion = "v2"
		config.ConnectionString = "Endpoint=sb://my-namespace.servicebus.windows.net/;SharedAccessKeyName=my-key;SharedAccessKey=my-secret;"
		config.EventHubName = "my-event-hub"
		config.SAName = "teststorageaccount"
		config.SAConnectionString = "DefaultEndpointsProtocol=https;AccountName=teststorageaccount;AccountKey=my-secret;EndpointSuffix=core.windows.net"
		config.SAContainer = "filebeat-activitylogs-event_hub_00"
		require.NoError(t, config.Validate())

		// Check the parseConnectionString() function
		connectionStringProperties, err := parseConnectionString(config.ConnectionString)
		require.NoError(t, err)
		require.Nil(t, connectionStringProperties.EntityPath)
	})

	t.Run("Connection string contains entity path but does not match event hub name", func(t *testing.T) {
		config := defaultConfig()
		config.ProcessorVersion = "v2"
		config.ConnectionString = "Endpoint=sb://my-namespace.servicebus.windows.net/;SharedAccessKeyName=my-key;SharedAccessKey=my-secret;EntityPath=my-event-hub"
		config.EventHubName = "not-my-event-hub"
		config.SAName = "teststorageaccount"
		config.SAConnectionString = "DefaultEndpointsProtocol=https;AccountName=teststorageaccount;AccountKey=my-secret;EndpointSuffix=core.windows.net"
		config.SAContainer = "filebeat-activitylogs-event_hub_00"

		err := config.Validate()
		assert.ErrorContains(t, err, "invalid config: the entity path (my-event-hub) in the connection string does not match event hub name (not-my-event-hub)")
	})
}

func TestOAuth2ConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      azureInputConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid oauth2 config for eventhub (no connection_string)",
			config: func() azureInputConfig {
				c := defaultConfig()
				c.EventHubName = "test-hub"
				c.EventHubNamespace = "test-namespace.servicebus.windows.net"
				c.TenantID = "test-tenant-id"
				c.ClientID = "test-client-id"
				c.ClientSecret = "test-client-secret"
				c.SAName = "test-storage"
				c.SAConnectionString = "test-connection-string"
				c.ProcessorVersion = "v2"
				return c
			}(),
			expectError: false,
		},
		{
			name: "valid oauth2 config for only storage account (no storage_account_connection_string)",
			config: func() azureInputConfig {
				c := defaultConfig()
				c.EventHubName = "test-hub"
				c.ConnectionString = "Endpoint=sb://test.servicebus.windows.net/;SharedAccessKeyName=test;SharedAccessKey=test"
				c.TenantID = "test-tenant-id"
				c.ClientID = "test-client-id"
				c.ClientSecret = "test-client-secret"
				c.SAName = "test-storage"
				c.ProcessorVersion = "v2"
				return c
			}(),
			expectError: false,
		},
		{
			name: "valid oauth2 config for both eventhub and storage account (no connection strings)",
			config: func() azureInputConfig {
				c := defaultConfig()
				c.EventHubName = "test-hub"
				c.EventHubNamespace = "test-namespace.servicebus.windows.net"
				c.TenantID = "test-tenant-id"
				c.ClientID = "test-client-id"
				c.ClientSecret = "test-client-secret"
				c.SAName = "test-storage"
				c.ProcessorVersion = "v2"
				return c
			}(),
			expectError: false,
		},
		{
			name: "oauth2 config missing namespace",
			config: func() azureInputConfig {
				c := defaultConfig()
				c.EventHubName = "test-hub"
				c.TenantID = "test-tenant-id"
				c.ClientID = "test-client-id"
				c.ClientSecret = "test-client-secret"
				c.SAName = "test-storage"
				c.SAConnectionString = "test-connection-string"
				c.ProcessorVersion = "v2"
				return c
			}(),
			expectError: true,
			errorMsg:    "eventhub_namespace is required when connection_string is not provided (OAuth2 authentication)",
		},
		{
			name: "oauth2 config missing tenant_id",
			config: func() azureInputConfig {
				c := defaultConfig()
				c.EventHubName = "test-hub"
				c.EventHubNamespace = "test-namespace.servicebus.windows.net"
				c.ClientID = "test-client-id"
				c.ClientSecret = "test-client-secret"
				c.SAName = "test-storage"
				c.SAConnectionString = "test-connection-string"
				c.ProcessorVersion = "v2"
				return c
			}(),
			expectError: true,
			errorMsg:    "tenant_id is required when connection_string is not provided (OAuth2 authentication)",
		},
		{
			name: "oauth2 config missing client_id",
			config: func() azureInputConfig {
				c := defaultConfig()
				c.EventHubName = "test-hub"
				c.EventHubNamespace = "test-namespace.servicebus.windows.net"
				c.TenantID = "test-tenant-id"
				c.ClientSecret = "test-client-secret"
				c.SAName = "test-storage"
				c.SAConnectionString = "test-connection-string"
				c.ProcessorVersion = "v2"
				return c
			}(),
			expectError: true,
			errorMsg:    "client_id is required when connection_string is not provided (OAuth2 authentication)",
		},
		{
			name: "oauth2 config missing client_secret",
			config: func() azureInputConfig {
				c := defaultConfig()
				c.EventHubName = "test-hub"
				c.EventHubNamespace = "test-namespace.servicebus.windows.net"
				c.TenantID = "test-tenant-id"
				c.ClientID = "test-client-id"
				c.SAName = "test-storage"
				c.SAConnectionString = "test-connection-string"
				c.ProcessorVersion = "v2"
				return c
			}(),
			expectError: true,
			errorMsg:    "client_secret is required when connection_string is not provided (OAuth2 authentication)",
		},
		{
			name: "valid connection_string config",
			config: func() azureInputConfig {
				c := defaultConfig()
				c.EventHubName = "test-hub"
				c.ConnectionString = "Endpoint=sb://test.servicebus.windows.net/;SharedAccessKeyName=test;SharedAccessKey=test"
				c.SAName = "test-storage"
				c.SAConnectionString = "test-connection-string"
				c.ProcessorVersion = "v2"
				return c
			}(),
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.errorMsg != "" && err.Error() != tt.errorMsg {
					t.Errorf("expected error message %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}
