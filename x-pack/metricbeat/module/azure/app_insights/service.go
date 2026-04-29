// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !requirefips

package app_insights

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"

	"github.com/elastic/elastic-agent-libs/logp"
)

const (
	// appInsightsEndpoint is the host for the Application Insights v1 query API.
	appInsightsEndpoint = "https://api.applicationinsights.io"
	// appInsightsScope is the OAuth scope for Application Insights when using
	// Microsoft Entra ID (client_secret) authentication.
	appInsightsScope = "https://api.applicationinsights.io/.default"

	// moduleName / moduleVersion are reported in the User-Agent header by the
	// azcore pipeline. The version is informational only and does not need to
	// match the metricbeat release.
	moduleName    = "metricbeat-azure-appinsights"
	moduleVersion = "v1.0.0"
)

// AppInsightsService is a thin wrapper around metricsClient. It is the
// concrete implementation of the Service interface declared in mock_service.go.
type AppInsightsService struct {
	metricsClient *metricsClient
	context       context.Context
	log           *logp.Logger
}

// NewService instantiates the Azure Application Insights service client.
func NewService(config Config, logger *logp.Logger) (*AppInsightsService, error) {
	client, err := newMetricsClient(config, logger)
	if err != nil {
		return nil, err
	}
	return &AppInsightsService{
		metricsClient: client,
		context:       context.Background(),
		log:           logger.Named("app insights service"),
	}, nil
}

// GetMetricValues returns the specified Application Insights metrics.
func (s *AppInsightsService) GetMetricValues(applicationID string, bodyMetrics []MetricsBatchRequestItem) (ListMetricsResultsItem, error) {
	return s.metricsClient.GetMultiple(s.context, applicationID, bodyMetrics)
}

// metricsClient calls the Application Insights v1 batch metrics endpoint
// (POST /v1/apps/{appId}/metrics) using the modern azcore HTTP pipeline.
type metricsClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

// newMetricsClient builds a metricsClient with an authentication policy that
// matches the configured auth_type.
func newMetricsClient(cfg Config, logger *logp.Logger) (*metricsClient, error) {
	clientOpts := &policy.ClientOptions{}

	var perCall, perRetry []policy.Policy
	switch cfg.AuthType {
	case AuthTypeClientSecret:
		logger.Debug("Using client secret authentication for App Insights")
		cred, err := azidentity.NewClientSecretCredential(cfg.TenantId, cfg.ClientId, cfg.ClientSecret, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create client secret credential: %w", err)
		}
		perRetry = append(perRetry, runtime.NewBearerTokenPolicy(cred, []string{appInsightsScope}, nil))
	default:
		logger.Debug("Using API key authentication for App Insights")
		perCall = append(perCall, &apiKeyPolicy{apiKey: cfg.ApiKey})
	}

	pipeline := runtime.NewPipeline(moduleName, moduleVersion, runtime.PipelineOptions{
		PerCall:  perCall,
		PerRetry: perRetry,
	}, clientOpts)

	return &metricsClient{
		endpoint: appInsightsEndpoint,
		pipeline: pipeline,
	}, nil
}

// GetMultiple invokes the batch metrics endpoint and decodes its JSON array
// response into a ListMetricsResultsItem.
func (c *metricsClient) GetMultiple(ctx context.Context, applicationID string, body []MetricsBatchRequestItem) (ListMetricsResultsItem, error) {
	var result ListMetricsResultsItem

	endpoint := fmt.Sprintf("%s/v1/apps/%s/metrics", c.endpoint, url.PathEscape(applicationID))
	req, err := runtime.NewRequest(ctx, http.MethodPost, endpoint)
	if err != nil {
		return result, fmt.Errorf("creating app insights metrics request: %w", err)
	}
	if err := runtime.MarshalAsJSON(req, body); err != nil {
		return result, fmt.Errorf("marshaling app insights metrics request: %w", err)
	}

	resp, err := c.pipeline.Do(req)
	if err != nil {
		return result, fmt.Errorf("calling app insights metrics endpoint: %w", err)
	}
	if !runtime.HasStatusCode(resp, http.StatusOK) {
		return result, runtime.NewResponseError(resp)
	}
	if err := runtime.UnmarshalAsJSON(resp, &result); err != nil {
		return result, fmt.Errorf("decoding app insights metrics response: %w", err)
	}
	return result, nil
}

// apiKeyPolicy injects the Application Insights x-api-key header on every
// outgoing request. It is registered as a per-call policy so it runs once per
// API operation rather than on every retry attempt.
type apiKeyPolicy struct {
	apiKey string
}

// Do implements policy.Policy.
func (p *apiKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("x-api-key", p.apiKey)
	return req.Next()
}
