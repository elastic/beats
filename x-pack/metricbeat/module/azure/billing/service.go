// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package billing

import (
	"context"
	"fmt"
	"time"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/azure"
	"github.com/elastic/elastic-agent-libs/logp"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/consumption/armconsumption"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/costmanagement/armcostmanagement"
)

// Service offers access to Azure Usage Details and Forecast data.
type Service interface {
	GetForecast(
		scope string,
		startTime,
		endTime time.Time,
	) (armcostmanagement.QueryResult, error)
	GetUsageDetails(
		scope string,
		expand string,
		filter string,
		metricType armconsumption.Metrictype,
		startDate string,
		endDate string,
	) (armconsumption.UsageDetailsListResult, error)
}

// UsageService is a thin wrapper to the Usage Details API and the Forecast API from the Azure SDK for Go.
type UsageService struct {
	usageDetailsClient *armconsumption.UsageDetailsClient
	forecastClient     *armcostmanagement.ForecastClient
	context            context.Context
	log                *logp.Logger
}

// NewService builds a new UsageService using the given config.
func NewService(config azure.Config) (*UsageService, error) {
	cloudServicesConfig := cloud.AzurePublic.Services

	resourceManagerConfig := cloudServicesConfig[cloud.ResourceManager]

	if config.ResourceManagerEndpoint != "" && config.ResourceManagerEndpoint != azure.DefaultBaseURI {
		resourceManagerConfig.Endpoint = config.ResourceManagerEndpoint
	}

	if config.ResourceManagerAudience != "" {
		resourceManagerConfig.Audience = config.ResourceManagerAudience
	}

	cloudServicesConfig[cloud.ResourceManager] = resourceManagerConfig

	clientOptions := policy.ClientOptions{
		Cloud: cloud.Configuration{
			Services:                     cloudServicesConfig,
			ActiveDirectoryAuthorityHost: config.ActiveDirectoryEndpoint,
		},
	}

	credential, err := azidentity.NewClientSecretCredential(config.TenantId, config.ClientId, config.ClientSecret, &azidentity.ClientSecretCredentialOptions{
		ClientOptions: clientOptions,
	})
	if err != nil {
		return nil, fmt.Errorf("couldn't create client credentials: %w", err)
	}

	usageDetailsClient, err := armconsumption.NewUsageDetailsClient(credential, &arm.ClientOptions{
		ClientOptions: clientOptions,
	})
	if err != nil {
		return nil, fmt.Errorf("couldn't create usage details client: %w", err)
	}

	forecastsClient, err := armcostmanagement.NewForecastClient(credential, &arm.ClientOptions{
		ClientOptions: clientOptions,
	})
	if err != nil {
		return nil, fmt.Errorf("couldn't create forecast client: %w", err)
	}

	service := UsageService{
		usageDetailsClient: usageDetailsClient,
		forecastClient:     forecastsClient,
		context:            context.Background(),
		log:                logp.NewLogger("azure billing service"),
	}

	return &service, nil
}

// GetForecast fetches the forecast for the given scope and time interval.
func (service *UsageService) GetForecast(
	scope string,
	startTime,
	endTime time.Time,
) (armcostmanagement.QueryResult, error) {
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
	aggregationFunction := armcostmanagement.FunctionTypeSum

	granularityDaily := armcostmanagement.GranularityTypeDaily

	forecastTimeframeCustom := armcostmanagement.ForecastTimeframeTypeCustom
	forecastTypeActualCost := armcostmanagement.ForecastTypeActualCost

	forecastDefinition := armcostmanagement.ForecastDefinition{
		Dataset: &armcostmanagement.ForecastDataset{
			Aggregation: map[string]*armcostmanagement.QueryAggregation{
				"totalCost": {
					Function: &aggregationFunction,
					Name:     &aggregationName,
				},
			},
			Granularity: &granularityDaily,
		},

		// Time frame/period of the forecast. Required for MCA accounts.
		//
		// If omitted, EA users will get a forecast for the current month, and
		// MCA users will get an error.
		Timeframe: &forecastTimeframeCustom,
		TimePeriod: &armcostmanagement.QueryTimePeriod{
			From: &startTime,
			To:   &endTime,
		},

		Type:                    &forecastTypeActualCost,
		IncludeActualCost:       &includeActualCost,
		IncludeFreshPartialCost: &includeFreshPartialCost,
	}

	// required, but I don't have a use for it, yet.
	filter := ""

	queryResult, err := service.forecastClient.Usage(service.context, scope, forecastDefinition, &armcostmanagement.ForecastClientUsageOptions{
		Filter: &filter,
	})
	if err != nil {
		return armcostmanagement.QueryResult{}, err
	}

	return queryResult.QueryResult, nil
}

// GetUsageDetails fetches the usage details for the given filters.
func (service *UsageService) GetUsageDetails(
	scope string,
	expand string,
	filter string,
	metrictype armconsumption.Metrictype,
	startDate string,
	endDate string,
) (armconsumption.UsageDetailsListResult, error) {
	pager := service.usageDetailsClient.NewListPager(scope, &armconsumption.UsageDetailsClientListOptions{
		Expand:    &expand,
		Filter:    &filter,
		Metric:    &metrictype,
		StartDate: &startDate,
		EndDate:   &endDate,
	})

	usageDetails := armconsumption.UsageDetailsListResult{}

	for pager.More() {
		nextPage, err := pager.NextPage(service.context)
		if err != nil {
			return armconsumption.UsageDetailsListResult{}, err
		}

		usageDetails.Value = append(usageDetails.Value, nextPage.Value...)
	}

	return usageDetails, nil
}
