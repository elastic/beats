// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !requirefips

package app_insights

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/services/preview/appinsights/v1/insights"
	"github.com/Azure/go-autorest/autorest"

	"github.com/elastic/elastic-agent-libs/logp"
)

const (
	// appInsightsScope is the OAuth2 scope for Azure Application Insights API.
	appInsightsScope = "https://api.applicationinsights.io/.default"
)

// AppInsightsService service wrapper to the azure sdk for go
type AppInsightsService struct {
	metricsClient *insights.MetricsClient
	context       context.Context
	log           *logp.Logger
}

// NewService instantiates the Azure monitoring service
func NewService(config Config, logger *logp.Logger) (*AppInsightsService, error) {
	metricsClient := insights.NewMetricsClient()

	authorizer, err := getAuthorizer(config, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create authorizer: %w", err)
	}
	metricsClient.Authorizer = authorizer

	service := &AppInsightsService{
		metricsClient: &metricsClient,
		context:       context.Background(),
		log:           logger.Named("app insights service"),
	}
	return service, nil
}

// getAuthorizer returns the appropriate authorizer based on the config.
// If OAuth2 credentials are provided, it uses OAuth2 authentication.
// Otherwise, it falls back to API key authentication.
func getAuthorizer(config Config, logger *logp.Logger) (autorest.Authorizer, error) {
	// OAuth2 has higher priority than API key
	if config.TenantId != "" && config.ClientId != "" && config.ClientSecret != "" {
		logger.Debug("Using OAuth2 authentication for App Insights")
		return newOAuth2Authorizer(config, logger)
	}

	logger.Debug("Using API key authentication for App Insights")
	return autorest.NewAPIKeyAuthorizerWithHeaders(map[string]interface{}{
		"x-api-key": config.ApiKey,
	}), nil
}

// newOAuth2Authorizer creates an OAuth2 authorizer using azidentity client credentials.
func newOAuth2Authorizer(config Config, logger *logp.Logger) (autorest.Authorizer, error) {
	clientOptions := policy.ClientOptions{}
	if config.ActiveDirectoryEndpoint != "" {
		clientOptions.Cloud = cloud.Configuration{
			ActiveDirectoryAuthorityHost: config.ActiveDirectoryEndpoint,
		}
	}

	credential, err := azidentity.NewClientSecretCredential(
		config.TenantId,
		config.ClientId,
		config.ClientSecret,
		&azidentity.ClientSecretCredentialOptions{
			ClientOptions: clientOptions,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create client secret credential: %w", err)
	}

	logger.Debugf("OAuth2 authorizer created for tenant: %s, client: %s", config.TenantId, config.ClientId)

	return &tokenCredentialAuthorizer{
		credential: credential,
		scopes:     []string{appInsightsScope},
	}, nil
}

// tokenCredentialAuthorizer wraps an azcore.TokenCredential to implement autorest.Authorizer.
// This allows using the modern azidentity package with the legacy autorest-based SDK.
type tokenCredentialAuthorizer struct {
	credential azcore.TokenCredential
	scopes     []string
}

// WithAuthorization implements autorest.Authorizer interface.
func (a *tokenCredentialAuthorizer) WithAuthorization() autorest.PrepareDecorator {
	return func(p autorest.Preparer) autorest.Preparer {
		return autorest.PreparerFunc(func(r *http.Request) (*http.Request, error) {
			// Run the previous preparer in the chain
			r, err := p.Prepare(r)
			if err != nil {
				return r, err
			}

			ctx := r.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			token, err := a.credential.GetToken(ctx, policy.TokenRequestOptions{
				Scopes: a.scopes,
			})
			if err != nil {
				return r, fmt.Errorf("failed to get token: %w", err)
			}

			r.Header.Set("Authorization", "Bearer "+token.Token)
			return r, nil
		})
	}
}

// GetMetricValues will return specified app insights metrics
func (service *AppInsightsService) GetMetricValues(applicationId string, bodyMetrics []insights.MetricsPostBodySchema) (insights.ListMetricsResultsItem, error) {
	return service.metricsClient.GetMultiple(service.context, applicationId, bodyMetrics)
}
