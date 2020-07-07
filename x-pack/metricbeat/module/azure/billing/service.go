// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package billing

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/consumption/mgmt/2019-01-01/consumption"
	"github.com/Azure/go-autorest/autorest/azure/auth"

	"github.com/elastic/beats/v7/libbeat/logp"
)

// BillingService service wrapper to the azure sdk for go
type Service struct {
	forcastsClient *consumption.ForecastsClient
	usageClient    *consumption.UsageDetailsClient
	context        context.Context
	log            *logp.Logger
	//aggregatedClient  *consumption.AggregatedCostClient
	//chargesClient     *consumption.ChargesClient
	//balanceClient     *consumption.BalancesClient
	//invoiceClient     *billing.InvoicesClient
	//marketplaceClient *consumption.MarketplacesClient
	//usagClient        *commerce.UsageAggregatesClient
}

// NewService instantiates the Azure monitoring service
func NewService(clientId string, clientSecret string, tenantId string, subscriptionId string) (*Service, error) {
	clientConfig := auth.NewClientCredentialsConfig(clientId, clientSecret, tenantId)
	authorizer, err := clientConfig.Authorizer()
	if err != nil {
		return nil, err
	}
	//aggregatedCostClient := consumption.NewAggregatedCostClient(subscriptionId)
	//chargesClient := consumption.NewChargesClient(subscriptionId)
	//balanceClient := consumption.NewBalancesClient(subscriptionId)
	forcastsClient := consumption.NewForecastsClient(subscriptionId)
	usageDetailsClient := consumption.NewUsageDetailsClient(subscriptionId)
	//invoiceClient := billing.NewInvoicesClient(subscriptionId)
	//marketplaceClient := consumption.NewMarketplacesClient(subscriptionId)
	//usageClient := commerce.NewUsageAggregatesClient(subscriptionId)
	forcastsClient.Authorizer = authorizer
	usageDetailsClient.Authorizer = authorizer
	//aggregatedCostClient.Authorizer = authorizer
	//chargesClient.Authorizer = authorizer
	//balanceClient.Authorizer = authorizer
	//invoiceClient.Authorizer = authorizer
	//marketplaceClient.Authorizer = authorizer
	//usageClient.Authorizer = authorizer
	service := &Service{
		forcastsClient: &forcastsClient,
		usageClient:    &usageDetailsClient,
		context:        context.Background(),
		log:            logp.NewLogger("azure billing service"),
		//aggregatedClient:  &aggregatedCostClient,
		//chargesClient:     &chargesClient,
		//balanceClient:     &balanceClient,
		//invoiceClient:     &invoiceClient,
		//marketplaceClient: &marketplaceClient,
		//usagClient:        &usageClient,
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
//func (service *Service) GetAggregatedCosts(managementGroupID string, filter string) (consumption.ManagementGroupAggregatedCostResult, error) {
//	return service.aggregatedClient.GetByManagementGroup(service.context, managementGroupID, filter)
//}

// GetCharges
//func (service *Service) GetCharges(scope string, filter string) (consumption.ChargeSummary, error) {
//	return service.chargesClient.ListByScope(service.context, scope, filter)
//}

// GetCharges
//func (service *Service) GetInvoices(accountId string, startDate string, endDate string) (billing.InvoiceListResultPage, error) {
//	return service.invoiceClient.ListByBillingAccountName(service.context, accountId, startDate, endDate)
//}

// GetCharges
//func (service *Service) GetMarketplace(scope string, filter string, top *int32, skiptoken string) (consumption.MarketplacesListResultPage, error) {
//	return service.marketplaceClient.List(service.context, scope, filter, top, skiptoken)
//}

// GetCharges
//func (service *Service) GetUsage(reportedStartTime date.Time, reportedEndTime date.Time, showDetails *bool, aggregationGranularity commerce.AggregationGranularity, continuationToken string) (commerce.UsageAggregationListResultPage, error) {
//	return service.usagClient.List(service.context, reportedStartTime, reportedEndTime, showDetails, aggregationGranularity, continuationToken)
//}
