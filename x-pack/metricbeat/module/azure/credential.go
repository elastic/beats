// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !requirefips

package azure

import (
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"

	"github.com/elastic/beats/v7/pkg/logp"
	"github.com/elastic/beats/v7/x-pack/libbeat/common/identityfederation"
)

// NewCredential returns the Azure credential for the configured authentication
// method: identity federation when use_cloud_connectors is enabled, the client
// secret otherwise.
func NewCredential(config Config, clientOptions policy.ClientOptions, logger *logp.Logger) (azcore.TokenCredential, error) {
	if config.UseCloudConnectors {
		cred, err := identityfederation.AzureNewCloudConnectorCredential(config.TenantId, config.ClientId,
			&azidentity.ClientAssertionCredentialOptions{ClientOptions: clientOptions}, logger)
		if err != nil {
			return nil, fmt.Errorf("couldn't create identity federation credentials: %w", err)
		}
		return cred, nil
	}

	cred, err := azidentity.NewClientSecretCredential(config.TenantId, config.ClientId, config.ClientSecret,
		&azidentity.ClientSecretCredentialOptions{ClientOptions: clientOptions})
	if err != nil {
		return nil, fmt.Errorf("couldn't create client credentials: %w", err)
	}
	return cred, nil
}
