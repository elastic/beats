// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix

package azureeventhub

import (
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"

	"github.com/elastic/elastic-agent-libs/logp"
)

// eventHubClientConfig holds configuration for creating an Event Hub consumer client.
type eventHubClientConfig struct {
	Namespace        string
	EventHubName     string
	ConsumerGroup    string
	Credential       azcore.TokenCredential
	ConnectionString string
}

// newEventHubConsumerClient creates a new Event Hub consumer client using the provided credential or connection string.
func newEventHubConsumerClient(config eventHubClientConfig, authType string, log *logp.Logger) (*azeventhubs.ConsumerClient, error) {
	if authType == AuthTypeConnectionString {
		// Use connection string authentication for Event Hub
		// There is a mismatch between how the azure-eventhub input and the new
		// Event Hub SDK expect the event hub name in the connection string.
		//
		// The azure-eventhub input was designed to work with the old Event Hub SDK,
		// which worked using the event hub name in the connection string.
		//
		// The new Event Hub SDK expects clients to pass the event hub name as a
		// parameter, or in the connection string as the entity path.
		//
		// We need to handle both cases.
		connectionStringProperties, err := parseConnectionString(config.ConnectionString)
		if err != nil {
			return nil, fmt.Errorf("failed to parse connection string: %w", err)
		}
		if connectionStringProperties.EntityPath != nil {
			// If the connection string contains an entity path, we need to
			// set the event hub name to an empty string.
			//
			// This is a requirement of the new Event Hub SDK.
			//
			// See: https://github.com/Azure/azure-sdk-for-go/blob/4ece3e50652223bba502f2b73e7f297de34a799c/sdk/messaging/azeventhubs/producer_client.go#L304-L306
			config.EventHubName = ""
		}

		// Use connection string authentication
		consumerClient, err := azeventhubs.NewConsumerClientFromConnectionString(
			config.ConnectionString,
			config.EventHubName,
			config.ConsumerGroup,
			nil,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create consumer client from connection string: %w", err)
		}
		return consumerClient, nil
	}

	// Use credential authentication
	if config.Credential == nil {
		return nil, fmt.Errorf("credential cannot be empty when auth_type is not connection_string")
	}

	consumerClient, err := azeventhubs.NewConsumerClient(
		config.Namespace,
		config.EventHubName,
		config.ConsumerGroup,
		config.Credential,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create consumer client with credential: %w", err)
	}

	log.Infow("successfully created consumer client with credential authentication",
		"namespace", config.Namespace,
		"eventhub", config.EventHubName,
	)

	return consumerClient, nil
}

// storageContainerClientConfig holds configuration for creating a Storage container client.
type storageContainerClientConfig struct {
	StorageAccount   string
	Container        string
	Credential       azcore.TokenCredential
	ConnectionString string
	Cloud            cloud.Configuration
}

// newStorageContainerClient creates a new Storage container client using the provided credential or connection string.
func newStorageContainerClient(config storageContainerClientConfig, authType string, log *logp.Logger) (*container.Client, error) {
	if authType == AuthTypeConnectionString {
		// Use connection string authentication
		if config.Cloud.ActiveDirectoryAuthorityHost == "" {
			config.Cloud = cloud.AzurePublic
		}
		containerClient, err := container.NewClientFromConnectionString(
			config.ConnectionString,
			config.Container,
			&container.ClientOptions{
				ClientOptions: azcore.ClientOptions{
					Cloud: config.Cloud,
				},
			},
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create container client from connection string: %w", err)
		}
		return containerClient, nil
	}

	// Use credential authentication
	if config.Credential == nil {
		return nil, fmt.Errorf("credential cannot be empty when auth_type is not connection_string")
	}

	// Build the storage account URL
	storageAccountURL := fmt.Sprintf("https://%s.blob.core.windows.net/%s", config.StorageAccount, config.Container)
	containerClient, err := container.NewClient(storageAccountURL, config.Credential, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create container client with credential: %w", err)
	}

	log.Infow("successfully created container client with credential authentication",
		"storage_account", config.StorageAccount,
		"container", config.Container,
	)

	return containerClient, nil
}
