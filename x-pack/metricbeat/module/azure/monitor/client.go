// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package monitor

import (
	"errors"
	"fmt"
	"github.com/elastic/beats/libbeat/logp"
	"strings"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-03-01/resources"
	"github.com/elastic/beats/x-pack/metricbeat/module/azure"
)

// Client represents the azure client which will make use of the azure sdk go metrics related clients
type Client struct {
	azureMonitorService AzureService
	config              azure.Config
	resources           ResourceConfiguration
	log                 *logp.Logger
}

// ResourceConfiguration represents the resource related configuration entered by the user
type ResourceConfiguration struct {
	metrics         []Metric
	refreshInterval time.Duration
	lastUpdate      struct {
		time.Time
		sync.Mutex
	}
}

// Resource will contain the main azure resource details
type Resource struct {
	ID       string
	Name     string
	Location string
	Type     string
}

// Metric will contain the main azure metric details
type Metric struct {
	resource     Resource
	namespace    string
	names        []string
	aggregations string
	dimensions   []Dimension
	values       []MetricValue
	timeGrain    string
}

// Dimension represents the azure metric dimension details
type Dimension struct {
	name  string
	value string
}

// MetricValue represents the azure metric values
type MetricValue struct {
	name      string
	average   *float64
	min       *float64
	max       *float64
	total     *float64
	count     *float64
	timestamp time.Time
}

// NewClient instantiates the an Azure monitoring client
func NewClient(config azure.Config) (*Client, error) {
	azureMonitorService, err := NewAzureService(config.ClientID, config.ClientSecret, config.TenantID, config.SubscriptionID)
	if err != nil {
		return nil, err
	}
	client := &Client{
		azureMonitorService: azureMonitorService,
		config:              config,
		log:                 logp.NewLogger("azure monitor client"),
	}
	client.resources.refreshInterval = config.RefreshListInterval
	return client, nil
}

// InitResources returns the list of resources and maps them.
func (client *Client) InitResources() error {
	if !client.resources.expired() {
		return nil
	}
	var metrics []Metric
	if len(client.config.Resources) == 0 {
		return errors.New("no resource options were configured")
	}
	for _, resource := range client.config.Resources {
		resourceList, err := client.azureMonitorService.GetResourceDefinitions(resource.ID, resource.Group, resource.Type, resource.Query)
		if err == nil {
			for _, res := range resourceList {
				for _, metric := range resource.Metrics {
					met, err := client.mapMetric(metric, res)
					if err != nil {
						return err
					}
					metrics = append(metrics, met)
				}
			}
		}
	}
	client.resources.metrics = metrics
	return nil
}

// GetMetricValues returns the specified metric data points for the specified resource ID/namespace.
func (client *Client) GetMetricValues() error {
	for i, metric := range client.resources.metrics {
		client.resources.metrics[i].values = nil
		endTime := time.Now().UTC()
		startTime := endTime.Add(client.config.Period * (-1))
		timespan := fmt.Sprintf("%s/%s", startTime.Format(time.RFC3339), endTime.Format(time.RFC3339))
		var filter string
		if len(metric.dimensions) > 0 {
			var filterList []string
			for _, dim := range metric.dimensions {
				filterList = append(filterList, dim.name+" eq '"+dim.value+"'")
			}
			filter = strings.Join(filterList, " AND ")
		}
		resp, err := client.azureMonitorService.GetMetricValues(metric.resource.ID, metric.namespace, metric.timeGrain, timespan, metric.names,
			metric.aggregations, filter)
		if err != nil {
			continue
		}
		for _, v := range resp {
			for _, t := range *v.Timeseries {
				for _, mv := range *t.Data {
					var val MetricValue
					val.name = *v.Name.Value
					val.timestamp = mv.TimeStamp.Time
					if mv.Minimum != nil {
						val.min = mv.Minimum
					}
					if mv.Maximum != nil {
						val.max = mv.Maximum
					}
					if mv.Average != nil {
						val.average = mv.Average
					}
					if mv.Total != nil {
						val.total = mv.Total
					}
					if mv.Count != nil {
						val.count = mv.Count
					}
					client.resources.metrics[i].values = append(client.resources.metrics[i].values, val)
				}
			}
		}
	}
	return nil
}

// mapMetric should validate and map the metric related configuration to relevant azure monitor api parameters
func (client *Client) mapMetric(metric azure.MetricConfig, resource resources.GenericResource) (Metric, error) {
	var dim []Dimension
	var supportedMetricNames []string
	var unsupportedMetricNames []string
	var supportedAggregations []string
	var unsupportedAggregations []string
	metricDefinitions, err := client.azureMonitorService.GetMetricDefinitions(*resource.ID, metric.Namespace)
	if err != nil {
		client.log.Errorf("No metric definitions were found for resource %s and namespace %s. Error %v", *resource.ID, metric.Namespace, err)
		return Metric{}, err
	}

	// validate metric names
	// if all metric names are selected (*)
	if stringInSlice("*", metric.Name) {
		for _, definition := range metricDefinitions {
			supportedMetricNames = append(supportedMetricNames, *definition.Name.Value)
		}
	} else {
		// verify if configured metric names are valid, return log error event for the invalid ones, map only  the valid metric names
		supportedMetricNames, unsupportedMetricNames = filterMetrics(metric.Name, metricDefinitions)
		if len(unsupportedMetricNames) > 0 {
			client.log.Errorf("The metric names configured : %s are not supported for namespace %s ", strings.Join(unsupportedMetricNames, ","), metric.Namespace)
		}
	}

	//validate aggregations and filter on supported ones
	metricDefs := getMetricDefinitionsByNames(metricDefinitions, supportedMetricNames)
	supportedAggregations, unsupportedAggregations = filterAggregations(metric.Aggregations, metricDefs)
	if len(unsupportedAggregations) > 0 {
		client.log.Errorf("The aggregations configured : %s are not supported for some of the metrics selected %s ", strings.Join(unsupportedAggregations, ","), strings.Join(supportedMetricNames, ","))
	}

	// map dimensions
	if len(metric.Dimensions) > 0 {
		for _, dimension := range metric.Dimensions {
			dim = append(dim, Dimension{name: dimension.Name, value: dimension.Value})
		}
	}
	return Metric{resource: Resource{ID: *resource.ID, Name: *resource.Name, Location: *resource.Location, Type: *resource.Type},
		namespace: metric.Namespace, names: supportedMetricNames, aggregations: strings.Join(supportedAggregations, ","), dimensions: dim, timeGrain: metric.Timegrain}, nil

}
