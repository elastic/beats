// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package billing

import (
	"context"
	"github.com/Azure/azure-sdk-for-go/services/consumption/mgmt/2019-01-01/consumption"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-03-01/resources"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/elastic/beats/v7/libbeat/logp"
)

// BillingService service wrapper to the azure sdk for go
type Service struct {
	forcastsClient *consumption.ForecastsClient
	usageClient    *consumption.UsageDetailsClient
	resourceClient *resources.Client
	context        context.Context
	log            *logp.Logger
	aggregatedClient *consumption.AggregatedCostClient
	chargesClient *consumption.ChargesClient
	balanceClient *consumption.BalancesClient
}

// NewService instantiates the Azure monitoring service
func NewService(clientId string, clientSecret string, tenantId string, subscriptionId string) (*Service, error) {
	clientConfig := auth.NewClientCredentialsConfig(clientId, clientSecret, tenantId)
	authorizer, err := clientConfig.Authorizer()
	if err != nil {
		return nil, err
	}
	aggregatedCostClient := consumption.NewAggregatedCostClient(subscriptionId)
	chargesClient:= consumption.NewChargesClient(subscriptionId)
	balanceClient:= consumption.NewBalancesClient(subscriptionId)
	forcastsClient := consumption.NewForecastsClient(subscriptionId)
	usageDetailsClient := consumption.NewUsageDetailsClient(subscriptionId)
	forcastsClient.Authorizer = authorizer
	usageDetailsClient.Authorizer = authorizer
	aggregatedCostClient.Authorizer= authorizer
	chargesClient.Authorizer= authorizer
	balanceClient.Authorizer= authorizer
	service := &Service{
		forcastsClient: &forcastsClient,
		usageClient:    &usageDetailsClient,
		context:        context.Background(),
		log:            logp.NewLogger("azure billing service"),
		aggregatedClient: &aggregatedCostClient,
		chargesClient: &chargesClient,
		balanceClient: &balanceClient,
	}
	return service, nil
}

// GetForcast
func (service *Service) GetForcast(filter string) (consumption.ForecastsListResult, error) {
	return service.forcastsClient.List(service.context, filter)
}

// GetUsageDetails
func (service *Service) GetUsageDetails(scope string, expand string, filter string, skiptoken string, top *int32, apply string) (consumption.UsageDetailsListResultPage, error) {
	return service.usageClient.List(service.context, scope, expand, filter, skiptoken, top, apply)

}

// GetAggregatedCosts
func (service *Service) GetAggregatedCosts(managementGroupID string, filter string) (consumption.ManagementGroupAggregatedCostResult, error) {
	return service.aggregatedClient.GetByManagementGroup(service.context, managementGroupID, filter)
}
// GetCharges
func (service *Service) GetCharges(scope string, filter string) (consumption.ChargeSummary, error) {
	return service.chargesClient.ListByScope(service.context, scope, filter)
}
