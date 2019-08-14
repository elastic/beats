// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package monitor

import (
	"context"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/services/preview/monitor/mgmt/2019-06-01/insights"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-03-01/resources"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/elastic/beats/libbeat/logp"
)

type AzureMonitorService struct {
	metricsClient          *insights.MetricsClient
	metricDefinitionClient *insights.MetricDefinitionsClient
	resourceClient         *resources.Client
	log                    *logp.Logger
}

// Init instantiates the an Azure monitoring service
func Init(clientID string, clientSecret string, tenantID string, subscriptionID string) (*AzureMonitorService, error) {
	clientConfig := auth.NewClientCredentialsConfig(clientID, clientSecret, tenantID)
	authorizer, err := clientConfig.Authorizer()
	if err != nil {
		return nil, err
	}
	metricsClient := insights.NewMetricsClient(subscriptionID)
	metricsDefinitionClient := insights.NewMetricDefinitionsClient(subscriptionID)
	resourceClient := resources.NewClient(subscriptionID)
	metricsClient.Authorizer = authorizer
	metricsDefinitionClient.Authorizer = authorizer
	resourceClient.Authorizer = authorizer
	client := &AzureMonitorService{
		metricDefinitionClient: &metricsDefinitionClient,
		metricsClient:          &metricsClient,
		resourceClient:         &resourceClient,
		log:                    logp.NewLogger("azure monitor"),
	}
	return client, nil
}

// GetResourceById will retrieve the azure resource details based on the id entered
func (client AzureMonitorService) GetResourceById(resourceID string) (resources.GenericResource, error) {
	resource, err := client.resourceClient.GetByID(context.Background(), resourceID)
	if err != nil {
		client.log.Errorf(" error while retrieving resource by id  %s : %v", resourceID, err)
	}
	return resource, err
}

// GetResourcesByResourceGroup will retrieve the list of resources that match the resource group and type
func (client *AzureMonitorService) GetResourcesByResourceGroup(resourceGroup string, resourceType string) ([]resources.GenericResource, error) {
	var top int32 = 500
	result, err := client.resourceClient.ListByResourceGroup(context.Background(), resourceGroup, fmt.Sprintf("resourceType eq '%s'", resourceType), "true", &top)
	if err != nil {
		client.log.Errorf("error while listing resources by resource group %s  and filter %s : %s", resourceGroup, resourceType, err)
	}
	return result.Values(), err
}

// GetResourcesByResourceQuery will retrieve the list of resources that match the filter entered by the user
func (client *AzureMonitorService) GetResourcesByResourceQuery(resourceQury string) ([]resources.GenericResource, error) {
	var top int32 = 500
	result, err := client.resourceClient.List(context.Background(), resourceQury, "true", &top)
	if err != nil {
		client.log.Errorf("error while listing resources by resource query  %s : %s", resourceQury, err)
	}
	return result.Values(), err
}

// GetMetricDefinitions will return all supported metrics based on the resource id and namespace
func (client *AzureMonitorService) GetMetricDefinitions(resourceID string, namespace string) ([]insights.MetricDefinition, error) {
	result, err := client.metricDefinitionClient.List(context.Background(), resourceID, namespace)
	if err != nil {
		client.log.Errorf("error while listing metric definitions by resource ID %s and namespace  %s : %s", resourceID, namespace, err)
	}
	return *result.Value, err
}

// GetMetricValues will return the metric values based on the resource and metric details
func (client *AzureMonitorService) GetMetricValues(resourceID string, namespace string, timespan string, metricNames string, aggregations string, filter string) ([]insights.Metric, error) {
	resp, err := client.metricsClient.List(context.Background(), resourceID, timespan, nil, metricNames,
		aggregations, nil, "", filter, insights.Data, namespace)
	if err != nil {
		return nil, err
	}
	return *resp.Value, nil
}
