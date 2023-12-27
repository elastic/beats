// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package app_insights

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/preview/appinsights/v1/insights"
	"github.com/Azure/go-autorest/autorest"

	"github.com/elastic/elastic-agent-libs/logp"
)

// AppInsightsService service wrapper to the azure sdk for go
type AppInsightsService struct {
	metricsClient *insights.MetricsClient
	eventClient   *insights.EventsClient
	context       context.Context
	log           *logp.Logger
}

// NewService instantiates the Azure monitoring service
func NewService(config Config) (*AppInsightsService, error) {
	metricsClient := insights.NewMetricsClient()
	metricsClient.Authorizer = autorest.NewAPIKeyAuthorizerWithHeaders(map[string]interface{}{
		"x-api-key": config.ApiKey,
	})
	service := &AppInsightsService{
		metricsClient: &metricsClient,
		context:       context.Background(),
		log:           logp.NewLogger("app insights service"),
	}
	return service, nil
}

// GetMetricValues will return specified app insights metrics
func (service *AppInsightsService) GetMetricValues(applicationId string, bodyMetrics []insights.MetricsPostBodySchema) (insights.ListMetricsResultsItem, error) {
	return service.metricsClient.GetMultiple(service.context, applicationId, bodyMetrics)
}
