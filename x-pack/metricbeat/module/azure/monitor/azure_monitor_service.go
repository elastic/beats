// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package monitor

import (
	"context"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/preview/monitor/mgmt/2019-06-01/insights"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-03-01/resources"
	"github.com/Azure/go-autorest/autorest/azure/auth"
)

type AzureMonitorService struct {
	metricsClient          *insights.MetricsClient
	metricDefinitionClient *insights.MetricDefinitionsClient
	resourceClient         *resources.Client
	context                context.Context
}

const metricNameLimit = 20

// NewAzureService instantiates the Azure monitoring service
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
	service := &AzureMonitorService{
		metricDefinitionClient: &metricsDefinitionClient,
		metricsClient:          &metricsClient,
		resourceClient:         &resourceClient,
		context:                context.Background(),
	}
	return service, nil
}

// GetResourceDefinitions will retrieve the azure resources based on the options entered
func (service AzureMonitorService) GetResourceDefinitions(ID string, group string, rType string, query string) (resources.ListResultPage, error) {
	var resourceQuery string
	if ID != "" {
		resourceQuery = fmt.Sprintf("resourceID eq '%s'", ID)
	} else if group != "" {
		resourceQuery = fmt.Sprintf("resourceGroup eq '%s'", group)
		if rType != "" {
			resourceQuery += fmt.Sprintf(" AND resourceType eq '%s'", rType)
		}
	} else if query != "" {
		resourceQuery = query
	}
	return service.resourceClient.List(service.context, resourceQuery, "true", nil)
}

// GetMetricDefinitions will return all supported metrics based on the resource id and namespace
func (service *AzureMonitorService) GetMetricDefinitions(resourceID string, namespace string) (insights.MetricDefinitionCollection, error) {
	return service.metricDefinitionClient.List(service.context, resourceID, namespace)
}

// GetMetricValues will return the metric values based on the resource and metric details
func (service *AzureMonitorService) GetMetricValues(resourceID string, namespace string, timegrain string, timespan string, metricNames []string, aggregations string, filter string) ([]insights.Metric, error) {
	var tg *string
	if timegrain != "" {
		tg = &timegrain
	}
	// check for limit of requested metrics (20)
	var metrics []insights.Metric
	for i := 0; i < len(metricNames); i += metricNameLimit {
		end := i + metricNameLimit
		if end > len(metricNames) {
			end = len(metricNames)
		}
		resp, err := service.metricsClient.List(service.context, resourceID, timespan, tg, strings.Join(metricNames[i:end], ","),
			aggregations, nil, "", filter, insights.Data, namespace)
		if err != nil {
			return metrics, err
		}
		metrics = append(metrics, *resp.Value...)

	}
	return metrics, nil
}
