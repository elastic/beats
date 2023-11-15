// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azureblobstorage

import (
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	azcontainer "github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/service"

	"github.com/elastic/elastic-agent-libs/logp"
)

func fetchServiceClientAndCreds(cfg config, url string, log *logp.Logger) (*service.Client, *serviceCredentials, error) {
	if cfg.Auth.SharedCredentials != nil {
		return fetchServiceClientWithSharedKeyCreds(url, cfg.AccountName, cfg.Auth.SharedCredentials, log)
	} else if cfg.Auth.ConnectionString != nil {
		return fetchServiceClientWithConnectionString(cfg.Auth.ConnectionString, log)
	}

	return nil, nil, fmt.Errorf("no valid auth specified")
}

func fetchServiceClientWithSharedKeyCreds(url string, accountName string, cfg *sharedKeyConfig, log *logp.Logger) (*service.Client, *serviceCredentials, error) {
	// Creates a default request pipeline using your storage account name and account key.
	credential, err := azblob.NewSharedKeyCredential(accountName, cfg.AccountKey)
	if err != nil {
		log.Errorf("Invalid credentials with error: %v", err)
		return nil, nil, err
	}

	client, err := service.NewClientWithSharedKeyCredential(url, credential, nil)
	if err != nil {
		log.Errorf("Invalid credentials with error: %v", err)
		return nil, nil, err
	}
	return client, &serviceCredentials{sharedKeyCreds: credential, cType: sharedKeyType}, nil
}

func fetchServiceClientWithConnectionString(connectionString *connectionStringConfig, log *logp.Logger) (*service.Client, *serviceCredentials, error) {
	// Creates a default request pipeline using your connection string.
	serviceClient, err := service.NewClientFromConnectionString(connectionString.URI, nil)
	if err != nil {
		log.Errorf("Invalid credentials with error: %v", err)
		return nil, nil, err
	}

	return serviceClient, &serviceCredentials{connectionStrCreds: connectionString.URI, cType: connectionStringType}, nil
}

// fetchBlobClient, generic function that returns a BlobClient based on the credential type
func fetchBlobClient(url string, credential *blobCredentials, log *logp.Logger) (*blob.Client, error) {
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

func fetchBlobClientWithSharedKey(url string, credential *azblob.SharedKeyCredential, log *logp.Logger) (*blob.Client, error) {
	blobClient, err := blob.NewClientWithSharedKeyCredential(url, credential, nil)
	if err != nil {
		log.Errorf("Error fetching blob client for url : %s, error : %v", url, err)
		return nil, err
	}

	return blobClient, nil
}

func fetchBlobClientWithConnectionString(connectionString string, containerName string, blobName string, log *logp.Logger) (*blob.Client, error) {
	blobClient, err := blob.NewClientFromConnectionString(connectionString, containerName, blobName, nil)
	if err != nil {
		log.Errorf("Error fetching blob client for connectionString : %s, error : %v", connectionString, err)
		return nil, err
	}

	return blobClient, nil
}

func fetchContainerClient(serviceClient *service.Client, containerName string, log *logp.Logger) (*azcontainer.Client, error) {
	return serviceClient.NewContainerClient(containerName), nil
}
