// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package auth

import (
	"fmt"

	"github.com/Azure/go-autorest/autorest/adal"
)

// TokenProvider is the interface that wraps an authentication mechanism and
// allows to obtain tokens.
type TokenProvider interface {
	// Token returns a valid OAuth token, or an error.
	Token() (string, error)

	// Renew must be called to re-authenticate against the oauth2 endpoint if
	// when the API returns an Authentication error.
	Renew() error
}

// servicePrincipalToken extends adal.ServicePrincipalToken with the
// the TokenProvider interface.
type servicePrincipalToken adal.ServicePrincipalToken

// Token returns an oauth token that can be used for bearer authorization.
func (provider *servicePrincipalToken) Token() (string, error) {
	inner := (*adal.ServicePrincipalToken)(provider)
	if err := inner.EnsureFresh(); err != nil {
		return "", fmt.Errorf("refreshing spt token: %w", err)
	}
	token := inner.Token()
	return token.OAuthToken(), nil
}

// Renew re-authenticates with the oauth2 endpoint to get a new Service Principal Token.
func (provider *servicePrincipalToken) Renew() error {
	inner := (*adal.ServicePrincipalToken)(provider)
	return inner.Refresh()
}
