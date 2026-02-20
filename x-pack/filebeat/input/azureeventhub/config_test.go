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
		config.ProcessorVersion = "v1"
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

	t.Run("Connection string fallback from SA key", func(t *testing.T) {
		// Check the Validate() function
		config := defaultConfig()
		config.ProcessorVersion = "v2"
		config.ConnectionString = "Endpoint=sb://my-namespace.servicebus.windows.net/;SharedAccessKeyName=my-key;SharedAccessKey=my-eh-secret;"
		config.EventHubName = "my-event-hub"
		config.SAName = "teststorageaccount"
		config.SAKey = "my-sa-secret"
		config.SAContainer = "filebeat-activitylogs-event_hub_00"
		require.NoError(t, config.Validate())
		require.Empty(t, config.SAKey)
		// Check the parseConnectionString() function
		connectionStringProperties, err := parseConnectionString(config.ConnectionString)
		require.NoError(t, err)
		require.Nil(t, connectionStringProperties.EntityPath)
		require.Equal(t, "DefaultEndpointsProtocol=https;AccountName=teststorageaccount;AccountKey=my-sa-secret;EndpointSuffix=core.windows.net", config.SAConnectionString)
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

func TestClientSecretConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      azureInputConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid client_secret config for both eventhub and storage account with processor v2",
			config: func() azureInputConfig {
				c := defaultConfig()
				c.EventHubName = "test-hub"
				c.EventHubNamespace = "test-namespace.servicebus.windows.net"
				c.TenantID = "test-tenant-id"
				c.ClientID = "test-client-id"
				c.ClientSecret = "test-client-secret"
				c.SAName = "test-storage"
				c.ProcessorVersion = "v2"
				c.AuthType = "client_secret"
				return c
			}(),
			expectError: false,
		},
		{
			name: "client_secret config missing namespace",
			config: func() azureInputConfig {
				c := defaultConfig()
				c.EventHubName = "test-hub"
				c.TenantID = "test-tenant-id"
				c.ClientID = "test-client-id"
				c.ClientSecret = "test-client-secret"
				c.SAName = "test-storage"
				c.ProcessorVersion = "v2"
				c.AuthType = "client_secret"
				return c
			}(),
			expectError: true,
			errorMsg:    "eventhub_namespace is required when using client_secret authentication",
		},
		{
			name: "client_secret config missing tenant_id",
			config: func() azureInputConfig {
				c := defaultConfig()
				c.EventHubName = "test-hub"
				c.EventHubNamespace = "test-namespace.servicebus.windows.net"
				c.ClientID = "test-client-id"
				c.ClientSecret = "test-client-secret"
				c.SAName = "test-storage"
				c.ProcessorVersion = "v2"
				c.AuthType = "client_secret"
				return c
			}(),
			expectError: true,
			errorMsg:    "tenant_id is required when using client_secret authentication",
		},
		{
			name: "client_secret config missing client_id",
			config: func() azureInputConfig {
				c := defaultConfig()
				c.EventHubName = "test-hub"
				c.EventHubNamespace = "test-namespace.servicebus.windows.net"
				c.TenantID = "test-tenant-id"
				c.ClientSecret = "test-client-secret"
				c.SAName = "test-storage"
				c.ProcessorVersion = "v2"
				c.AuthType = "client_secret"
				return c
			}(),
			expectError: true,
			errorMsg:    "client_id is required when using client_secret authentication",
		},
		{
			name: "client_secret config missing client_secret",
			config: func() azureInputConfig {
				c := defaultConfig()
				c.EventHubName = "test-hub"
				c.EventHubNamespace = "test-namespace.servicebus.windows.net"
				c.TenantID = "test-tenant-id"
				c.ClientID = "test-client-id"
				c.SAName = "test-storage"
				c.ProcessorVersion = "v2"
				c.AuthType = "client_secret"
				return c
			}(),
			expectError: true,
			errorMsg:    "client_secret is required when using client_secret authentication",
		},
		{
			name: "valid client_secret config with processor v1",
			config: func() azureInputConfig {
				c := defaultConfig()
				c.EventHubName = "test-hub"
				c.EventHubNamespace = "test-namespace.servicebus.windows.net"
				c.TenantID = "test-tenant-id"
				c.ClientID = "test-client-id"
				c.ClientSecret = "test-client-secret"
				c.SAName = "test-storage"
				c.SAKey = "test-storage-key"
				c.ProcessorVersion = "v1"
				c.AuthType = "client_secret"
				return c
			}(),
			expectError: false,
		},
		{
			name: "client_secret config with processor v1 missing storage account key",
			config: func() azureInputConfig {
				c := defaultConfig()
				c.EventHubName = "test-hub"
				c.EventHubNamespace = "test-namespace.servicebus.windows.net"
				c.TenantID = "test-tenant-id"
				c.ClientID = "test-client-id"
				c.ClientSecret = "test-client-secret"
				c.SAName = "test-storage"
				c.ProcessorVersion = "v1"
				c.AuthType = "client_secret"
				return c
			}(),
			expectError: true,
			errorMsg:    "storage_account_key is required when using client_secret authentication with processor v1",
		},
		{
			name: "client_secret config with processor v2 uses same credentials for storage account",
			config: func() azureInputConfig {
				c := defaultConfig()
				c.EventHubName = "test-hub"
				c.EventHubNamespace = "test-namespace.servicebus.windows.net"
				c.TenantID = "test-tenant-id"
				c.ClientID = "test-client-id"
				c.ClientSecret = "test-client-secret"
				c.SAName = "test-storage"
				// No SAConnectionString - should use client_secret credentials
				c.ProcessorVersion = "v2"
				c.AuthType = "client_secret"
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

func TestConnectionStringConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      azureInputConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid connection_string config with processor v2",
			config: func() azureInputConfig {
				c := defaultConfig()
				c.EventHubName = "test-hub"
				c.ConnectionString = "Endpoint=sb://test.servicebus.windows.net/;SharedAccessKeyName=test;SharedAccessKey=test"
				c.SAName = "test-storage"
				c.SAConnectionString = "test-connection-string"
				c.ProcessorVersion = "v2"
				c.AuthType = "connection_string"
				return c
			}(),
			expectError: false,
		},
		{
			name: "valid connection_string config without auth_type (defaults to connection_string)",
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
		{
			name: "valid connection_string config with processor v1",
			config: func() azureInputConfig {
				c := defaultConfig()
				c.EventHubName = "test-hub"
				c.ConnectionString = "Endpoint=sb://test.servicebus.windows.net/;SharedAccessKeyName=test;SharedAccessKey=test"
				c.SAName = "test-storage"
				c.SAKey = "test-storage-key"
				c.ProcessorVersion = "v1"
				c.AuthType = "connection_string"
				return c
			}(),
			expectError: false,
		},
		{
			name: "connection_string config missing connection_string",
			config: func() azureInputConfig {
				c := defaultConfig()
				c.EventHubName = "test-hub"
				c.SAName = "test-storage"
				c.SAConnectionString = "test-connection-string"
				c.ProcessorVersion = "v2"
				c.AuthType = "connection_string"
				return c
			}(),
			expectError: true,
			errorMsg:    "connection_string is required when auth_type is empty or set to connection_string",
		},
		{
			name: "connection_string config with processor v1 missing storage account key",
			config: func() azureInputConfig {
				c := defaultConfig()
				c.EventHubName = "test-hub"
				c.ConnectionString = "Endpoint=sb://test.servicebus.windows.net/;SharedAccessKeyName=test;SharedAccessKey=test"
				c.SAName = "test-storage"
				c.ProcessorVersion = "v1"
				c.AuthType = "connection_string"
				return c
			}(),
			expectError: true,
			errorMsg:    "storage_account_key is required when using connection_string authentication with processor v1",
		},
		{
			name: "connection_string config with processor v2 missing storage account connection string",
			config: func() azureInputConfig {
				c := defaultConfig()
				c.EventHubName = "test-hub"
				c.ConnectionString = "Endpoint=sb://test.servicebus.windows.net/;SharedAccessKeyName=test;SharedAccessKey=test"
				c.SAName = "test-storage"
				c.ProcessorVersion = "v2"
				c.AuthType = "connection_string"
				return c
			}(),
			expectError: true,
			errorMsg:    "no storage account connection string configured (config: storage_account_connection_string)",
		},
		{
			name: "invalid auth_type",
			config: func() azureInputConfig {
				c := defaultConfig()
				c.EventHubName = "test-hub"
				c.SAName = "test-storage"
				c.ProcessorVersion = "v2"
				c.AuthType = "invalid_auth_type"
				return c
			}(),
			expectError: true,
			errorMsg:    "unknown auth_type: invalid_auth_type (valid values: connection_string, client_secret, managed_identity)",
		},
		{
			name: "invalid processor_version",
			config: func() azureInputConfig {
				c := defaultConfig()
				c.EventHubName = "test-hub"
				c.ConnectionString = "Endpoint=sb://test.servicebus.windows.net/;SharedAccessKeyName=test;SharedAccessKey=test"
				c.SAName = "test-storage"
				c.SAKey = "test-storage-key"
				c.ProcessorVersion = "v3"
				c.AuthType = "connection_string"
				return c
			}(),
			expectError: true,
			errorMsg:    "invalid processor_version: v3 (available versions: v1, v2)",
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

func TestGetFullyQualifiedEventHubNamespace(t *testing.T) {
	tests := []struct {
		name           string
		config         azureInputConfig
		expectedResult string
		expectError    bool
		errorMsg       string
	}{
		{
			name: "connection_string auth with valid connection string",
			config: func() azureInputConfig {
				c := defaultConfig()
				c.AuthType = AuthTypeConnectionString
				c.ConnectionString = "Endpoint=sb://my-namespace.servicebus.windows.net/;SharedAccessKeyName=my-key;SharedAccessKey=my-secret"
				return c
			}(),
			expectedResult: "my-namespace.servicebus.windows.net",
			expectError:    false,
		},
		{
			name: "connection_string auth with connection string containing entity path",
			config: func() azureInputConfig {
				c := defaultConfig()
				c.AuthType = AuthTypeConnectionString
				c.ConnectionString = "Endpoint=sb://test-ns.servicebus.windows.net/;SharedAccessKeyName=RootManageSharedAccessKey;SharedAccessKey=SECRET;EntityPath=my-event-hub"
				return c
			}(),
			expectedResult: "test-ns.servicebus.windows.net",
			expectError:    false,
		},
		{
			name: "connection_string auth with invalid connection string",
			config: func() azureInputConfig {
				c := defaultConfig()
				c.AuthType = AuthTypeConnectionString
				c.ConnectionString = "InvalidConnectionString"
				return c
			}(),
			expectError: true,
			errorMsg:    "failed to parse connection string",
		},
		{
			name: "connection_string auth with empty connection string",
			config: func() azureInputConfig {
				c := defaultConfig()
				c.AuthType = AuthTypeConnectionString
				c.ConnectionString = ""
				return c
			}(),
			expectError: true,
			errorMsg:    "failed to parse connection string",
		},
		{
			name: "client_secret auth with valid namespace",
			config: func() azureInputConfig {
				c := defaultConfig()
				c.AuthType = AuthTypeClientSecret
				c.EventHubNamespace = "test-namespace.servicebus.windows.net"
				return c
			}(),
			expectedResult: "test-namespace.servicebus.windows.net",
			expectError:    false,
		},
		{
			name: "client_secret auth with empty namespace",
			config: func() azureInputConfig {
				c := defaultConfig()
				c.AuthType = AuthTypeClientSecret
				c.EventHubNamespace = ""
				return c
			}(),
			expectError: true,
			errorMsg:    "eventhub_namespace is required when using client_secret authentication",
		},
		{
			name: "unknown auth type",
			config: func() azureInputConfig {
				c := defaultConfig()
				c.AuthType = "unknown_auth_type"
				return c
			}(),
			expectError: true,
			errorMsg:    "unknown auth_type: unknown_auth_type",
		},
		{
			name: "empty auth type (should default but method doesn't handle default)",
			config: func() azureInputConfig {
				c := defaultConfig()
				c.AuthType = ""
				return c
			}(),
			expectError: true,
			errorMsg:    "unknown auth_type:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.config.GetFullyQualifiedEventHubNamespace()
			if tt.expectError {
				require.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
				assert.Empty(t, result)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedResult, result)
			}
		})
	}
}
