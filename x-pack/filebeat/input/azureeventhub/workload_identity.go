// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix

package azureeventhub

import (
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"

	"github.com/elastic/beats/v7/pkg/logp"
)

// newWorkloadIdentityCredential exchanges a projected service account token for Entra tokens.
func newWorkloadIdentityCredential(config *azureInputConfig, log *logp.Logger) (azcore.TokenCredential, error) {
	log = log.Named("workload_identity")
	// Let the SDK read and refresh the projected token file as Kubernetes rotates it.
	credential, err := azidentity.NewWorkloadIdentityCredential(workloadIdentityCredentialOptions(config))
	if err != nil {
		return nil, err
	}
	log.Infow("successfully created workload identity credential")
	return credential, nil
}

// workloadIdentityCredentialOptions leaves unset identity fields for the SDK to resolve from the environment.
func workloadIdentityCredentialOptions(config *azureInputConfig) *azidentity.WorkloadIdentityCredentialOptions {
	options := &azidentity.WorkloadIdentityCredentialOptions{
		ClientID:      config.ClientID,
		TenantID:      config.TenantID,
		TokenFilePath: config.FederatedTokenFile,
		ClientOptions: azcore.ClientOptions{
			Cloud: getAzureCloud(config.AuthorityHost),
		},
	}
	if httpClient := proxyHTTPClient(config.Proxy); httpClient != nil {
		options.Transport = httpClient
	}
	return options
}
