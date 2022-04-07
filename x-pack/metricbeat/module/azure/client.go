// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azure

import (
	"fmt"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/preview/monitor/mgmt/2019-06-01/insights"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-10-01/resources"
	"github.com/pkg/errors"

	"github.com/elastic/beats/v8/libbeat/logp"
	"github.com/elastic/beats/v8/metricbeat/mb"
)

// Client represents the azure client which will make use of the azure sdk go metrics related clients
type Client struct {
	AzureMonitorService    Service
	Config                 Config
	ResourceConfigurations ResourceConfiguration
	Log                    *logp.Logger
	Resources              []Resource
}

// mapResourceMetrics function type will map the configuration options to client metrics (depending on the metricset)
type mapResourceMetrics func(client *Client, resources []resources.GenericResourceExpanded, resourceConfig ResourceConfig) ([]Metric, error)

// NewClient instantiates the an Azure monitoring client
func NewClient(config Config) (*Client, error) {
	azureMonitorService, err := NewService(config)
	if err != nil {
		return nil, err
	}
	client := &Client{
		AzureMonitorService: azureMonitorService,
		Config:              config,
		Log:                 logp.NewLogger("azure monitor client"),
	}
	client.ResourceConfigurations.RefreshInterval = config.RefreshListInterval
	return client, nil
}

// InitResources function will retrieve and validate the resources configured by the users and then map the information configured to client metrics.
// the mapMetric function sent in this case will handle the mapping part as different metric and aggregation options work for different metricsets
func (client *Client) InitResources(fn mapResourceMetrics) error {
	if len(client.Config.Resources) == 0 {
		return errors.New("no resource options defined")
	}
	// check if refresh interval has been set and if it has expired
	if !client.ResourceConfigurations.Expired() {
		return nil
	}
	var metrics []Metric
	//reset client resources
	client.Resources = []Resource{}
	for _, resource := range client.Config.Resources {
		// retrieve azure resources information
		resourceList, err := client.AzureMonitorService.GetResourceDefinitions(resource.Id, resource.Group, resource.Type, resource.Query)
		if err != nil {
			err = errors.Wrap(err, "failed to retrieve resources")
			return err
		}
		if len(resourceList) == 0 {
			err = errors.Errorf("failed to retrieve resources: No resources returned using the configuration options resource ID %s, resource group %s, resource type %s, resource query %s",
				resource.Id, resource.Group, resource.Type, resource.Query)
			client.Log.Error(err)
			continue
		}
		//map resources to the client
		for _, resource := range resourceList {
			if !containsResource(*resource.ID, client.Resources) {
				client.Resources = append(client.Resources, Resource{
					Id:           *resource.ID,
					Name:         *resource.Name,
					Location:     *resource.Location,
					Type:         *resource.Type,
					Group:        getResourceGroupFromId(*resource.ID),
					Tags:         mapTags(resource.Tags),
					Subscription: client.Config.SubscriptionId})
			}
		}
		resourceMetrics, err := fn(client, resourceList, resource)
		if err != nil {
			return err
		}
		metrics = append(metrics, resourceMetrics...)
	}
	// users could add or remove resources while metricbeat is running so we could encounter the situation where resources are unavailable we log an error message (see above)
	// we also log a debug message when absolutely no resources are found
	if len(metrics) == 0 {
		client.Log.Debug("no resources were found based on all the configurations options entered")
	}
	client.ResourceConfigurations.Metrics = metrics
	return nil
}

// GetMetricValues returns the specified metric data points for the specified resource ID/namespace.
func (client *Client) GetMetricValues(metrics []Metric, report mb.ReporterV2) []Metric {
	var resultedMetrics []Metric
	// loop over the set of metrics
	for _, metric := range metrics {
		// select period to collect metrics, will double the interval value in order to retrieve any missing values
		//if timegrain is larger than intervalx2 then interval will be assigned the timegrain value
		interval := client.Config.Period
		if t := convertTimegrainToDuration(metric.TimeGrain); t > interval*2 {
			interval = t
		}
		endTime := time.Now().UTC()
		startTime := endTime.Add(interval * (-2))
		timespan := fmt.Sprintf("%s/%s", startTime.Format(time.RFC3339), endTime.Format(time.RFC3339))

		// build the 'filter' parameter which will contain any dimensions configured
		var filter string
		if len(metric.Dimensions) > 0 {
			var filterList []string
			for _, dim := range metric.Dimensions {
				filterList = append(filterList, dim.Name+" eq '"+dim.Value+"'")
			}
			filter = strings.Join(filterList, " AND ")
		}
		resp, timegrain, err := client.AzureMonitorService.GetMetricValues(metric.ResourceSubId, metric.Namespace, metric.TimeGrain, timespan, metric.Names,
			metric.Aggregations, filter)
		if err != nil {
			err = errors.Wrapf(err, "error while listing metric values by resource ID %s and namespace  %s", metric.ResourceSubId, metric.Namespace)
			client.Log.Error(err)
			report.Error(err)
		} else {
			for i, currentMetric := range client.ResourceConfigurations.Metrics {
				if matchMetrics(currentMetric, metric) {
					current := mapMetricValues(resp, currentMetric.Values, endTime.Truncate(time.Minute).Add(interval*(-1)), endTime.Truncate(time.Minute))
					client.ResourceConfigurations.Metrics[i].Values = current
					if client.ResourceConfigurations.Metrics[i].TimeGrain == "" {
						client.ResourceConfigurations.Metrics[i].TimeGrain = timegrain
					}
					resultedMetrics = append(resultedMetrics, client.ResourceConfigurations.Metrics[i])
				}
			}
		}
	}
	return resultedMetrics
}

// CreateMetric function will create a client metric based on the resource and metrics configured
func (client *Client) CreateMetric(resourceId string, subResourceId string, namespace string, metrics []string, aggregations string, dimensions []Dimension, timegrain string) Metric {
	if subResourceId == "" {
		subResourceId = resourceId
	}
	met := Metric{
		ResourceId:    resourceId,
		ResourceSubId: subResourceId,
		Namespace:     namespace,
		Names:         metrics,
		Dimensions:    dimensions,
		Aggregations:  aggregations,
		TimeGrain:     timegrain,
	}
	for _, prevMet := range client.ResourceConfigurations.Metrics {
		if len(prevMet.Values) != 0 && matchMetrics(prevMet, met) {
			met.Values = prevMet.Values
		}
	}
	return met
}

// MapMetricByPrimaryAggregation will map the primary aggregation of the metric definition to the client metric
func (client *Client) MapMetricByPrimaryAggregation(metrics []insights.MetricDefinition, resourceId string, subResourceId string, namespace string, dim []Dimension, timegrain string) []Metric {
	var clientMetrics []Metric
	metricGroups := make(map[string][]insights.MetricDefinition)

	for _, met := range metrics {
		metricGroups[string(met.PrimaryAggregationType)] = append(metricGroups[string(met.PrimaryAggregationType)], met)
	}
	for key, metricGroup := range metricGroups {
		var metricNames []string
		for _, metricName := range metricGroup {
			metricNames = append(metricNames, *metricName.Name.Value)
		}
		clientMetrics = append(clientMetrics, client.CreateMetric(resourceId, subResourceId, namespace, metricNames, key, dim, timegrain))
	}
	return clientMetrics
}

// GetVMForMetaData func will retrieve the vm details in order to fill in the cloud metadata and also update the client resources
func (client *Client) GetVMForMetaData(resource *Resource, metricValues []MetricValue) VmResource {
	var vm VmResource
	resourceName := resource.Name
	resourceId := resource.Id
	// check first if this is a vm scaleset and the instance name is stored in the dimension value
	if dimension, ok := getDimension("VMName", metricValues[0].dimensions); ok {
		instanceId := getInstanceId(dimension.Value)
		if instanceId != "" {
			resourceId += fmt.Sprintf("/virtualMachines/%s", instanceId)
			resourceName = dimension.Value
		}
	}
	// if vm has been already added to the resource then it should be returned
	if existingVM, ok := getVM(resourceName, resource.Vms); ok {
		return existingVM
	}
	// an additional call is necessary in order to retrieve the vm specific details
	expandedResource, err := client.AzureMonitorService.GetResourceDefinitionById(resourceId)
	if err != nil {
		client.Log.Error(err, "could not retrieve the resource details by resource ID %s", resourceId)
		return VmResource{}
	}
	vm.Name = *expandedResource.Name
	if expandedResource.Properties != nil {
		if properties, ok := expandedResource.Properties.(map[string]interface{}); ok {
			if hardware, ok := properties["hardwareProfile"]; ok {
				if vmSz, ok := hardware.(map[string]interface{})["vmSize"]; ok {
					vm.Size = vmSz.(string)
				}
				if vmID, ok := properties["vmId"]; ok {
					vm.Id = vmID.(string)
				}
			}
		}
	}
	if len(vm.Size) == 0 && expandedResource.Sku != nil && expandedResource.Sku.Name != nil {
		vm.Size = *expandedResource.Sku.Name
	}
	// the client resource and selected resources are being updated in order to avoid additional calls
	client.AddVmToResource(resource.Id, vm)
	resource.Vms = append(resource.Vms, vm)
	return vm
}

// GetResourceForMetaData will retrieve resource details for the selected metric configuration
func (client *Client) GetResourceForMetaData(grouped Metric) Resource {
	for _, res := range client.Resources {
		if res.Id == grouped.ResourceId {
			return res
		}
	}
	return Resource{}
}

// AddVmToResource will add the vm details to the resource
func (client *Client) AddVmToResource(resourceId string, vm VmResource) {
	if len(vm.Id) > 0 && len(vm.Name) > 0 {
		for i, res := range client.Resources {
			if res.Id == resourceId {
				client.Resources[i].Vms = append(client.Resources[i].Vms, vm)
			}
		}
	}
}

// NewMockClient instantiates a new client with the mock azure service
func NewMockClient() *Client {
	azureMockService := new(MockService)
	client := &Client{
		AzureMonitorService: azureMockService,
		Config:              Config{},
		Log:                 logp.NewLogger("test azure monitor"),
	}
	return client
}
