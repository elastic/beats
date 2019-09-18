// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azure

import (
	"fmt"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/preview/monitor/mgmt/2019-06-01/insights"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-03-01/resources"
	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
)

// Client represents the azure client which will make use of the azure sdk go metrics related clients
type Client struct {
	AzureMonitorService Service
	Config              Config
	Resources           ResourceConfiguration
	Log                 *logp.Logger
}

// mapMetric function type will map the configuration options to client metrics (depending on the metricset)
type mapMetric func(client *Client, metric MetricConfig, resource resources.GenericResource) ([]Metric, error)

// NewClient instantiates the an Azure monitoring client
func NewClient(config Config) (*Client, error) {
	azureMonitorService, err := NewService(config.ClientID, config.ClientSecret, config.TenantID, config.SubscriptionID)
	if err != nil {
		return nil, err
	}
	client := &Client{
		AzureMonitorService: azureMonitorService,
		Config:              config,
		Log:                 logp.NewLogger("azure monitor client"),
	}
	client.Resources.RefreshInterval = config.RefreshListInterval
	return client, nil
}

// InitResources returns the list of resources and maps them.
func (client *Client) InitResources(fn mapMetric, report mb.ReporterV2) error {
	if len(client.Config.Resources) == 0 {
		return errors.New("no resource options defined")
	}
	// check if refresh interval has been set and if it has expired
	if !client.Resources.Expired() {
		return nil
	}
	var metrics []Metric
	for _, resource := range client.Config.Resources {
		// retrieve azure resources information
		resourceList, err := client.AzureMonitorService.GetResourceDefinitions(resource.ID, resource.Group, resource.Type, resource.Query)
		if err != nil {
			err = errors.Wrap(err, "failed to retrieve resources")
			return err
		}
		if len(resourceList.Values()) == 0 {
			err = errors.Errorf("failed to retrieve resources: No resources returned using the configuration options resource ID %s, resource group %s, resource type %s, resource query %s",
				resource.ID, resource.Group, resource.Type, resource.Query)
			client.LogError(report, err)
			continue
		}
		for _, res := range resourceList.Values() {
			for _, metric := range resource.Metrics {
				met, err := fn(client, metric, res)
				if err != nil {
					return err
				}
				metrics = append(metrics, met...)
			}
		}
	}
	// users could add or remove resources while metricbeat is running so we could encounter the situation where resources are unavailable, we log and create an event if this is the case (see above)
	// but we return an error when absolutely no resources are found
	if len(metrics) == 0 {
		return errors.New("no resources were found based on all the configurations options entered")
	}
	client.Resources.Metrics = metrics
	return nil
}

// GetMetricValues returns the specified metric data points for the specified resource ID/namespace.
func (client *Client) GetMetricValues(report mb.ReporterV2) error {
	// loop over the set of metrics
	for i, metric := range client.Resources.Metrics {
		// select period to collect metrics, will double the interval value in order to retrieve any missing values
		endTime := time.Now().UTC()
		startTime := endTime.Add(client.Config.Period * (-2))
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
		resp, err := client.AzureMonitorService.GetMetricValues(metric.Resource.ID, metric.Namespace, metric.TimeGrain, timespan, metric.Names,
			metric.Aggregations, filter)
		if err != nil {
			err = errors.Wrapf(err, "error while listing metric values by resource ID %s and namespace  %s", metric.Resource.ID, metric.Namespace)
			client.LogError(report, err)
		} else {
			current, err := mapMetricValues(resp, client.Resources.Metrics[i].Values, endTime.Truncate(time.Minute).Add(client.Config.Period*(-1)), endTime.Truncate(time.Minute))
			if err != nil {
				client.LogError(report, err)
			}
			client.Resources.Metrics[i].Values = current
		}
	}
	return nil
}

// LogError is used to reduce the number of lines written when logging errors
func (client *Client) LogError(report mb.ReporterV2, err error) {
	client.Log.Error(err)
	report.Error(err)
}

// CreateMetric function will create a client metric based on the resource and metrics configured
func (client *Client) CreateMetric(resource resources.GenericResource, namespace string, metrics []string, aggregations string, dimensions []Dimension, timegrain string) Metric {
	met := Metric{Resource: Resource{ID: *resource.ID, Name: *resource.Name, Location: *resource.Location, Type: *resource.Type, Group: getResourceGroupFormID(*resource.ID),
		Tags: mapTags(resource.Tags), Subscription: client.Config.SubscriptionID},
		Namespace: namespace, Names: metrics, Dimensions: dimensions, Aggregations: aggregations, TimeGrain: timegrain}
	for _, prevMet := range client.Resources.Metrics {
		if len(prevMet.Values) != 0 && matchMetrics(prevMet, met) {
			met.Values = prevMet.Values
		}
	}
	return met
}

// MapMetricByPrimaryAggregation will map the primary aggregation of the metric definition to the client metric
func MapMetricByPrimaryAggregation(client *Client, metrics []insights.MetricDefinition, resource resources.GenericResource, namespace string, dim []Dimension, timegrain string) []Metric {
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
		clientMetrics = append(clientMetrics, client.CreateMetric(resource, namespace, metricNames, key, dim, timegrain))
	}
	return clientMetrics
}
