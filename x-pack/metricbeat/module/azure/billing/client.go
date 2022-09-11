// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package billing

import (
	"context"
	"fmt"
	"time"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/azure"

	"github.com/pkg/errors"

	"github.com/Azure/azure-sdk-for-go/services/consumption/mgmt/2019-10-01/consumption"
	"github.com/Azure/azure-sdk-for-go/services/costmanagement/mgmt/2019-11-01/costmanagement"

	"github.com/elastic/beats/v7/libbeat/logp"
)

// Client represents the azure client which will make use of the azure sdk go metrics related clients
type Client struct {
	BillingService Service
	Config         azure.Config
	Log            *logp.Logger
}

// Usage contains the usage details and forecast values.
type Usage struct {
	UsageDetails []consumption.BasicUsageDetail
	Forecasts    costmanagement.QueryResult
}

// NewClient instantiates the an Azure monitoring client
func NewClient(config azure.Config) (*Client, error) {
	usageService, err := NewService(config)
	if err != nil {
		return nil, err
	}
	client := &Client{
		BillingService: usageService,
		Config:         config,
		Log:            logp.NewLogger("azure monitor client"),
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

	paginator, err := client.BillingService.GetUsageDetails(
		scope,
		"properties/meterDetails",
		filter,
		"",  // skipToken, used for paging, not required on the first call.
		nil, // result page size, defaults to ?
		consumption.MetrictypeActualCostMetricType,
		timeOpts.usageStart.Format("2006-01-02"), // startDate
		timeOpts.usageEnd.Format("2006-01-02"),   // endDate
	)
	if err != nil {
		return usage, errors.Wrap(err, "Retrieving usage details failed in client")
	}

	for paginator.NotDone() {
		usage.UsageDetails = append(usage.UsageDetails, paginator.Values()...)
		if err := paginator.NextWithContext(context.Background()); err != nil {
			return usage, fmt.Errorf("retrieving usage details failed in client: %w", err)
		}
	}

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
		return usage, errors.Wrap(err, "Retrieving forecast failed in client")
	}

	usage.Forecasts = queryResult

	return usage, nil
}
