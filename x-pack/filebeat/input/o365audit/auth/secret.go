// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package auth

import (
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
)

// NewProviderFromClientSecret returns a token provider that uses a secret
// for authentication.
func NewProviderFromClientSecret(endpoint, resource, applicationID, tenantID, secret string) (p TokenProvider, err error) {
	clientOpts := azcore.ClientOptions{Cloud: cloud.Configuration{ActiveDirectoryAuthorityHost: endpoint}}

	cred, err := azidentity.NewClientSecretCredential(
		tenantID, applicationID, secret, &azidentity.ClientSecretCredentialOptions{ClientOptions: clientOpts},
	)
	if err != nil {
		return nil, err
	}

	return (*credentialTokenProvider)(cred), nil
}
