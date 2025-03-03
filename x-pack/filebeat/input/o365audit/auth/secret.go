// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package auth

import (
	"fmt"

	"github.com/Azure/go-autorest/autorest/adal"
)

// NewProviderFromClientSecret returns a token provider that uses a secret
// for authentication.
func NewProviderFromClientSecret(endpoint, resource, applicationID, tenantID, secret string) (p TokenProvider, err error) {
	oauth, err := adal.NewOAuthConfig(endpoint, tenantID)
	if err != nil {
		return nil, fmt.Errorf("error generating OAuthConfig: %w", err)
	}
	spt, err := adal.NewServicePrincipalToken(*oauth, applicationID, secret, resource)
	if err != nil {
		return nil, err
	}
	spt.SetAutoRefresh(true)
	return (*servicePrincipalToken)(spt), nil
}
