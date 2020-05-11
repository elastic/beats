// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package billing

import (
	"fmt"
	"github.com/Azure/azure-sdk-for-go/services/consumption/mgmt/2019-01-01/consumption"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"time"
)

// Client represents the azure client which will make use of the azure sdk go metrics related clients
type Client struct {
	BillingService Service
	Config         Config
	Log            *logp.Logger
}

// NewClient instantiates the an Azure monitoring client
func NewClient(config Config) (*Client, error) {
	billingService, err := NewService(config.ClientId, config.ClientSecret, config.TenantId, config.SubscriptionId)
	if err != nil {
		return nil, err
	}
	client := &Client{
		BillingService: *billingService,
		Config:         config,
		Log:            logp.NewLogger("azure monitor client"),
	}
	return client, nil
}

// GetMetricValues returns the specified metric data points for the specified resource ID/namespace.
func (client *Client) Forcast(report mb.ReporterV2) (consumption.ForecastsListResult, error) {
	var top int32 = 10
	endTime := time.Now().UTC()
	startTime := endTime.Add(client.Config.Period * (-2))
	actualCosts3, _ := client.BillingService.GetUsageDetails( fmt.Sprintf("subscriptions/%s", client.Config.SubscriptionId), "meterDetails",
		fmt.Sprintf("properties/usageStart eq '%s' and properties/usageEnd eq '%s'", startTime.Format(time.RFC3339), endTime.Format(time.RFC3339)),
		"", &top, "properties/usageStart")
	_= actualCosts3

	//dsds, _ := client.BillingService.GetCharges("/providers/Microsoft.Billing/billingAccounts/56437391", "")
	//_ = dsds

	//usageEnd=2020-06-01T23:59:59.0000000Z
	return client.BillingService.GetForcast("")
}
