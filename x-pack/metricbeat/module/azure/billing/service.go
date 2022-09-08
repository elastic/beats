// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package billing

import (
	"context"
	"time"

	"github.com/Azure/go-autorest/autorest/date"

	"github.com/Azure/azure-sdk-for-go/services/consumption/mgmt/2019-10-01/consumption"
	//"github.com/Azure/azure-sdk-for-go/services/costmanagement/mgmt/2019-11-01/costmanagement"
	"github.com/Azure/go-autorest/autorest/azure/auth"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/azure"
	"github.com/elastic/elastic-agent-libs/logp"

	"github.com/Azure/azure-sdk-for-go/services/costmanagement/mgmt/2019-11-01/costmanagement"
)

// Service offers access to Azure Usage Details and Forecast data.
type Service interface {
	GetForecast(scope string, startTime, endTime time.Time) (costmanagement.QueryResult, error)
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
	forecastClient     *costmanagement.ForecastClient
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

	usageDetailsClient := consumption.NewUsageDetailsClientWithBaseURI(config.ResourceManagerEndpoint, config.SubscriptionId)
	forecastsClient := costmanagement.NewForecastClientWithBaseURI(config.ResourceManagerEndpoint, config.SubscriptionId)

	usageDetailsClient.Authorizer = authorizer
	forecastsClient.Authorizer = authorizer

	service := UsageService{
		usageDetailsClient: &usageDetailsClient,
		forecastClient:     &forecastsClient,
		context:            context.Background(),
		log:                logp.NewLogger("azure billing service"),
	}

	return &service, nil
}

// GetForecast fetches the forecast for the given scope and time interval.
func (service *UsageService) GetForecast(scope string, startTime, endTime time.Time) (costmanagement.QueryResult, error) {
	// With this flag, the Forecast API will also return actual usage data
	// for the given time interval (usually the current month).
	//
	// We can get both "Actual" and "Forecast" data from the same API call.
	includeActualCost := true

	// With this flag, the Forecast API will include "freshpartialCost" the response. This means we'll find
	// both "Forecast" and "Actual" mixed data for the same usage date.
	//
	// The current dashboard is designed to use final costs only (it averages actual/forecasts values), so we are
	// setting this flag to false for now. The downside is final data are available with a one-day delay.
	includeFreshPartialCost := false

	// The aggregation is performed by the "sum" of "cost" for each day.
	aggregationName := "Cost"
	aggregationFunction := costmanagement.FunctionTypeSum

	forecastDefinition := costmanagement.ForecastDefinition{
		Dataset: &costmanagement.QueryDataset{
			Aggregation: map[string]*costmanagement.QueryAggregation{
				"totalCost": {
					Function: aggregationFunction,
					Name:     &aggregationName,
				},
			},
			Granularity: costmanagement.GranularityTypeDaily,
		},

		// Time frame/period of the forecast. Required for MCA accounts.
		//
		// If omitted, EA users will get a forecast for the current month, and
		// MCA users will get an error.
		Timeframe: costmanagement.ForecastTimeframeTypeCustom,
		TimePeriod: &costmanagement.QueryTimePeriod{
			From: &date.Time{Time: startTime},
			To:   &date.Time{Time: endTime},
		},

		Type:                    costmanagement.ForecastTypeActualCost,
		IncludeActualCost:       &includeActualCost,
		IncludeFreshPartialCost: &includeFreshPartialCost,
	}

	// required, but I don't have a use for it, yet.
	filter := ""

	queryResult, err := service.forecastClient.Usage(service.context, scope, forecastDefinition, filter)
	if err != nil {
		return costmanagement.QueryResult{}, err
	}

	return queryResult, nil
}

// GetUsageDetails fetches the usage details for the given filters.
func (service *UsageService) GetUsageDetails(scope string, expand string, filter string, skipToken string, top *int32, metrictype consumption.Metrictype, startDate string, endDate string) (consumption.UsageDetailsListResultPage, error) {
	return service.usageDetailsClient.List(service.context, scope, expand, filter, skipToken, top, metrictype, startDate, endDate)
}
