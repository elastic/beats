// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix
// +build !aix

package azureblobstorage

import (
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/elastic/elastic-agent-libs/logp"
)

func fetchServiceClientAndCreds(config config, url string, log *logp.Logger) (*azblob.ServiceClient, *azblob.SharedKeyCredential, error) {

	// Create a default request pipeline using your storage account name and account key.
	credential, err := azblob.NewSharedKeyCredential(config.AccountName, config.AccountKey)
	if err != nil {
		log.Errorf("Invalid credentials with error: %v", err)
		return nil, nil, err
	}

	serviceClient, err := azblob.NewServiceClientWithSharedKey(url, credential, nil)
	if err != nil {
		log.Errorf("Invalid credentials with error: %v", err)
		return nil, nil, err
	}
	return serviceClient, credential, nil
}

func fetchBlobClients(url string, credential *azblob.SharedKeyCredential, log *logp.Logger) (*azblob.BlockBlobClient, error) {
	blobClient, err := azblob.NewBlockBlobClientWithSharedKey(url, credential, nil)
	if err != nil {
		log.Errorf("Error fetching blob client for url : %s, error : %v", url, err)
		return nil, err
	}

	return blobClient, nil
}
