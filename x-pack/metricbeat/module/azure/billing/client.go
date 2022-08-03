// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package billing

import (
	"context"
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/consumption/mgmt/2019-10-01/consumption"
	"github.com/Azure/azure-sdk-for-go/services/costmanagement/mgmt/2019-11-01/costmanagement"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/azure"
	"github.com/elastic/elastic-agent-libs/logp"
)

// Client represents the azure client which will make use of the azure sdk go metrics related clients
type Client struct {
	BillingService Service
	Config         azure.Config
	Log            *logp.Logger
}

type Usage struct {
	UsageDetails []consumption.BasicUsageDetail
	Forecasts    costmanagement.QueryResult
	// ForecastCosts []consumption.Forecast
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
func (client *Client) GetMetrics(opts TimeIntervalOptions) (Usage, error) {
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
		With("billing.usage.start_time", opts.usageStart).
		With("billing.usage.end_time", opts.usageEnd).
		Infow("Getting usage details for scope")

	page, err := client.BillingService.GetUsageDetails(
		scope,
		"properties/meterDetails",
		fmt.Sprintf(
			"properties/usageStart eq '%s' and properties/usageEnd eq '%s'",
			opts.usageStart.Format(time.RFC3339Nano),
			opts.usageEnd.Format(time.RFC3339Nano),
		),
		"", // skipToken
		//&pageSize,
		nil,
		consumption.MetrictypeActualCostMetricType,
		opts.usageStart.Format("2006-01-02"), // startDate
		opts.usageEnd.Format("2006-01-02"),   // endDate
	)
	if err != nil {
		return usage, fmt.Errorf("retrieving usage details failed in client: %w", err)
	}

	for page.NotDone() {
		usage.UsageDetails = append(usage.UsageDetails, page.Values()...)
		if err := page.NextWithContext(context.Background()); err != nil {
			return usage, fmt.Errorf("retrieving usage details failed in client: %w", err)
		}
	}

	//
	// Fetch the Forecast
	//

	client.Log.
		With("billing.scope", scope).
		With("billing.forecast.start_time", opts.forecastStart).
		With("billing.forecast.end_time", opts.forecastEnd).
		Infow("Getting forecast for scope")

	queryResult, err := client.BillingService.GetForecast(scope, opts.forecastStart, opts.forecastEnd)
	if err != nil {
		return usage, fmt.Errorf("retrieving forecast - forecast costs failed in client: %w", err)
	}

	usage.Forecasts = queryResult

	return usage, nil
}
