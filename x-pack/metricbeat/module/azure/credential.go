// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !requirefips

package azure

import (
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"

	"github.com/elastic/beats/v7/pkg/logp"
	"github.com/elastic/beats/v7/x-pack/libbeat/common/identityfederation"
)

// NewCredential returns the Microsoft Entra credential that the config selects.
func NewCredential(config Config, clientOptions policy.ClientOptions, logger *logp.Logger) (azcore.TokenCredential, error) {
	if config.UseCloudConnectors {
		logger.Info("Switching credentials provider to Identity Federation (WII)")
		return identityfederation.AzureNewWIIClientAssertionCredentialFromEnv(config.TenantId, config.ClientId,
			&azidentity.ClientAssertionCredentialOptions{ClientOptions: clientOptions}, logger)
	}
	return azidentity.NewClientSecretCredential(config.TenantId, config.ClientId, config.ClientSecret,
		&azidentity.ClientSecretCredentialOptions{ClientOptions: clientOptions})
}
