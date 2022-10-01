// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azureblobstorage

import (
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"

	"github.com/elastic/elastic-agent-libs/logp"
)

func fetchServiceClientAndCreds(cfg config, url string, log *logp.Logger) (*azblob.ServiceClient, *serviceCredentials, error) {
	if cfg.Auth.SharedCredentials != nil {
		return fetchServiceClientWithSharedKeyCreds(url, cfg.AccountName, cfg.Auth.SharedCredentials, log)
	} else if cfg.Auth.ConnectionString != nil {
		return fetchServiceClientWithConnectionString(cfg.Auth.ConnectionString, log)
	}

	return nil, nil, fmt.Errorf("no valid auth specified")
}

func fetchServiceClientWithSharedKeyCreds(url string, accountName string, cfg *sharedKeyConfig, log *logp.Logger) (*azblob.ServiceClient, *serviceCredentials, error) {
	// Creates a default request pipeline using your storage account name and account key.
	credential, err := azblob.NewSharedKeyCredential(accountName, cfg.AccountKey)
	if err != nil {
		log.Errorf("Invalid credentials with error: %v", err)
		return nil, nil, err
	}

	client, err := azblob.NewServiceClientWithSharedKey(url, credential, nil)
	if err != nil {
		log.Errorf("Invalid credentials with error: %v", err)
		return nil, nil, err
	}
	return client, &serviceCredentials{sharedKeyCreds: credential, cType: sharedKeyType}, nil
}

func fetchServiceClientWithConnectionString(connectionString *connectionStringConfig, log *logp.Logger) (*azblob.ServiceClient, *serviceCredentials, error) {
	// Creates a default request pipeline using your connection string.
	serviceClient, err := azblob.NewServiceClientFromConnectionString(connectionString.URI, nil)
	if err != nil {
		log.Errorf("Invalid credentials with error: %v", err)
		return nil, nil, err
	}

	return serviceClient, &serviceCredentials{connectionStrCreds: connectionString.URI, cType: connectionStringType}, nil
}

// fetchBlobClient, generic function that returns a BlobClient based on the credential type
func fetchBlobClient(url string, credential *blobCredentials, log *logp.Logger) (*azblob.BlobClient, error) {
	if credential == nil {
		return nil, fmt.Errorf("no valid blob credentials found")
	}

	switch credential.serviceCreds.cType {
	case sharedKeyType:
		return fetchBlobClientWithSharedKey(url, credential.serviceCreds.sharedKeyCreds, log)
	case connectionStringType:
		return fetchBlobClientWithConnectionString(credential.serviceCreds.connectionStrCreds, credential.containerName, credential.blobName, log)
	default:
		return nil, fmt.Errorf("no valid service credential 'type' found: %s", credential.serviceCreds.cType)
	}
}

func fetchBlobClientWithSharedKey(url string, credential *azblob.SharedKeyCredential, log *logp.Logger) (*azblob.BlobClient, error) {
	blobClient, err := azblob.NewBlobClientWithSharedKey(url, credential, nil)
	if err != nil {
		log.Errorf("Error fetching blob client for url : %s, error : %v", url, err)
		return nil, err
	}

	return blobClient, nil
}

func fetchBlobClientWithConnectionString(connectionString string, containerName string, blobName string, log *logp.Logger) (*azblob.BlobClient, error) {
	blobClient, err := azblob.NewBlobClientFromConnectionString(connectionString, containerName, blobName, nil)
	if err != nil {
		log.Errorf("Error fetching blob client for connectionString : %s, error : %v", connectionString, err)
		return nil, err
	}

	return blobClient, nil
}

func fetchContainerClient(serviceClient *azblob.ServiceClient, containerName string, log *logp.Logger) (*azblob.ContainerClient, error) {
	containerClient, err := serviceClient.NewContainerClient(containerName)
	if err != nil {
		log.Errorf("Error fetching container client for container : %s, error : %v", containerName, err)
		return nil, err
	}

	return containerClient, nil
}
