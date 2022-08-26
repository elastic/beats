// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix
// +build !aix

package gcs

import (
	"context"
	"fmt"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"

	"github.com/elastic/elastic-agent-libs/logp"
)

func fetchStorageClient(ctx context.Context, cfg config, log *logp.Logger) (*storage.Client, error) {
	if cfg.Auth.CredentialsJSON != nil {
		return storage.NewClient(ctx, option.WithCredentialsJSON([]byte(cfg.Auth.CredentialsJSON.AccountKey)))
	} else if cfg.Auth.CredentialsFile != nil {
		return storage.NewClient(ctx, option.WithCredentialsFile(cfg.Auth.CredentialsFile.Path))
	}
	return nil, fmt.Errorf("no valid auth specified")
}

// func fetchServiceClientWithSharedKeyCreds(url string, accountName string, cfg *sharedKeyConfig, log *logp.Logger) (*azblob.ServiceClient, *types.ServiceCredentials, error) {
// 	// Creates a default request pipeline using your storage account name and account key.
// 	credential, err := azblob.NewSharedKeyCredential(accountName, cfg.AccountKey)
// 	if err != nil {
// 		log.Errorf("Invalid credentials with error: %v", err)
// 		return nil, nil, err
// 	}

// 	client, err := azblob.NewServiceClientWithSharedKey(url, credential, nil)
// 	if err != nil {
// 		log.Errorf("Invalid credentials with error: %v", err)
// 		return nil, nil, err
// 	}
// 	return client, &types.ServiceCredentials{SharedKeyCreds: credential, Ctype: types.SharedKeyType}, nil
// }

// func fetchServiceClientWithConnectionString(connectionString *connectionStringConfig, log *logp.Logger) (*azblob.ServiceClient, *types.ServiceCredentials, error) {
// 	// Creates a default request pipeline using your connection string.
// 	serviceClient, err := azblob.NewServiceClientFromConnectionString(connectionString.URI, nil)
// 	if err != nil {
// 		log.Errorf("Invalid credentials with error: %v", err)
// 		return nil, nil, err
// 	}

// 	return serviceClient, &types.ServiceCredentials{ConnectionStrCreds: connectionString.URI, Ctype: types.ConnectionStringType}, nil
// }

// fetchBlobClient , generic function that returns a BlobClient based on the credential type
// func fetchBlobClient(url string, credential *types.BlobCredentials, log *logp.Logger) (*azblob.BlobClient, error) {
// 	if credential == nil {
// 		return nil, fmt.Errorf("no valid blob credentials found")
// 	}

// 	switch credential.ServiceCreds.Ctype {
// 	case types.SharedKeyType:
// 		return fetchBlobClientWithSharedKey(url, credential.ServiceCreds.SharedKeyCreds, log)
// 	case types.ConnectionStringType:
// 		return fetchBlobClientWithConnectionString(credential.ServiceCreds.ConnectionStrCreds, credential.ContainerName, credential.BlobName, log)
// 	default:
// 		return nil, fmt.Errorf("no valid service credential 'type' found")
// 	}
// }

// func fetchBlobClientWithSharedKey(url string, credential *azblob.SharedKeyCredential, log *logp.Logger) (*azblob.BlobClient, error) {
// 	blobClient, err := azblob.NewBlobClientWithSharedKey(url, credential, nil)
// 	if err != nil {
// 		log.Errorf("Error fetching blob client for url : %s, error : %v", url, err)
// 		return nil, err
// 	}

// 	return blobClient, nil
// }

// func fetchBlobClientWithConnectionString(connectionString string, containerName string, blobName string, log *logp.Logger) (*azblob.BlobClient, error) {
// 	blobClient, err := azblob.NewBlobClientFromConnectionString(connectionString, containerName, blobName, nil)
// 	if err != nil {
// 		log.Errorf("Error fetching blob client for connectionString : %s, error : %v", connectionString, err)
// 		return nil, err
// 	}

// 	return blobClient, nil
// }

// func fetchContainerClient(serviceClient *azblob.ServiceClient, containerName string, log *logp.Logger) (*azblob.ContainerClient, error) {
// 	containerClient, err := serviceClient.NewContainerClient(containerName)
// 	if err != nil {
// 		log.Errorf("Error fetching container client for container : %s, error : %v", containerName, err)
// 		return nil, err
// 	}

// 	return containerClient, nil
// }
