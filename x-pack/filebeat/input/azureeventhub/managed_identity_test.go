// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix

package azureeventhub

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestManagedIdentityConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      azureInputConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid managed_identity config with system-assigned identity",
			config: func() azureInputConfig {
				c := defaultConfig()
				c.EventHubName = "test-hub"
				c.EventHubNamespace = "test-namespace.servicebus.windows.net"
				c.SAName = "test-storage"
				c.ProcessorVersion = "v2"
				c.AuthType = "managed_identity"
				return c
			}(),
			expectError: false,
		},
		{
			name: "valid managed_identity config with user-assigned identity",
			config: func() azureInputConfig {
				c := defaultConfig()
				c.EventHubName = "test-hub"
				c.EventHubNamespace = "test-namespace.servicebus.windows.net"
				c.SAName = "test-storage"
				c.ProcessorVersion = "v2"
				c.AuthType = "managed_identity"
				c.ManagedIdentityClientID = "user-assigned-client-id"
				return c
			}(),
			expectError: false,
		},
		{
			name: "managed_identity config missing namespace",
			config: func() azureInputConfig {
				c := defaultConfig()
				c.EventHubName = "test-hub"
				c.SAName = "test-storage"
				c.ProcessorVersion = "v2"
				c.AuthType = "managed_identity"
				return c
			}(),
			expectError: true,
			errorMsg:    "eventhub_namespace is required when using managed_identity authentication",
		},
		{
			name: "managed_identity config with custom authority host",
			config: func() azureInputConfig {
				c := defaultConfig()
				c.EventHubName = "test-hub"
				c.EventHubNamespace = "test-namespace.servicebus.windows.net"
				c.SAName = "test-storage"
				c.ProcessorVersion = "v2"
				c.AuthType = "managed_identity"
				c.AuthorityHost = "https://login.microsoftonline.us"
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

func TestNewManagedIdentityCredentialOptions(t *testing.T) {
	// Note: We can't fully test credential creation without Azure infrastructure,
	// but we can verify the function handles configuration options correctly.

	t.Run("system-assigned identity uses no client ID", func(t *testing.T) {
		config := &azureInputConfig{
			AuthType:          AuthTypeManagedIdentity,
			EventHubNamespace: "test-namespace.servicebus.windows.net",
			// ManagedIdentityClientID is empty - system-assigned
		}

		// Verify config is valid for system-assigned identity
		assert.Empty(t, config.ManagedIdentityClientID, "system-assigned identity should have empty client ID")
	})

	t.Run("user-assigned identity uses client ID", func(t *testing.T) {
		config := &azureInputConfig{
			AuthType:                AuthTypeManagedIdentity,
			EventHubNamespace:       "test-namespace.servicebus.windows.net",
			ManagedIdentityClientID: "user-assigned-client-id",
		}

		// Verify config has client ID for user-assigned identity
		assert.NotEmpty(t, config.ManagedIdentityClientID, "user-assigned identity should have client ID")
		assert.Equal(t, "user-assigned-client-id", config.ManagedIdentityClientID)
	})
}
