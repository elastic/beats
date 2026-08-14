// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// This file was contributed to by generative AI

package azureblobstorage

import (
	"errors"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	azcontainer "github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/service"

	"github.com/elastic/elastic-agent-libs/logp"
)

func fetchServiceClientAndCreds(cfg config, retryCfg retryConfig, url string, log *logp.Logger) (*service.Client, *serviceCredentials, error) {
	// storageOpts start from the (test-injected) client options and add the retry
	// policy for requests to Azure Storage.
	storageOpts := cfg.clientOptions
	storageOpts.Retry = azureRetryOptions(retryCfg)

	switch {
	case cfg.Auth.SharedCredentials != nil:
		return fetchServiceClientWithSharedKeyCreds(url, cfg.AccountName, cfg.Auth.SharedCredentials, storageOpts, log)
	case cfg.Auth.ConnectionString != nil:
		return fetchServiceClientWithConnectionString(cfg.Auth.ConnectionString, storageOpts, log)
	case cfg.Auth.OAuth2 != nil:
		creds, err := newClientSecretCredential(cfg.Auth.OAuth2, storageOpts)
		if err != nil {
			return nil, nil, err
		}
		return fetchServiceClientWithTokenCreds(url, creds, storageOpts, oauth2Type)
	case cfg.Auth.ManagedIdentity.Enabled:
		// Note the options: this credential gets no retry policy, while the client
		// secret credential above gets the storage one.
		creds, err := newManagedIdentityCredential(cfg.Auth.ManagedIdentity, cfg.clientOptions, log)
		if err != nil {
			return nil, nil, err
		}
		return fetchServiceClientWithTokenCreds(url, creds, storageOpts, managedIdentityType)
	}

	return nil, nil, errors.New("no valid auth specified: configure one of auth.shared_credentials, auth.connection_string, auth.oauth2 or auth.managed_identity")
}

// azureRetryOptions translates the input's retry configuration into the Azure
// SDK's pipeline retry policy. The policy lives in the client pipeline, so these
// settings apply to every request the client makes — listing blobs (pagination)
// as well as downloading them. The values are seeded with the SDK defaults in
// defaultConfig, so a zero value here only occurs when explicitly configured.
func azureRetryOptions(rc retryConfig) policy.RetryOptions {
	return policy.RetryOptions{
		MaxRetries:    rc.MaxRetries,
		RetryDelay:    rc.InitialRetryDelay,
		MaxRetryDelay: rc.MaxRetryDelay,
	}
}

func fetchServiceClientWithSharedKeyCreds(url string, accountName string, cfg *sharedKeyConfig, opts azcore.ClientOptions, log *logp.Logger) (*service.Client, *serviceCredentials, error) {
	// Creates a default request pipeline using your storage account name and account key.
	credential, err := azblob.NewSharedKeyCredential(accountName, cfg.AccountKey)
	if err != nil {
		log.Errorf("Invalid credentials with error: %v", err)
		return nil, nil, err
	}

	client, err := service.NewClientWithSharedKeyCredential(url, credential, &service.ClientOptions{
		ClientOptions: opts,
	})
	if err != nil {
		log.Errorf("Invalid credentials with error: %v", err)
		return nil, nil, err
	}
	return client, &serviceCredentials{sharedKeyCreds: credential, clientOpts: opts, cType: sharedKeyType}, nil
}

func fetchServiceClientWithConnectionString(connectionString *connectionStringConfig, opts azcore.ClientOptions, log *logp.Logger) (*service.Client, *serviceCredentials, error) {
	// Creates a default request pipeline using your connection string.
	serviceClient, err := service.NewClientFromConnectionString(connectionString.URI, &service.ClientOptions{
		ClientOptions: opts,
	})
	if err != nil {
		log.Errorf("Invalid credentials with error: %v", err)
		return nil, nil, err
	}

	return serviceClient, &serviceCredentials{connectionStrCreds: connectionString.URI, clientOpts: opts, cType: connectionStringType}, nil
}

// newClientSecretCredential returns a credential for an Entra ID application.
//
// opts carry the input's retry policy, so the retry block also governs the token
// requests this credential makes. That is how the input has always behaved for
// oauth2. Note that it differs from newManagedIdentityCredential below, where the
// SDK defaults matter.
func newClientSecretCredential(cfg *OAuth2Config, opts azcore.ClientOptions) (*azidentity.ClientSecretCredential, error) {
	creds, err := azidentity.NewClientSecretCredential(cfg.TenantID, cfg.ClientID, cfg.ClientSecret,
		&azidentity.ClientSecretCredentialOptions{ClientOptions: opts},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create client secret credential with oauth2 config: %w", err)
	}
	return creds, nil
}

// newManagedIdentityCredential returns a credential for the managed identity of
// the Azure host that runs Filebeat. An empty ClientID selects the
// system-assigned identity of the host.
//
// opts must not carry the input's retry policy. When the credential reads tokens
// from IMDS, the Azure SDK fills every retry field that is still zero with values
// that suit the token endpoint. Those values are more patient than the ones that
// suit Azure Storage, which matters on a host that has just started and does not
// serve tokens yet. Nothing else covers a failed token request, because the SDK
// marks a credential failure as non-retriable and the storage retry policy
// therefore gives up at once.
func newManagedIdentityCredential(cfg managedIdentityConfig, opts azcore.ClientOptions, log *logp.Logger) (*azidentity.ManagedIdentityCredential, error) {
	credOpts := &azidentity.ManagedIdentityCredentialOptions{ClientOptions: opts}
	if cfg.ClientID != "" {
		credOpts.ID = azidentity.ClientID(cfg.ClientID)
		log.Infow("using user-assigned managed identity", "client_id", cfg.ClientID)
	} else {
		log.Info("using system-assigned managed identity")
	}

	creds, err := azidentity.NewManagedIdentityCredential(credOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to create managed identity credential: %w", err)
	}
	return creds, nil
}

// fetchServiceClientWithTokenCreds builds a service client from an Entra ID token
// credential. The oauth2 and managed identity methods both use it.
func fetchServiceClientWithTokenCreds(url string, creds azcore.TokenCredential, opts azcore.ClientOptions, cType string) (*service.Client, *serviceCredentials, error) {
	client, err := service.NewClient(url, creds, &service.ClientOptions{ClientOptions: opts})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create azblob service client: %w", err)
	}

	return client, &serviceCredentials{tokenCreds: creds, clientOpts: opts, cType: cType}, nil
}

// fetchBlobClient returns a blob client for the credential type in use. The
// client options travel with the credential, so they match the ones used for the
// service client.
func fetchBlobClient(url string, credential *blobCredentials, log *logp.Logger) (*blob.Client, error) {
	if credential == nil {
		return nil, errors.New("no valid blob credentials found")
	}

	creds := credential.serviceCreds
	switch creds.cType {
	case sharedKeyType:
		return fetchBlobClientWithSharedKey(url, creds.sharedKeyCreds, creds.clientOpts, log)
	case connectionStringType:
		return fetchBlobClientWithConnectionString(creds.connectionStrCreds, credential.containerName, credential.blobName, creds.clientOpts, log)
	case oauth2Type, managedIdentityType:
		return fetchBlobClientWithTokenCreds(url, creds.tokenCreds, creds.clientOpts)
	default:
		return nil, fmt.Errorf("no valid service credential 'type' found: %s", creds.cType)
	}
}

func fetchBlobClientWithSharedKey(url string, credential *azblob.SharedKeyCredential, opts azcore.ClientOptions, log *logp.Logger) (*blob.Client, error) {
	blobClient, err := blob.NewClientWithSharedKeyCredential(url, credential, &blob.ClientOptions{
		ClientOptions: opts,
	})
	if err != nil {
		log.Errorf("Error fetching blob client for url: %s, error: %v", url, err)
		return nil, err
	}

	return blobClient, nil
}

func fetchBlobClientWithConnectionString(connectionString string, containerName string, blobName string, opts azcore.ClientOptions, log *logp.Logger) (*blob.Client, error) {
	blobClient, err := blob.NewClientFromConnectionString(connectionString, containerName, blobName, &blob.ClientOptions{
		ClientOptions: opts,
	})
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

// fetchBlobClientWithTokenCreds builds a blob client from an Entra ID token
// credential.
func fetchBlobClientWithTokenCreds(url string, creds azcore.TokenCredential, opts azcore.ClientOptions) (*blob.Client, error) {
	blobClient, err := blob.NewClient(url, creds, &blob.ClientOptions{
		ClientOptions: opts,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch blob client for %s: %w", url, err)
	}

	return blobClient, nil
}

func fetchContainerClient(serviceClient *service.Client, containerName string, log *logp.Logger) (*azcontainer.Client, error) {
	return serviceClient.NewContainerClient(containerName), nil
}
