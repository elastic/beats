// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package monitor

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/preview/monitor/mgmt/2019-06-01/insights"
	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"

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
	ID           string
	Name         string
	Location     string
	Type         string
	Group        string
	Tags         map[string]string
	Subscription string
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
func (client *Client) InitResources(report mb.ReporterV2) error {
	if len(client.config.Resources) == 0 {
		return errors.New("no resource options defined")
	}
	if !client.resources.expired() {
		return nil
	}
	var metrics []Metric
	for _, resource := range client.config.Resources {
		resourceList, err := client.azureMonitorService.GetResourceDefinitions(resource.ID, resource.Group, resource.Type, resource.Query)
		if err != nil {
			err = errors.Wrap(err, "failed to retrieve resources")
			client.logError(report, err)
			continue
		}
		if len(resourceList.Values()) == 0 {
			err = errors.Errorf("failed to retrieve resources: No resources returned using the configuration options resource ID %s, resource group %s, resource type %s, resource query %s",
				resource.ID, resource.Group, resource.Type, resource.Query)
			client.logError(report, err)
			continue
		}
		for _, res := range resourceList.Values() {
			for _, metric := range resource.Metrics {
				met, err := client.mapMetric(metric, res)
				if err != nil {
					client.logError(report, err)
					continue
				}
				metrics = append(metrics, met)
			}
		}
	}

	// users could add or remove resources while metricbeat is running so we could encounter the situation where resources are unavailable, we log and create an event if this is the case (see above)
	// but we return an error when absolutely no resources are found
	if len(metrics) == 0 {
		return errors.New("no resources were found based on all the configurations options entered")
	}

	client.resources.metrics = metrics
	return nil
}

// GetMetricValues returns the specified metric data points for the specified resource ID/namespace.
func (client *Client) GetMetricValues(report mb.ReporterV2) error {
	// loop over the set of metrics
	for i, metric := range client.resources.metrics {
		// select period to collect metrics, will double the interval value in order to retrieve any missing values
		endTime := time.Now().UTC()
		startTime := endTime.Add(client.config.Period * (-2))
		timespan := fmt.Sprintf("%s/%s", startTime.Format(time.RFC3339), endTime.Format(time.RFC3339))

		// build the 'filter' parameter which will contain any dimensions configured
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
			err = errors.Wrapf(err, "error while listing metric values by resource ID %s and namespace  %s", metric.resource.ID, metric.namespace)
			client.logError(report, err)
		} else {
			err = client.mapMetricValues(i, resp)
			client.logError(report, err)
		}
	}
	return nil
}

// mapMetric should validate and map the metric related configuration to relevant azure monitor api parameters
func (client *Client) mapMetric(metric azure.MetricConfig, resource resources.GenericResource) (Metric, error) {
	var met = Metric{}
	// get all metrics supported by the namespace provided
	metricDefinitions, err := client.azureMonitorService.GetMetricDefinitions(*resource.ID, metric.Namespace)
	if err != nil {
		client.log.Errorf("no metric definitions were found for resource %s and namespace %s. Error %v", *resource.ID, metric.Namespace, err)
		return met, err
	}
	if len(*metricDefinitions.Value) == 0 {
		client.log.Error("no metric definitions were found for resource %s and namespace %s. Error %v", *resource.ID, metric.Namespace)
		return met, err
	}

	// validate metric names
	// check if all metric names are selected (*)
	var supportedMetricNames []string
	var unsupportedMetricNames []string
	if stringInSlice("*", metric.Name) {
		for _, definition := range *metricDefinitions.Value {
			supportedMetricNames = append(supportedMetricNames, *definition.Name.Value)
		}
	} else {
		// verify if configured metric names are valid, return log error event for the invalid ones, map only  the valid metric names
		supportedMetricNames, unsupportedMetricNames = filterMetrics(metric.Name, *metricDefinitions.Value)
		if len(unsupportedMetricNames) > 0 {
			client.log.Errorf("none of metric names configured are supported by the resources found : %s are not supported for namespace %s ",
				strings.Join(unsupportedMetricNames, ","), metric.Namespace)
		}
	}
	if len(supportedMetricNames) == 0 {
		return met, errors.Errorf("the metric names configured : %s are not supported for namespace %s ", strings.Join(metric.Name, ","), metric.Namespace)
	}

	//validate aggregations and filter on supported ones
	var supportedAggregations []string
	var unsupportedAggregations []string
	metricDefs := getMetricDefinitionsByNames(*metricDefinitions.Value, supportedMetricNames)
	supportedAggregations, unsupportedAggregations = filterAggregations(metric.Aggregations, metricDefs)
	if len(unsupportedAggregations) > 0 {
		client.log.Errorf("the aggregations configured : %s are not supported for some of the metrics selected %s ", strings.Join(unsupportedAggregations, ","),
			strings.Join(supportedMetricNames, ","))
	}
	if len(supportedAggregations) == 0 {
		return met, errors.Errorf("no shared aggregations were found based on the aggregation values configured or supported between the metrics : %s",
			strings.Join(supportedMetricNames, ","))
	}

	// map dimensions
	var dim []Dimension
	if len(metric.Dimensions) > 0 {
		for _, dimension := range metric.Dimensions {
			dim = append(dim, Dimension{name: dimension.Name, value: dimension.Value})
		}
	}

	met = Metric{resource: Resource{ID: *resource.ID, Name: *resource.Name, Location: *resource.Location, Type: *resource.Type, Group: mapResourceGroupFormID(*resource.ID),
		Tags: mapTags(resource.Tags), Subscription: client.config.SubscriptionID},
		namespace: metric.Namespace, names: supportedMetricNames, aggregations: strings.Join(supportedAggregations, ","), dimensions: dim, timeGrain: metric.Timegrain}

	//map previous metric values if existing
	for _, prevMet := range client.resources.metrics {
		if len(prevMet.values) != 0 && matchMetrics(prevMet, met) {
			met.values = prevMet.values
		}
	}
	return met, nil
}

func (client *Client) mapMetricValues(index int, metrics []insights.Metric) error {
	if len(metrics) == 0 {
		return errors.New("no metric values found")
	}

	// compare with the previously returned values and filter out any double records
	var previousMetrics []MetricValue
	previousMetrics = client.resources.metrics[index].values
	client.resources.metrics[index].values = nil
	for _, v := range metrics {
		for _, t := range *v.Timeseries {
			for _, mv := range *t.Data {
				if metricExists(*v.Name.Value, mv, previousMetrics) || metricIsEmpty(mv) {
					continue
				}
				// define the new metric value and match aggregations values
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
				client.resources.metrics[index].values = append(client.resources.metrics[index].values, val)
			}
		}
	}
	return nil
}

// logError is used to reduce the number of lines written when logging errors
func (client *Client) logError(report mb.ReporterV2, err error) {
	client.log.Error(err)
	report.Error(err)
}
