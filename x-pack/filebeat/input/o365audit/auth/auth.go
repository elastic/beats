// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package auth

// TokenProvider is the interface that wraps an authentication mechanism and
// allows to obtain tokens.
type TokenProvider interface {
	// Token returns a valid OAuth token, or an error.
	Token() (string, error)

	// Renew must be called to re-authenticate against the oauth2 endpoint if
	// when the API returns an Authentication error.
	Renew() error
}
