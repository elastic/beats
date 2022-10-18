// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package billing

import (
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/consumption/mgmt/2019-10-01/consumption"

	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/azure"
)

// Client represents the azure client which will make use of the azure sdk go metrics related clients
type Client struct {
	BillingService Service
	Config         azure.Config
	Log            *logp.Logger
}

type Usage struct {
	UsageDetails  []consumption.BasicUsageDetail
	ActualCosts   []consumption.Forecast
	ForecastCosts []consumption.Forecast
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
func (client *Client) GetMetrics(startTime time.Time, endTime time.Time) (Usage, error) {
	var usage Usage
	scope := fmt.Sprintf("subscriptions/%s", client.Config.SubscriptionId)
	if client.Config.BillingScopeDepartment != "" {
		scope = fmt.Sprintf("/providers/Microsoft.Billing/departments/%s", client.Config.BillingScopeDepartment)
	} else if client.Config.BillingScopeAccountId != "" {
		scope = fmt.Sprintf("/providers/Microsoft.Billing/billingAccounts/%s", client.Config.BillingScopeAccountId)
	}
	client.Log.
		With("billing.scope", scope).
		With("billing.start_time", startTime).
		With("billing.end_time", endTime).
		Infow("Getting usage details for scope")

	usageDetails, err := client.BillingService.GetUsageDetails(
		scope,
		"properties/meterDetails",
		fmt.Sprintf(
			"properties/usageStart eq '%s' and properties/usageEnd eq '%s'",
			startTime.Format(time.RFC3339Nano),
			endTime.Format(time.RFC3339Nano),
		),
		"", // skipToken
		nil,
		consumption.MetrictypeActualCostMetricType,
		startTime.Format("2006-01-02"), // startDate
		endTime.Format("2006-01-02"),   // endDate
	)
	if err != nil {
		return usage, fmt.Errorf("retrieving usage details failed in client: %w", err)
	}

	usage.UsageDetails = usageDetails.Values()

	//
	// Forecast
	//

	actualCosts, err := client.BillingService.GetForecast(fmt.Sprintf("properties/chargeType eq '%s'", "Actual"))
	if err != nil {
		return usage, fmt.Errorf("retrieving forecast - actual costs failed in client: %w", err)
	}
	usage.ActualCosts = actualCosts

	forecastCosts, err := client.BillingService.GetForecast(fmt.Sprintf("properties/chargeType eq '%s'", "Forecast"))
	if err != nil {
		return usage, fmt.Errorf("retrieving forecast - forecast costs failed in client: %w", err)
	}
	usage.ForecastCosts = forecastCosts

	return usage, nil
}
