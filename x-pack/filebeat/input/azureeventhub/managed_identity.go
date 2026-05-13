// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix

package azureeventhub

import (
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"

	"github.com/elastic/elastic-agent-libs/logp"
)

// newManagedIdentityCredential creates a new managed identity credential.
//
// If ManagedIdentityClientID is set, uses user-assigned managed identity.
// Otherwise, uses system-assigned managed identity.
func newManagedIdentityCredential(config *azureInputConfig, log *logp.Logger) (azcore.TokenCredential, error) {
	log = log.Named("managed_identity")

	// Create credential options
	credentialOptions := &azidentity.ManagedIdentityCredentialOptions{
		ClientOptions: azcore.ClientOptions{
			Cloud: getAzureCloud(config.AuthorityHost),
		},
	}

	// If a client ID is provided, use user-assigned managed identity
	if config.ManagedIdentityClientID != "" {
		credentialOptions.ID = azidentity.ClientID(config.ManagedIdentityClientID)
		log.Infow("using user-assigned managed identity",
			"client_id", config.ManagedIdentityClientID,
		)
	} else {
		log.Infow("using system-assigned managed identity")
	}

	// Create the credential
	credential, err := azidentity.NewManagedIdentityCredential(credentialOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to create managed identity credential: %w", err)
	}

	log.Infow("successfully created managed identity credential")

	return credential, nil
}
