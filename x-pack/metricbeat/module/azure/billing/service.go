// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package billing

import (
	"context"

	"github.com/Azure/go-autorest/autorest/azure/auth"

	"github.com/Azure/azure-sdk-for-go/services/consumption/mgmt/2019-10-01/consumption"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/azure"
	"github.com/elastic/elastic-agent-libs/logp"
	// prevConsumption "github.com/Azure/azure-sdk-for-go/services/consumption/mgmt/2019-01-01/consumption"
	// "github.com/Azure/azure-sdk-for-go/services/consumption/mgmt/2019-10-01/consumption"
)

// Service interface for the azure monitor service and mock for testing
type Service interface {
	GetForecast(filter string) ([]consumption.Forecast, error)
	GetUsageDetails(scope string, expand string, filter string, skiptoken string, top *int32, metrictype consumption.Metrictype, startDate string, endDate string) (consumption.UsageDetailsListResultPage, error)
}

// BillingService service wrapper to the azure sdk for go
type UsageService struct {
	usageDetailsClient *consumption.UsageDetailsClient
	forecastsClient    *consumption.ForecastsClient
	context            context.Context
	log                *logp.Logger
}

// NewService instantiates the Azure monitoring service
func NewService(config azure.Config) (*UsageService, error) {
	clientConfig := auth.NewClientCredentialsConfig(config.ClientId, config.ClientSecret, config.TenantId)
	clientConfig.AADEndpoint = config.ActiveDirectoryEndpoint
	clientConfig.Resource = config.ResourceManagerEndpoint
	authorizer, err := clientConfig.Authorizer()
	if err != nil {
		return nil, err
	}
	forecastsClient := consumption.NewForecastsClientWithBaseURI(config.ResourceManagerEndpoint, config.SubscriptionId)
	usageDetailsClient := consumption.NewUsageDetailsClientWithBaseURI(config.ResourceManagerEndpoint, config.SubscriptionId)

	forecastsClient.Authorizer = authorizer
	usageDetailsClient.Authorizer = authorizer
	service := UsageService{
		usageDetailsClient: &usageDetailsClient,
		forecastsClient:    &forecastsClient,
		context:            context.Background(),
		log:                logp.NewLogger("azure billing service"),
	}
	return &service, nil
}

// GetForecast fetches the forecast for the given filter.
func (service *UsageService) GetForecast(filter string) ([]consumption.Forecast, error) {
	response, err := service.forecastsClient.List(service.context, filter)
	if err != nil {
		switch response.StatusCode {
		case 404:
			service.log.Warnf(
				"no forecasts found for filter: \"%s\"; root cause: %v",
				filter,
				err,
			)
			return []consumption.Forecast{}, nil
		default:
			return nil, err
		}
	}

	return *response.Value, nil
}

// GetUsageDetails
func (service *UsageService) GetUsageDetails(scope string, expand string, filter string, skipToken string, top *int32, metrictype consumption.Metrictype, startDate string, endDate string) (consumption.UsageDetailsListResultPage, error) {
	return service.usageDetailsClient.List(service.context, scope, expand, filter, skipToken, top, metrictype, startDate, endDate)
}
