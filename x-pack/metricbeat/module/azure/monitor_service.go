// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azure

import (
	"context"
	"fmt"
	"strings"

	"github.com/elastic/beats/libbeat/logp"

	"github.com/Azure/azure-sdk-for-go/services/preview/monitor/mgmt/2019-06-01/insights"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-03-01/resources"
	"github.com/Azure/go-autorest/autorest/azure/auth"
)

// MonitorService service wrapper to the azure sdk for go
type MonitorService struct {
	metricsClient          *insights.MetricsClient
	metricDefinitionClient *insights.MetricDefinitionsClient
	metricNamespaceClient  *insights.MetricNamespacesClient
	resourceClient         *resources.Client
	context                context.Context
	log                    *logp.Logger
}

const metricNameLimit = 20

// NewService instantiates the Azure monitoring service
func NewService(clientID string, clientSecret string, tenantID string, subscriptionID string) (*MonitorService, error) {
	clientConfig := auth.NewClientCredentialsConfig(clientID, clientSecret, tenantID)
	authorizer, err := clientConfig.Authorizer()
	if err != nil {
		return nil, err
	}
	metricsClient := insights.NewMetricsClient(subscriptionID)
	metricsDefinitionClient := insights.NewMetricDefinitionsClient(subscriptionID)
	resourceClient := resources.NewClient(subscriptionID)
	metricNamespaceClient := insights.NewMetricNamespacesClient(subscriptionID)
	metricsClient.Authorizer = authorizer
	metricsDefinitionClient.Authorizer = authorizer
	resourceClient.Authorizer = authorizer
	metricNamespaceClient.Authorizer = authorizer
	service := &MonitorService{
		metricDefinitionClient: &metricsDefinitionClient,
		metricsClient:          &metricsClient,
		metricNamespaceClient:  &metricNamespaceClient,
		resourceClient:         &resourceClient,
		context:                context.Background(),
		log:                    logp.NewLogger("azure monitor service"),
	}
	return service, nil
}

// GetResourceDefinitions will retrieve the azure resources based on the options entered
func (service MonitorService) GetResourceDefinitions(ID []string, group []string, rType string, query string) (resources.ListResultPage, error) {
	var resourceQuery string
	if len(ID) > 0 {
		var filterList []string
		// listing resourceID conditions does not seem to work with the API but querying by name or resource types will work
		for _, id := range ID {
			filterList = append(filterList, fmt.Sprintf("name eq '%s'", getResourceNameFromID(id)))
		}
		resourceQuery = fmt.Sprintf("(%s) AND resourceType eq '%s'", strings.Join(filterList, " OR "), getResourceTypeFromID(ID[0]))
	} else if len(group) > 0 {
		var filterList []string
		for _, gr := range group {
			filterList = append(filterList, fmt.Sprintf("resourceGroup eq '%s'", gr))
		}
		resourceQuery = strings.Join(filterList, " OR ")
		if rType != "" {
			resourceQuery = fmt.Sprintf("(%s) AND resourceType eq '%s'", resourceQuery, rType)
		}
	} else if query != "" {
		resourceQuery = query
	}
	return service.resourceClient.List(service.context, resourceQuery, "resourceID", nil)
}

// GetMetricNamespaces will return all supported namespaces based on the resource id and namespace
func (service *MonitorService) GetMetricNamespaces(resourceID string) (insights.MetricNamespaceCollection, error) {
	return service.metricNamespaceClient.List(service.context, resourceID, "")
}

// GetMetricDefinitions will return all supported metrics based on the resource id and namespace
func (service *MonitorService) GetMetricDefinitions(resourceID string, namespace string) (insights.MetricDefinitionCollection, error) {
	return service.metricDefinitionClient.List(service.context, resourceID, namespace)
}

// GetMetricValues will return the metric values based on the resource and metric details
func (service *MonitorService) GetMetricValues(resourceID string, namespace string, timegrain string, timespan string, metricNames []string, aggregations string, filter string) ([]insights.Metric, string, error) {
	var tg *string
	var interval string
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

		// check for applied charges before returning any errors
		if resp.Cost != nil && *resp.Cost != 0 {
			service.log.Warnf("Charges amounted to %v are being applied while retrieving the metric values from the resource %s ", *resp.Cost, resourceID)
		}
		if err != nil {
			return metrics, "", err
		}
		interval = *resp.Interval
		metrics = append(metrics, *resp.Value...)

	}
	return metrics, interval, nil
}
