// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix

package azureeventhub

import (
	"fmt"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/elastic/elastic-agent-libs/logp"
)

// newClientSecretCredential creates a new client secret credential(Oauth2).
func newClientSecretCredential(config authConfig, log *logp.Logger) (azcore.TokenCredential, error) {
	log = log.Named("client_secret")

	if config.TenantID == "" {
		return nil, fmt.Errorf("tenant_id is required for client_secret authentication")
	}
	if config.ClientID == "" {
		return nil, fmt.Errorf("client_id is required for client_secret authentication")
	}
	if config.ClientSecret == "" {
		return nil, fmt.Errorf("client_secret is required for client_secret authentication")
	}

	// Create credential options
	credentialOptions := &azidentity.ClientSecretCredentialOptions{
		ClientOptions: azcore.ClientOptions{
			Cloud: getAzureCloud(config.AuthorityHost),
		},
	}

	// Create the credential
	credential, err := azidentity.NewClientSecretCredential(
		config.TenantID,
		config.ClientID,
		config.ClientSecret,
		credentialOptions,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create client secret credential: %w", err)
	}

	log.Infow("successfully created client secret credential",
		"tenant_id", config.TenantID,
		"client_id", config.ClientID,
	)

	return credential, nil
}

// getAzureCloud returns the appropriate Azure cloud configuration based on the authority host.
func getAzureCloud(authorityHost string) cloud.Configuration {
	switch authorityHost {
	case "https://login.microsoftonline.us":
		return cloud.AzureGovernment
	case "https://login.chinacloudapi.cn":
		return cloud.AzureChina
	default:
		return cloud.AzurePublic
	}
}
