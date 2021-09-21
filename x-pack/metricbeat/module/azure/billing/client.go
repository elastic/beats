// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package billing

import (
	"fmt"
	"time"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/azure"

	"github.com/pkg/errors"

	"github.com/Azure/azure-sdk-for-go/services/consumption/mgmt/2019-10-01/consumption"

	"github.com/elastic/beats/v7/libbeat/logp"
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
func (client *Client) GetMetrics(startTime time.Time, endTime time.Time) (Usage, error) {
	var usage Usage
	subScope := fmt.Sprintf("subscriptions/%s", client.Config.SubscriptionId)
	filter := fmt.Sprintf("properties/usageStart eq '%s' and properties/usageEnd eq '%s'", startTime.Format(time.RFC3339Nano), endTime.Format(time.RFC3339Nano))
	marketplaceDetails, err := client.BillingService.GetMarketplaceUsage(subScope, filter, "", nil)
	if err != nil {
		//return usage, errors.Wrap(err, "Retrieving marketplace usage details failed in client")
	}
	_ = marketplaceDetails

	charges, err := client.BillingService.GetCharges(subScope, startTime.Format(time.RFC3339Nano), endTime.Format(time.RFC3339Nano), "", "")
	if err != nil {
		//return usage, errors.Wrap(err, "Retrieving marketplace usage details failed in client")
	}
	_ = charges
	usageDetails, err := client.BillingService.GetUsageDetails(subScope, "properties/meterDetails", filter, "", nil, consumption.MetrictypeActualCostMetricType)
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
