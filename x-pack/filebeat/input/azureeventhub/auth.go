// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix

package azureeventhub

import (
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"

	"github.com/elastic/elastic-agent-libs/logp"
)

const (
	// AuthTypeConnectionString uses connection string authentication (default).
	AuthTypeConnectionString string = "connection_string"
	// AuthTypeClientSecret uses client secret credentials (OAuth2).
	AuthTypeClientSecret string = "client_secret"
	// AuthTypeManagedIdentity uses Azure Managed Identity authentication.
	AuthTypeManagedIdentity string = "managed_identity"
)

// createCredential creates a TokenCredential if needed based on the authentication type.
// Returns nil for connection_string authentication (which doesn't use credentials).
func createCredential(cfg *azureInputConfig, log *logp.Logger) (azcore.TokenCredential, error) {
	switch cfg.AuthType {
	case AuthTypeConnectionString:
		// No credential needed for connection string authentication
		return nil, nil
	case AuthTypeClientSecret:
		credential, err := newClientSecretCredential(cfg, log)
		if err != nil {
			return nil, fmt.Errorf("failed to create client secret credential: %w", err)
		}
		return credential, nil
	case AuthTypeManagedIdentity:
		credential, err := newManagedIdentityCredential(cfg, log)
		if err != nil {
			return nil, fmt.Errorf("failed to create managed identity credential: %w", err)
		}
		return credential, nil
	default:
		return nil, fmt.Errorf("invalid auth_type: %s", cfg.AuthType)
	}
}

// CreateEventHubConsumerClient creates an Event Hub consumer client
// using the configured authentication method from the provided config.
func CreateEventHubConsumerClient(cfg *azureInputConfig, log *logp.Logger) (*azeventhubs.ConsumerClient, error) {
	// Create the consumer client options
	options := azeventhubs.ConsumerClientOptions{}

	// Set up the transport
	switch cfg.Transport {
	case transportWebsocket:
		// Enable WebSocket transport if configured.
		// This allows connectivity through HTTP proxies and firewalls
		// that block AMQP port 5671 but allow HTTPS on port 443.
		log.Infow("using AMQP-over-WebSocket transport for Event Hub connection")
		options.NewWebSocketConn = newWebSocketConn
	default:
		// Default transport, nothing to do.
		log.Infow("using AMQP transport for Event Hub connection")
	}

	// Set up the consumer client based on the authentication type
	if cfg.AuthType == AuthTypeConnectionString {
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
		connectionStringProperties, err := parseConnectionString(cfg.ConnectionString)
		if err != nil {
			return nil, fmt.Errorf("failed to parse connection string: %w", err)
		}

		// Determine the event hub name to use
		// If the connection string contains an entity path, we need to
		// set the event hub name to an empty string.
		//
		// This is a requirement of the new Event Hub SDK.
		//
		// See: https://github.com/Azure/azure-sdk-for-go/blob/4ece3e50652223bba502f2b73e7f297de34a799c/sdk/messaging/azeventhubs/producer_client.go#L304-L306
		eventHubName := cfg.EventHubName
		if connectionStringProperties.EntityPath != nil {
			eventHubName = ""
		}

		// Use connection string authentication
		consumerClient, err := azeventhubs.NewConsumerClientFromConnectionString(
			cfg.ConnectionString,
			eventHubName,
			cfg.ConsumerGroup,
			&options,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create consumer client from connection string: %w", err)
		}
		return consumerClient, nil
	}

	// All credential-based authentication types (client_secret, managed_identity, etc.)
	credential, err := createCredential(cfg, log)
	if err != nil {
		return nil, err
	}
	if credential == nil {
		return nil, fmt.Errorf("credential cannot be empty when auth_type is %s", cfg.AuthType)
	}

	consumerClient, err := azeventhubs.NewConsumerClient(
		cfg.EventHubNamespace,
		cfg.EventHubName,
		cfg.ConsumerGroup,
		credential,
		&options,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create consumer client with credential: %w", err)
	}

	log.Infow("successfully created consumer client with credential authentication",
		"namespace", cfg.EventHubNamespace,
		"eventhub", cfg.EventHubName,
		"auth_type", cfg.AuthType,
	)

	return consumerClient, nil
}

// CreateStorageAccountContainerClient creates a Storage Account container client
// using the configured authentication method from the provided config.
func CreateStorageAccountContainerClient(cfg *azureInputConfig, log *logp.Logger) (*container.Client, error) {
	if cfg.AuthType == AuthTypeConnectionString {
		// Use connection string authentication
		cloudConfig := getAzureCloud(cfg.AuthorityHost)

		containerClient, err := container.NewClientFromConnectionString(
			cfg.SAConnectionString,
			cfg.SAContainer,
			&container.ClientOptions{
				ClientOptions: azcore.ClientOptions{
					Cloud: cloudConfig,
				},
			},
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create container client from connection string: %w", err)
		}
		return containerClient, nil
	}

	// All credential-based authentication types (client_secret, managed_identity, etc.)
	credential, err := createCredential(cfg, log)
	if err != nil {
		return nil, err
	}
	if credential == nil {
		return nil, fmt.Errorf("credential cannot be empty when auth_type is %s", cfg.AuthType)
	}

	// Get the storage endpoint suffix based on the authority host.
	storageEndpointSuffix := getStorageEndpointSuffix(cfg.AuthorityHost)

	// Build the storage account URL using the correct endpoint suffix for the cloud environment
	storageAccountURL := fmt.Sprintf("https://%s.blob.%s/%s", cfg.SAName, storageEndpointSuffix, cfg.SAContainer)
	containerClient, err := container.NewClient(storageAccountURL, credential, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create container client with credential: %w", err)
	}

	log.Infow("successfully created container client with credential authentication",
		"storage_account", cfg.SAName,
		"container", cfg.SAContainer,
		"auth_type", cfg.AuthType,
	)

	return containerClient, nil
}
