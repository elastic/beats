// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix

package azureeventhub

import (
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"

	"github.com/elastic/elastic-agent-libs/logp"
)

const (
	// AuthTypeConnectionString uses connection string authentication (default).
	AuthTypeConnectionString string = "connection_string"
	// AuthTypeClientSecret uses client secret credentials (OAuth2).
	AuthTypeClientSecret string = "client_secret"
)

// authConfig represents the authentication configuration.
type authConfig struct {
	// AuthType specifies the authentication method to use.
	// If not specified, will be inferred from other fields:
	// - If connection_string is provided, defaults to connection_string
	// - Otherwise, defaults to client_secret
	AuthType string

	// Connection string authentication
	ConnectionString string

	// Client secret authentication
	TenantID      string
	ClientID      string
	ClientSecret  string
	AuthorityHost string
}

// newCredential creates a new TokenCredential based on the configured auth type.
func newCredential(config authConfig, authType string, log *logp.Logger) (azcore.TokenCredential, error) {
	switch authType {
	case AuthTypeConnectionString:
		// Connection string authentication doesn't use TokenCredential
		// This is handled separately in the client creation
		return nil, fmt.Errorf("connection_string authentication does not use TokenCredential")
	case AuthTypeClientSecret:
		// This function is not required right now for only supporting client_secret.
		// But we will need it once we start supporting more auth types.
		return newClientSecretCredential(config, log)
	default:
		return nil, fmt.Errorf("unknown auth_type: %s (valid values: connection_string, client_secret)", authType)
	}
}
