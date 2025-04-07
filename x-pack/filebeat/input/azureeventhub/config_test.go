// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix

package azureeventhub

import (
	"testing"

	"github.com/stretchr/testify/assert"
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

		if err := config.Validate(); err != nil {
			t.Fatalf("unexpected validation error: %v", err)
		}

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
		config := defaultConfig()
		config.ProcessorVersion = "v1"
		config.ConnectionString = "Endpoint=sb://my-namespace.servicebus.windows.net/;SharedAccessKeyName=my-key;SharedAccessKey=my-secret;EntityPath=my-event-hub;"
		config.EventHubName = "my-event-hub"
		config.SAName = "teststorageaccount"
		config.SAKey = "my-secret"
		config.SAContainer = "filebeat-activitylogs-event_hub_00"

		err := config.Validate()
		require.NoError(t, err, "unexpected validation error)
		

		require.NotNil(t, config.ConnectionStringProperties.EntityPath)
		assert.Equal(t, config.EventHubName, *config.ConnectionStringProperties.EntityPath)
	})

	t.Run("Connection string does not contain entity path", func(t *testing.T) {
		config := defaultConfig()
		config.ProcessorVersion = "v1"
		config.ConnectionString = "Endpoint=sb://my-namespace.servicebus.windows.net/;SharedAccessKeyName=my-key;SharedAccessKey=my-secret"
		config.EventHubName = "my-event-hub"
		config.SAName = "teststorageaccount"
		config.SAKey = "my-secret"
		config.SAContainer = "filebeat-activitylogs-event_hub_00"

		if err := config.Validate(); err != nil {
			t.Fatalf("unexpected validation error: %v", err)
		}

		assert.Nil(t, config.ConnectionStringProperties.EntityPath)
	})
}

func TestValidateConnectionStringV2(t *testing.T) {
	t.Run("Connection string contains entity path", func(t *testing.T) {
		config := defaultConfig()
		config.ProcessorVersion = "v2"
		config.ConnectionString = "Endpoint=sb://my-namespace.servicebus.windows.net/;SharedAccessKeyName=my-key;SharedAccessKey=my-secret;EntityPath=my-event-hub"
		config.EventHubName = "my-event-hub"
		config.SAName = "teststorageaccount"
		config.SAConnectionString = "DefaultEndpointsProtocol=https;AccountName=teststorageaccount;AccountKey=my-secret;EndpointSuffix=core.windows.net"
		config.SAContainer = "filebeat-activitylogs-event_hub_00"

		if err := config.Validate(); err != nil {
			t.Fatalf("unexpected validation error: %v", err)
		}

		require.NotNil(t, config.ConnectionStringProperties.EntityPath)
		assert.Equal(t, config.EventHubName, *config.ConnectionStringProperties.EntityPath)
	})

	t.Run("Connection string does not contain entity path", func(t *testing.T) {
		config := defaultConfig()
		config.ProcessorVersion = "v2"
		config.ConnectionString = "Endpoint=sb://my-namespace.servicebus.windows.net/;SharedAccessKeyName=my-key;SharedAccessKey=my-secret;"
		config.EventHubName = "my-event-hub"
		config.SAName = "teststorageaccount"
		config.SAConnectionString = "DefaultEndpointsProtocol=https;AccountName=teststorageaccount;AccountKey=my-secret;EndpointSuffix=core.windows.net"
		config.SAContainer = "filebeat-activitylogs-event_hub_00"

		if err := config.Validate(); err != nil {
			t.Fatalf("unexpected validation error: %v", err)
		}

		assert.Nil(t, config.ConnectionStringProperties.EntityPath)
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
		if err == nil {
			t.Fatalf("expected validation error")
		}

		assert.NotNil(t, config.ConnectionStringProperties.EntityPath)
		assert.NotEqual(t, *config.ConnectionStringProperties.EntityPath, config.EventHubName)
		assert.ErrorContains(t, err, "invalid connection string: entity path (my-event-hub) does not match event hub name (not-my-event-hub)")
	})
}
