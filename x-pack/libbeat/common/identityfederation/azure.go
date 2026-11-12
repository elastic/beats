// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// Azure identity federation is excluded from FIPS builds because the Azure SDK
// transitively imports golang.org/x/crypto/pkcs12, which uses RC2 and DES —
// algorithms not approved under FIPS 140.  See
// https://github.com/Azure/azure-sdk-for-go/issues/24336.
//
//go:build !requirefips

package identityfederation

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"golang.org/x/oauth2"

	"github.com/elastic/beats/v7/pkg/logp"
)

// AzureWIIAudience is the audience Entra requires on federated client assertions.
const AzureWIIAudience = "api://AzureADTokenExchange"

// AzureParams configures the Azure client assertion credential flow.
//
// Authentication chain:
//  1. Read JWT from JWTFilePath on each token refresh
//  2. ClientAssertionCredential(TenantID, ClientID, JWT) → Azure access token
type AzureParams struct {
	// TenantID is the Azure Active Directory tenant ID.
	TenantID string
	// ClientID is the Azure application (client) ID.
	ClientID string
	// JWTFilePath is the path to the OIDC identity token file.
	// The file is re-read on each token refresh to pick up rotated tokens.
	JWTFilePath string
	// Options are passed directly to azidentity.NewClientAssertionCredential.
	// Use this to configure custom HTTP clients (e.g. for FIPS-compliant TLS).
	Options *azidentity.ClientAssertionCredentialOptions
}

func (p AzureParams) validate() error {
	var errs []error
	if p.TenantID == "" {
		errs = append(errs, errors.New("TenantID is required"))
	}
	if p.ClientID == "" {
		errs = append(errs, errors.New("ClientID is required"))
	}
	if p.JWTFilePath == "" {
		errs = append(errs, errors.New("JWTFilePath is required"))
	}
	if len(errs) > 0 {
		return fmt.Errorf("invalid Azure identity federation params: %w", errors.Join(errs...))
	}
	return nil
}

// AzureNewClientAssertionCredential creates an Azure credential that uses a JWT from
// JWTFilePath as the client assertion. The JWT file is re-read on each token
// refresh so rotated tokens are picked up automatically.
func AzureNewClientAssertionCredential(params AzureParams) (*azidentity.ClientAssertionCredential, error) {
	if err := params.validate(); err != nil {
		return nil, err
	}

	jwtFilePath := params.JWTFilePath
	getAssertion := func(_ context.Context) (string, error) {
		return AzureReadJWT(jwtFilePath)
	}

	return azidentity.NewClientAssertionCredential(params.TenantID, params.ClientID, getAssertion, params.Options)
}

// AzureWIIParams configures the Azure client assertion credential flow with a
// workload-identity-issuer token.
//
// Authentication chain:
//  1. Fetch JWT from the workload-identity issuer via TokenSource on each token refresh
//  2. ClientAssertionCredential(TenantID, ClientID, JWT) → Azure access token
type AzureWIIParams struct {
	// TenantID is the Azure Active Directory tenant ID.
	TenantID string
	// ClientID is the Azure application (client) ID.
	ClientID string
	// TokenSource fetches the JWT from the workload-identity issuer. Its audience must
	// match the audience of the federated identity credential in the customer tenant.
	TokenSource *WIITokenSource
	// Options are passed directly to azidentity.NewClientAssertionCredential.
	Options *azidentity.ClientAssertionCredentialOptions
}

func (p AzureWIIParams) validate() error {
	var errs []error
	if p.TenantID == "" {
		errs = append(errs, errors.New("TenantID is required"))
	}
	if p.ClientID == "" {
		errs = append(errs, errors.New("ClientID is required"))
	}
	if p.TokenSource == nil {
		errs = append(errs, errors.New("TokenSource is required"))
	}
	if len(errs) > 0 {
		return fmt.Errorf("invalid Azure identity federation params: %w", errors.Join(errs...))
	}
	return nil
}

// AzureNewWIIClientAssertionCredential creates an Azure credential that uses a JWT from
// the workload-identity issuer as the client assertion. The token source caches the JWT
// and refreshes it before it expires.
func AzureNewWIIClientAssertionCredential(params AzureWIIParams) (*azidentity.ClientAssertionCredential, error) {
	if err := params.validate(); err != nil {
		return nil, err
	}

	tokenSource := params.TokenSource
	getAssertion := func(ctx context.Context) (string, error) {
		token, err := tokenSource.GetIdentityTokenContext(ctx)
		if err != nil {
			return "", err
		}
		return string(token), nil
	}

	return azidentity.NewClientAssertionCredential(params.TenantID, params.ClientID, getAssertion, params.Options)
}

// AzureNewCloudConnectorCredential creates the Azure credential for a cloud connector.
//
// The identity token source is selected from the environment:
//   - WII path (WORKLOAD_IDENTITY_ISSUER_URL is set): the token is fetched from the
//     workload-identity issuer over mTLS, using WORKLOAD_IDENTITY_SSL_CERT_FILE and
//     WORKLOAD_IDENTITY_SSL_KEY_FILE.
//   - Token file path (CLOUD_CONNECTORS_ID_TOKEN_FILE is set): the token is read
//     from the file on each refresh.
func AzureNewCloudConnectorCredential(tenantID, clientID string, options *azidentity.ClientAssertionCredentialOptions, logger *logp.Logger) (*azidentity.ClientAssertionCredential, error) {
	if issuerURL := os.Getenv(WIIIssuerURLEnvVar); issuerURL != "" {
		certFile := os.Getenv(WIISSLCertFileEnvVar)
		keyFile := os.Getenv(WIISSLKeyFileEnvVar)
		if certFile == "" || keyFile == "" {
			return nil, fmt.Errorf("%s and %s are required when %s is set",
				WIISSLCertFileEnvVar, WIISSLKeyFileEnvVar, WIIIssuerURLEnvVar)
		}
		tokenSource, err := NewWIITokenSource(issuerURL, certFile, keyFile, AzureWIIAudience, logger)
		if err != nil {
			return nil, fmt.Errorf("configuring WII token source: %w", err)
		}
		logger.Info("Using Azure identity federation with the workload-identity issuer")
		return AzureNewWIIClientAssertionCredential(AzureWIIParams{
			TenantID:    tenantID,
			ClientID:    clientID,
			TokenSource: tokenSource,
			Options:     options,
		})
	}

	if jwtFilePath := os.Getenv(CloudConnectorsIDTokenFileEnvVar); jwtFilePath != "" {
		logger.Info("Using Azure identity federation with the cloud connectors token file")
		return AzureNewClientAssertionCredential(AzureParams{
			TenantID:    tenantID,
			ClientID:    clientID,
			JWTFilePath: jwtFilePath,
			Options:     options,
		})
	}

	return nil, fmt.Errorf("no identity token source configured (set %s or %s)",
		WIIIssuerURLEnvVar, CloudConnectorsIDTokenFileEnvVar)
}

// AzureNewCloudConnectorTokenSource returns an OAuth 2.0 token source that serves
// Azure access tokens for scopes, using the cloud connector credential from
// AzureNewCloudConnectorCredential.
func AzureNewCloudConnectorTokenSource(ctx context.Context, tenantID, clientID string, scopes []string, options *azidentity.ClientAssertionCredentialOptions, logger *logp.Logger) (oauth2.TokenSource, error) {
	if len(scopes) == 0 {
		return nil, errors.New("at least one scope is required for an Azure access token")
	}
	cred, err := AzureNewCloudConnectorCredential(tenantID, clientID, options, logger)
	if err != nil {
		return nil, err
	}
	return oauth2.ReuseTokenSource(nil, &azureTokenSource{ctx: ctx, cred: cred, scopes: scopes}), nil
}

type azureTokenSource struct {
	ctx    context.Context
	cred   azcore.TokenCredential
	scopes []string
}

func (s *azureTokenSource) Token() (*oauth2.Token, error) {
	token, err := s.cred.GetToken(s.ctx, policy.TokenRequestOptions{Scopes: s.scopes})
	if err != nil {
		return nil, fmt.Errorf("getting Azure access token: %w", err)
	}
	return &oauth2.Token{
		AccessToken: token.Token,
		TokenType:   "Bearer",
		Expiry:      token.ExpiresOn,
	}, nil
}

// AzureReadJWT reads and validates a JWT token from the given file path.
// It trims whitespace and performs a basic structural check (three dot-separated parts).
func AzureReadJWT(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("reading JWT file %s: %w", filePath, err)
	}

	jwt := strings.TrimSpace(string(data))
	if jwt == "" {
		return "", fmt.Errorf("JWT file %s is empty", filePath)
	}

	// Basic structural validation: JWT must have exactly three dot-separated parts.
	if strings.Count(jwt, ".") != 2 {
		return "", fmt.Errorf("invalid JWT in %s: expected 3 dot-separated parts", filePath)
	}

	return jwt, nil
}
