// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package billing

import (
	"context"

	"github.com/elastic/beats/v8/x-pack/metricbeat/module/azure"

	"github.com/Azure/azure-sdk-for-go/services/consumption/mgmt/2019-10-01/consumption"
	"github.com/Azure/go-autorest/autorest/azure/auth"

	prevConsumption "github.com/Azure/azure-sdk-for-go/services/consumption/mgmt/2019-01-01/consumption"

	"github.com/elastic/beats/v8/libbeat/logp"
)

// Service interface for the azure monitor service and mock for testing
type Service interface {
	GetForcast(filter string) (consumption.ForecastsListResult, error)
	GetUsageDetails(scope string, expand string, filter string, skiptoken string, top *int32, apply string) (prevConsumption.UsageDetailsListResultPage, error)
}

// BillingService service wrapper to the azure sdk for go
type UsageService struct {
	usageDetailsClient *prevConsumption.UsageDetailsClient
	forcastsClient     *consumption.ForecastsClient
	context            context.Context
	log                *logp.Logger
}

// NewService instantiates the Azure monitoring service
func NewService(config azure.Config) (*UsageService, error) {
	clientConfig := auth.NewClientCredentialsConfig(config.ClientId, config.ClientSecret, config.TenantId)
	clientConfig.AADEndpoint = config.ActiveDirectoryEndpoint
	clientConfig.Resource = config.ResourceManagerEndpoint
	authorizer, err := clientConfig.Authorizer()
	if err != nil {
		return nil, err
	}
	forcastsClient := consumption.NewForecastsClientWithBaseURI(config.ResourceManagerEndpoint, config.SubscriptionId)
	usageDetailsClient := prevConsumption.NewUsageDetailsClientWithBaseURI(config.ResourceManagerEndpoint, config.SubscriptionId)

	forcastsClient.Authorizer = authorizer
	usageDetailsClient.Authorizer = authorizer
	service := &UsageService{
		usageDetailsClient: &usageDetailsClient,
		forcastsClient:     &forcastsClient,
		context:            context.Background(),
		log:                logp.NewLogger("azure billing service"),
	}
	return service, nil
}

// GetForcast
func (service *UsageService) GetForcast(filter string) (consumption.ForecastsListResult, error) {
	return service.forcastsClient.List(service.context, filter)
}

// GetUsageDetails
func (service *UsageService) GetUsageDetails(scope string, expand string, filter string, skiptoken string, top *int32, apply string) (prevConsumption.UsageDetailsListResultPage, error) {
	return service.usageDetailsClient.List(service.context, scope, expand, filter, skiptoken, top, apply)
}
