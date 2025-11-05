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
func newEventHubConsumerClient(config eventHubClientConfig, log *logp.Logger) (*azeventhubs.ConsumerClient, error) {
	if config.ConnectionString != "" {
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

	if config.Credential == nil {
		return nil, fmt.Errorf("credential is required when connection_string is not provided")
	}

	// Use credential authentication
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

	if config.Credential == nil {
		return nil, fmt.Errorf("credential is required when connection_string is not provided")
	}

	// Build the storage account URL
	storageAccountURL := fmt.Sprintf("https://%s.blob.core.windows.net/%s", config.StorageAccount, config.Container)

	// Use credential authentication
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
