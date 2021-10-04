// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azure

import (
	"context"
	"fmt"
	"strings"

	"github.com/elastic/beats/v7/libbeat/logp"

	"github.com/Azure/azure-sdk-for-go/services/preview/monitor/mgmt/2019-06-01/insights"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-10-01/resources"
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

const (
	metricNameLimit = 20
	ApiVersion      = "2019-12-01"
)

// NewService instantiates the Azure monitoring service
func NewService(config Config) (*MonitorService, error) {
	clientConfig := auth.NewClientCredentialsConfig(config.ClientId, config.ClientSecret, config.TenantId)
	clientConfig.AADEndpoint = config.ActiveDirectoryEndpoint
	clientConfig.Resource = config.ResourceManagerEndpoint
	authorizer, err := clientConfig.Authorizer()
	if err != nil {
		return nil, err
	}
	metricsClient := insights.NewMetricsClientWithBaseURI(config.ResourceManagerEndpoint, config.SubscriptionId)
	metricsDefinitionClient := insights.NewMetricDefinitionsClientWithBaseURI(config.ResourceManagerEndpoint, config.SubscriptionId)
	resourceClient := resources.NewClientWithBaseURI(config.ResourceManagerEndpoint, config.SubscriptionId)
	metricNamespaceClient := insights.NewMetricNamespacesClientWithBaseURI(config.ResourceManagerEndpoint, config.SubscriptionId)
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
func (service MonitorService) GetResourceDefinitions(id []string, group []string, rType string, query string) ([]resources.GenericResourceExpanded, error) {
	var resourceQuery string
	var resourceList []resources.GenericResourceExpanded
	if len(id) > 0 {
		// listing multiple resourceId conditions does not seem to work with the API, extracting the name and resource type does not work as the position of the `resourceType` can move if a parent resource is involved, filtering by resource name and resource group (if extracted) is also not possible as
		// different types of resources can contain the same name.
		for _, id := range id {
			resource, err := service.resourceClient.List(service.context, fmt.Sprintf("resourceId eq '%s'", id), "", nil)
			if err != nil {
				return nil, err
			}
			if len(resource.Values()) > 0 {
				resourceList = append(resourceList, resource.Values()...)
			}
		}
		return resourceList, nil
	}
	if len(group) > 0 {
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
	result, err := service.resourceClient.List(service.context, resourceQuery, "", nil)
	if err == nil {
		resourceList = result.Values()
	}
	return resourceList, err
}

// GetResourceDefinitionById will retrieve the azure resource based on the resource Id
func (service MonitorService) GetResourceDefinitionById(id string) (resources.GenericResource, error) {
	return service.resourceClient.GetByID(service.context, id, ApiVersion)
}

// GetMetricNamespaces will return all supported namespaces based on the resource id and namespace
func (service *MonitorService) GetMetricNamespaces(resourceId string) (insights.MetricNamespaceCollection, error) {
	return service.metricNamespaceClient.List(service.context, resourceId, "")
}

// GetMetricDefinitions will return all supported metrics based on the resource id and namespace
func (service *MonitorService) GetMetricDefinitions(resourceId string, namespace string) (insights.MetricDefinitionCollection, error) {
	return service.metricDefinitionClient.List(service.context, resourceId, namespace)
}

// GetMetricValues will return the metric values based on the resource and metric details
func (service *MonitorService) GetMetricValues(resourceId string, namespace string, timegrain string, timespan string, metricNames []string, aggregations string, filter string) ([]insights.Metric, string, error) {
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
		resp, err := service.metricsClient.List(service.context, resourceId, timespan, tg, strings.Join(metricNames[i:end], ","),
			aggregations, nil, "", filter, insights.Data, namespace)

		// check for applied charges before returning any errors
		if resp.Cost != nil && *resp.Cost != 0 {
			service.log.Warnf("Charges amounted to %v are being applied while retrieving the metric values from the resource %s ", *resp.Cost, resourceId)
		}
		if err != nil {
			return metrics, "", err
		}
		interval = *resp.Interval
		metrics = append(metrics, *resp.Value...)

	}
	return metrics, interval, nil
}

// getResourceNameFormId maps resource group from resource ID
func getResourceNameFromId(path string) string {
	params := strings.Split(path, "/")
	if strings.HasSuffix(path, "/") {
		return params[len(params)-2]
	}
	return params[len(params)-1]

}

// getResourceTypeFromId maps resource group from resource ID
func getResourceTypeFromId(path string) string {
	params := strings.Split(path, "/")
	for i, param := range params {
		if param == "providers" {
			return fmt.Sprintf("%s/%s", params[i+1], params[i+2])
		}
	}
	return ""
}
