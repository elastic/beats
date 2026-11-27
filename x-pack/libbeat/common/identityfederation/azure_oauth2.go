// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package identityfederation

import (
	"context"
	"fmt"
	"maps"
	"net/url"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"

	"github.com/elastic/beats/v7/pkg/logp"
)

// AzureTokenExchangeAudience is the audience Microsoft Entra expects in a
// federated identity credential.
const AzureTokenExchangeAudience = "api://AzureADTokenExchange" //nolint:gosec // G101: a public audience URI, not a credential

const azureClientAssertionType = "urn:ietf:params:oauth:client-assertion-type:jwt-bearer"

// AzureNewWIIOAuth2TokenSourceFromEnv gets Entra tokens with a WII JWT as the client assertion.
// It does not use azidentity, so it is also available in FIPS builds.
func AzureNewWIIOAuth2TokenSourceFromEnv(ctx context.Context, tokenURL, clientID string, scopes []string, params url.Values, logger *logp.Logger) (oauth2.TokenSource, error) {
	wii, err := azureWIITokenSourceFromEnv(logger)
	if err != nil {
		return nil, err
	}
	return oauth2.ReuseTokenSource(nil, azureWIIAssertionSource{
		ctx:      ctx,
		wii:      wii,
		tokenURL: tokenURL,
		clientID: clientID,
		scopes:   scopes,
		params:   params,
	}), nil
}

func azureWIITokenSourceFromEnv(logger *logp.Logger) (*WIITokenSource, error) {
	wiiIssuerURL := os.Getenv(WIIIssuerURLEnvVar)
	wiiCertFile := os.Getenv(WIISSLCertFileEnvVar)
	wiiKeyFile := os.Getenv(WIISSLKeyFileEnvVar)

	if wiiIssuerURL == "" {
		return nil, fmt.Errorf("identity federation for Azure requires the workload-identity-issuer environment set by the agentless controller: %s is not set", WIIIssuerURLEnvVar)
	}
	if wiiCertFile == "" || wiiKeyFile == "" {
		return nil, fmt.Errorf("%s and %s must be set when %s is configured", WIISSLCertFileEnvVar, WIISSLKeyFileEnvVar, WIIIssuerURLEnvVar)
	}
	tokenSource, err := NewWIITokenSource(wiiIssuerURL, wiiCertFile, wiiKeyFile, AzureTokenExchangeAudience, logger)
	if err != nil {
		return nil, fmt.Errorf("configuring WII token source: %w", err)
	}
	return tokenSource, nil
}

type azureWIIAssertionSource struct {
	ctx      context.Context
	wii      *WIITokenSource
	tokenURL string
	clientID string
	scopes   []string
	params   url.Values
}

// Token builds the request params on each call because the WII JWT expires.
func (s azureWIIAssertionSource) Token() (*oauth2.Token, error) {
	jwt, err := s.wii.GetIdentityToken()
	if err != nil {
		return nil, fmt.Errorf("getting workload-identity token: %w", err)
	}
	params := url.Values{}
	maps.Copy(params, s.params)
	params.Set("client_assertion_type", azureClientAssertionType)
	params.Set("client_assertion", string(jwt))

	cfg := clientcredentials.Config{
		ClientID:       s.clientID,
		TokenURL:       s.tokenURL,
		Scopes:         s.scopes,
		EndpointParams: params,
		AuthStyle:      oauth2.AuthStyleInParams,
	}
	return cfg.Token(s.ctx)
}
