// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package billing

import (
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/consumption/armconsumption"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/costmanagement/armcostmanagement"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/azure"
	"github.com/elastic/elastic-agent-libs/logp"
)

// Client represents the azure client which will make use of the azure sdk go metrics related clients
type Client struct {
	BillingService Service
	Config         azure.Config
	Log            *logp.Logger
}

// Usage contains the usage details and forecast values.
type Usage struct {
	UsageDetails []armconsumption.UsageDetailClassification
	Forecasts    armcostmanagement.QueryResult
}

// NewClient builds a new client for the azure billing service
func NewClient(config azure.Config) (*Client, error) {
	usageService, err := NewService(config)
	if err != nil {
		return nil, err
	}
	client := &Client{
		BillingService: usageService,
		Config:         config,
		Log:            logp.NewLogger("azure billing client"),
	}
	return client, nil
}

// GetMetrics returns the usage detail and forecast values.
func (client *Client) GetMetrics(timeOpts TimeIntervalOptions) (Usage, error) {
	var usage Usage

	//
	// Establish the requested scope
	//

	scope := fmt.Sprintf("subscriptions/%s", client.Config.SubscriptionId)
	if client.Config.BillingScopeDepartment != "" {
		scope = fmt.Sprintf("/providers/Microsoft.Billing/departments/%s", client.Config.BillingScopeDepartment)
	} else if client.Config.BillingScopeAccountId != "" {
		scope = fmt.Sprintf("/providers/Microsoft.Billing/billingAccounts/%s", client.Config.BillingScopeAccountId)
	}

	//
	// Fetch the usage details
	//

	client.Log.
		With("billing.scope", scope).
		With("billing.usage.start_time", timeOpts.usageStart).
		With("billing.usage.end_time", timeOpts.usageEnd).
		Infow("Getting usage details for scope")

	filter := fmt.Sprintf(
		"properties/usageStart eq '%s' and properties/usageEnd eq '%s'",
		timeOpts.usageStart.Format(time.RFC3339Nano),
		timeOpts.usageEnd.Format(time.RFC3339Nano),
	)

	result, err := client.BillingService.GetUsageDetails(
		scope,
		"properties/meterDetails",
		filter,
		armconsumption.MetrictypeActualCostMetricType,
		timeOpts.usageStart.Format("2006-01-02"), // startDate
		timeOpts.usageEnd.Format("2006-01-02"),   // endDate
	)
	if err != nil {
		return usage, fmt.Errorf("retrieving usage details failed in client: %w", err)
	}

	usage.UsageDetails = append(usage.UsageDetails, result.Value...)

	//
	// Fetch the Forecast
	//

	client.Log.
		With("billing.scope", scope).
		With("billing.forecast.start_time", timeOpts.forecastStart).
		With("billing.forecast.end_time", timeOpts.forecastEnd).
		Infow("Getting forecast for scope")

	queryResult, err := client.BillingService.GetForecast(scope, timeOpts.forecastStart, timeOpts.forecastEnd)
	if err != nil {
		return usage, fmt.Errorf("retrieving forecast - forecast costs failed in client: %w", err)
	}

	usage.Forecasts = queryResult

	return usage, nil
}
