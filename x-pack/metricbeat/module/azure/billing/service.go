// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package billing

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/consumption/mgmt/2019-10-01/consumption"
	"github.com/Azure/go-autorest/autorest/azure/auth"

	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/azure"
)

// Service offers access to Azure Usage Details and Forecast data.
type Service interface {
	GetForecast(filter string) ([]consumption.Forecast, error)
	GetUsageDetails(
		scope string,
		expand string,
		filter string,
		skipToken string,
		top *int32,
		metricType consumption.Metrictype,
		startDate string,
		endDate string) (consumption.UsageDetailsListResultPage, error)
}

// UsageService is a thin wrapper to the Usage Details API and the Forecast API from the Azure SDK for Go.
type UsageService struct {
	usageDetailsClient *consumption.UsageDetailsClient
	forecastsClient    *consumption.ForecastsClient
	context            context.Context
	log                *logp.Logger
}

// NewService builds a new UsageService using the given config.
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
			// Forecast API returns 404 when the subscription does not support forecasts.
			// For example, at the time of writing, forecasts are only available to
			// enterprises subscriptions:
			//
			// "[Forecasts API] Provides operations to get usage forecasts for Enterprise
			// Subscriptions." [1]
			//
			// [1]: https://docs.microsoft.com/en-us/rest/api/consumption/
			service.log.
				With("billing.filter", filter).
				With("billing.subscription_id", service.forecastsClient.SubscriptionID).
				Warnf(
					"no forecasts available for subscription; possibly because the subscription is not an enterprise subscription. For details, see: https://docs.microsoft.com/en-us/rest/api/consumption/",
				)
			return []consumption.Forecast{}, nil
		default:
			return nil, err
		}
	}

	return *response.Value, nil
}

// GetUsageDetails fetches the usage details for the given filters.
func (service *UsageService) GetUsageDetails(scope string, expand string, filter string, skipToken string, top *int32, metrictype consumption.Metrictype, startDate string, endDate string) (consumption.UsageDetailsListResultPage, error) {
	return service.usageDetailsClient.List(service.context, scope, expand, filter, skipToken, top, metrictype, startDate, endDate)
}
