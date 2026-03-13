// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package identity

import (
	"context"
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
)

const defaultTokenTimeout = 30 * time.Second

// TokenProvider retrieves OAuth2 access tokens from Azure AD.
// It is safe for concurrent use; the underlying azidentity credential
// handles token caching and refresh internally.
type TokenProvider struct {
	cred  azcore.TokenCredential
	scope string
}

// NewTokenProvider builds a TokenProvider from the given config.
// When ClientSecret is provided it uses ClientSecretCredential.
// Otherwise it falls back to DefaultAzureCredential which tries
// managed identity, Azure CLI, environment variables, etc.
func NewTokenProvider(cfg Config) (*TokenProvider, error) {
	var cred azcore.TokenCredential
	var err error

	if cfg.ClientSecret != "" {
		cred, err = azidentity.NewClientSecretCredential(cfg.TenantID, cfg.ClientID, cfg.ClientSecret, nil)
	} else {
		cred, err = azidentity.NewDefaultAzureCredential(nil)
	}
	if err != nil {
		return nil, fmt.Errorf("azure identity: failed to create credential: %w", err)
	}

	return &TokenProvider{
		cred:  cred,
		scope: cfg.Scope,
	}, nil
}

// Token retrieves a fresh access token from Azure AD.
func (p *TokenProvider) Token(ctx context.Context) (string, error) {
	tk, err := p.cred.GetToken(ctx, policy.TokenRequestOptions{
		Scopes: []string{p.scope},
	})
	if err != nil {
		return "", fmt.Errorf("azure identity: failed to get token: %w", err)
	}
	return tk.Token, nil
}

// GetIdentityToken implements the stscreds.IdentityTokenRetriever interface
// from the AWS SDK, returning the Azure AD token as bytes.
func (p *TokenProvider) GetIdentityToken() ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTokenTimeout)
	defer cancel()

	token, err := p.Token(ctx)
	if err != nil {
		return nil, err
	}
	return []byte(token), nil
}
