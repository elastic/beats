// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package billing

import (
	"fmt"
	"time"

	"github.com/elastic/beats/v8/x-pack/metricbeat/module/azure"

	"github.com/pkg/errors"

	prevConsumption "github.com/Azure/azure-sdk-for-go/services/consumption/mgmt/2019-01-01/consumption"
	"github.com/Azure/azure-sdk-for-go/services/consumption/mgmt/2019-10-01/consumption"

	"github.com/elastic/beats/v8/libbeat/logp"
)

// Client represents the azure client which will make use of the azure sdk go metrics related clients
type Client struct {
	BillingService Service
	Config         azure.Config
	Log            *logp.Logger
}

type Usage struct {
	UsageDetails  []prevConsumption.UsageDetail
	ActualCosts   []consumption.Forecast
	ForecastCosts []consumption.Forecast
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
func (client *Client) GetMetrics() (Usage, error) {

	var usage Usage
	scope := fmt.Sprintf("subscriptions/%s", client.Config.SubscriptionId)
	if client.Config.BillingScopeDepartment != "" {
		scope = fmt.Sprintf("/providers/Microsoft.Billing/departments/%s", client.Config.BillingScopeDepartment)
	} else if client.Config.BillingScopeAccountId != "" {
		scope = fmt.Sprintf("/providers/Microsoft.Billing/billingAccounts/%s", client.Config.BillingScopeAccountId)
	}
	startTime := time.Now().UTC().Truncate(24 * time.Hour).Add((-24) * time.Hour)
	endTime := startTime.Add(time.Hour * 24).Add(time.Second * (-1))
	usageDetails, err := client.BillingService.GetUsageDetails(scope, "properties/meterDetails",
		fmt.Sprintf("properties/usageStart eq '%s' and properties/usageEnd eq '%s'", startTime.Format(time.RFC3339Nano), endTime.Format(time.RFC3339Nano)),
		"", nil, "properties/instanceLocation")
	if err != nil {
		return usage, errors.Wrap(err, "Retrieving usage details failed in client")
	}
	usage.UsageDetails = usageDetails.Values()
	actualCosts, err := client.BillingService.GetForcast(fmt.Sprintf("properties/chargeType eq '%s'", "Actual"))
	if err != nil {
		return usage, errors.Wrap(err, "Retrieving forecast - actual costs failed in client")
	}
	usage.ActualCosts = *actualCosts.Value
	forecastCosts, err := client.BillingService.GetForcast(fmt.Sprintf("properties/chargeType eq '%s'", "Forecast"))
	if err != nil {
		return usage, errors.Wrap(err, "Retrieving forecast failed in client")
	}
	usage.ForecastCosts = *forecastCosts.Value
	return usage, nil
}
