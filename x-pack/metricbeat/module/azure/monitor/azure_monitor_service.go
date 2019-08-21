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
	"strings"
)

type AzureMonitorService struct {
	metricsClient          *insights.MetricsClient
	metricDefinitionClient *insights.MetricDefinitionsClient
	resourceClient         *resources.Client
	log                    *logp.Logger
}

// NewAzureService instantiates the an Azure monitoring service
func NewAzureService(clientID string, clientSecret string, tenantID string, subscriptionID string) (*AzureMonitorService, error) {
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
		log:                    logp.NewLogger("azure monitor service"),
	}
	return client, nil
}

// GetResourceDefinitions will retrieve the azure resources based on the options entered
func (client AzureMonitorService) GetResourceDefinitions(ID string, group string, rType string, query string) ([]resources.GenericResource, error) {
	var resourceQuery string
	if ID != "" {
		resourceQuery = fmt.Sprintf("resourceID eq '%s'", ID)
	}
	if group != "" {
		resourceQuery = fmt.Sprintf("resourceGroup eq '%s'", group)
		if rType != "" {
			resourceQuery += fmt.Sprintf(" AND resourceType eq '%s'", rType)
		}
	}
	if query != "" {
		resourceQuery = query
	}
	result, err := client.resourceClient.List(context.Background(), resourceQuery, "true", nil)
	if err != nil {
		client.log.Errorf("error while listing resources by resource query  %s : %s", resourceQuery, err)
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
func (client *AzureMonitorService) GetMetricValues(resourceID string, namespace string, timegrain string, timespan string, metricNames []string, aggregations string, filter string) ([]insights.Metric, error) {
	var tg *string
	if timegrain != "" {
		tg = &timegrain
	}
	// check for limit of requested metrics (20)
	var metrics []insights.Metric
	for i := 0; i < len(metricNames); i += 20 {
		end := i + 20
		if end > len(metricNames) {
			end = len(metricNames)
		}
		resp, err := client.metricsClient.List(context.Background(), resourceID, timespan, tg, strings.Join(metricNames[i:end], ","),
			aggregations, nil, "", filter, insights.Data, namespace)
		if err != nil {
			client.log.Errorf("error while listing metric values by resource ID %s and namespace  %s : %s", resourceID, namespace, err)
		} else {
			metrics = append(metrics, *resp.Value...)
		}

	}
	return metrics, nil
}
