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

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
)

// AzureWIIParams configures the Azure Workload Identity Federation flow using WII.
//
// Authentication chain:
//  1. POST WII /token via mTLS → short-lived JWT (audience = Azure AD audience)
//  2. ClientAssertionCredential(TenantID, ClientID, JWT) → Azure access token
type AzureWIIParams struct {
	// TenantID is the Azure Active Directory tenant ID.
	TenantID string
	// ClientID is the Azure application (client) ID registered for WIF.
	ClientID string
	// WIIIssuerURL is the WII ECP proxy URL (WORKLOAD_IDENTITY_ISSUER_URL env var).
	WIIIssuerURL string
	// WIICertFile is the path to the WII client certificate (WORKLOAD_IDENTITY_SSL_CERT_FILE env var).
	WIICertFile string
	// WIIKeyFile is the path to the WII client private key (WORKLOAD_IDENTITY_SSL_KEY_FILE env var).
	WIIKeyFile string
	// Audience is the Azure AD audience for the WII token request.
	// Typically "api://AzureADTokenExchange".
	Audience string
	// Options are passed to azidentity.NewClientAssertionCredential.
	Options *azidentity.ClientAssertionCredentialOptions
}

// AzureNewWIIClientAssertionCredential creates an Azure credential that uses a JWT
// from the WII /token endpoint (mTLS) as the client assertion. The token is
// fetched on each credential refresh — no intermediate role is assumed.
func AzureNewWIIClientAssertionCredential(params AzureWIIParams) (*azidentity.ClientAssertionCredential, error) {
	var errs []error
	if params.TenantID == "" {
		errs = append(errs, errors.New("TenantID is required"))
	}
	if params.ClientID == "" {
		errs = append(errs, errors.New("ClientID is required"))
	}
	if params.WIIIssuerURL == "" {
		errs = append(errs, errors.New("WIIIssuerURL is required"))
	}
	if params.WIICertFile == "" || params.WIIKeyFile == "" {
		errs = append(errs, errors.New("WIICertFile and WIIKeyFile are required"))
	}
	if params.Audience == "" {
		errs = append(errs, errors.New("Audience is required"))
	}
	if len(errs) > 0 {
		return nil, fmt.Errorf("invalid AzureWIIParams: %w", errors.Join(errs...))
	}

	tokenSource, err := NewWIITokenSource(params.WIIIssuerURL, params.WIICertFile, params.WIIKeyFile, params.Audience)
	if err != nil {
		return nil, fmt.Errorf("configuring WII token source: %w", err)
	}
	getAssertion := func(_ context.Context) (string, error) {
		token, err := tokenSource.GetIdentityToken()
		if err != nil {
			return "", fmt.Errorf("fetching WII JWT for Azure assertion: %w", err)
		}
		return string(token), nil
	}

	return azidentity.NewClientAssertionCredential(params.TenantID, params.ClientID, getAssertion, params.Options)
}

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
