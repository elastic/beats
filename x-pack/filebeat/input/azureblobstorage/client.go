// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azureblobstorage

import (
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	azcontainer "github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/service"

	"github.com/elastic/elastic-agent-libs/logp"
)

func fetchServiceClientAndCreds(cfg config, url string, log *logp.Logger) (*service.Client, *serviceCredentials, error) {
	switch {
	case cfg.Auth.SharedCredentials != nil:
		return fetchServiceClientWithSharedKeyCreds(url, cfg.AccountName, cfg.Auth.SharedCredentials, log)
	case cfg.Auth.ConnectionString != nil:
		return fetchServiceClientWithConnectionString(cfg.Auth.ConnectionString, log)
	case cfg.Auth.OAuth2 != nil:
		return fetchServiceClientWithOAuth2(url, cfg.Auth.OAuth2)
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

func fetchServiceClientWithOAuth2(url string, cfg *OAuth2Config) (*service.Client, *serviceCredentials, error) {
	creds, err := azidentity.NewClientSecretCredential(cfg.TenantID, cfg.ClientID, cfg.ClientSecret, &azidentity.ClientSecretCredentialOptions{
		ClientOptions: cfg.clientOptions,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create client secret credential with oauth2 config: %w", err)
	}

	client, err := azblob.NewClient(url, creds, &azblob.ClientOptions{
		ClientOptions: cfg.clientOptions,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create azblob service client: %w", err)
	}

	return client.ServiceClient(), &serviceCredentials{oauth2Creds: creds, cType: oauth2Type}, nil
}

// fetchBlobClient, generic function that returns a BlobClient based on the credential type
func fetchBlobClient(url string, credential *blobCredentials, cfg config, log *logp.Logger) (*blob.Client, error) {
	if credential == nil {
		return nil, fmt.Errorf("no valid blob credentials found")
	}

	switch credential.serviceCreds.cType {
	case sharedKeyType:
		return fetchBlobClientWithSharedKey(url, credential.serviceCreds.sharedKeyCreds, log)
	case connectionStringType:
		return fetchBlobClientWithConnectionString(credential.serviceCreds.connectionStrCreds, credential.containerName, credential.blobName, log)
	case oauth2Type:
		return fetchBlobClientWithOAuth2(url, credential.serviceCreds.oauth2Creds, cfg.Auth.OAuth2)
	default:
		return nil, fmt.Errorf("no valid service credential 'type' found: %s", credential.serviceCreds.cType)
	}
}

func fetchBlobClientWithSharedKey(url string, credential *azblob.SharedKeyCredential, log *logp.Logger) (*blob.Client, error) {
	blobClient, err := blob.NewClientWithSharedKeyCredential(url, credential, nil)
	if err != nil {
		log.Errorf("Error fetching blob client for url: %s, error: %v", url, err)
		return nil, err
	}

	return blobClient, nil
}

func fetchBlobClientWithConnectionString(connectionString string, containerName string, blobName string, log *logp.Logger) (*blob.Client, error) {
	blobClient, err := blob.NewClientFromConnectionString(connectionString, containerName, blobName, nil)
	if err != nil {
		log.Errorf("Error fetching blob client for connectionString: %s, error: %v", stripKey(connectionString), err)
		return nil, err
	}

	return blobClient, nil
}

// stripKey returns the URI part only of a connection string to remove
// sensitive information. A connection string should look like this:
//
//	sb://dummynamespace.servicebus.windows.net/;SharedAccessKeyName=DummyAccessKeyName;SharedAccessKey=5dOntTRytoC24opYThisAsit3is2B+OGY1US/fuL3ly=
//
// so return only the text before the first semi-colon.
func stripKey(s string) string {
	uri, _, ok := strings.Cut(s, ";")
	if !ok {
		// We expect the string to have the documented format if we reach
		// here something is wrong, so let's stay on the safe side.
		return "(redacted)"
	}
	return uri
}

func fetchBlobClientWithOAuth2(url string, credential *azidentity.ClientSecretCredential, oauth2Cfg *OAuth2Config) (*blob.Client, error) {
	blobClient, err := blob.NewClient(url, credential, &blob.ClientOptions{
		ClientOptions: oauth2Cfg.clientOptions,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch blob client for %s: %w", url, err)
	}

	return blobClient, nil
}

func fetchContainerClient(serviceClient *service.Client, containerName string, log *logp.Logger) (*azcontainer.Client, error) {
	return serviceClient.NewContainerClient(containerName), nil
}
