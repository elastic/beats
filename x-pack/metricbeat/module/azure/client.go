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

// mapResourceMetrics function type will map the configuration options to client metrics (depending on the metricset)
type mapResourceMetrics func(client *Client, resources []resources.GenericResource, resourceConfig ResourceConfig) ([]Metric, error)

// NewClient instantiates the an Azure monitoring client
func NewClient(config Config) (*Client, error) {
	azureMonitorService, err := NewService(config.ClientId, config.ClientSecret, config.TenantId, config.SubscriptionId)
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

// InitResources function will retrieve and validate the resources configured by the users and then map the information configured to client metrics.
// the mapMetric function sent in this case will handle the mapping part as different metric and aggregation options work for different metricsets
func (client *Client) InitResources(fn mapResourceMetrics, report mb.ReporterV2) error {
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
		resourceList, err := client.AzureMonitorService.GetResourceDefinitions(resource.Id, resource.Group, resource.Type, resource.Query)
		if err != nil {
			err = errors.Wrap(err, "failed to retrieve resources")
			return err
		}
		if len(resourceList.Values()) == 0 {
			err = errors.Errorf("failed to retrieve resources: No resources returned using the configuration options resource ID %s, resource group %s, resource type %s, resource query %s",
				resource.Id, resource.Group, resource.Type, resource.Query)
			client.Log.Error(err)
			continue
		}
		resourceMetrics, err := fn(client, resourceList.Values(), resource)
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
	client.Resources.Metrics = metrics
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
		resp, timegrain, err := client.AzureMonitorService.GetMetricValues(metric.Resource.SubId, metric.Namespace, metric.TimeGrain, timespan, metric.Names,
			metric.Aggregations, filter)
		if err != nil {
			err = errors.Wrapf(err, "error while listing metric values by resource ID %s and namespace  %s", metric.Resource.SubId, metric.Namespace)
			client.Log.Error(err)
			report.Error(err)
		} else {
			for i, currentMetric := range client.Resources.Metrics {
				if matchMetrics(currentMetric, metric) {
					current := mapMetricValues(resp, currentMetric.Values, endTime.Truncate(time.Minute).Add(interval*(-1)), endTime.Truncate(time.Minute))
					client.Resources.Metrics[i].Values = current
					if client.Resources.Metrics[i].TimeGrain == "" {
						client.Resources.Metrics[i].TimeGrain = timegrain
					}
					resultedMetrics = append(resultedMetrics, client.Resources.Metrics[i])
				}
			}
		}
	}
	return resultedMetrics
}

// CreateMetric function will create a client metric based on the resource and metrics configured
func (client *Client) CreateMetric(selectedResourceID string, resource resources.GenericResource, resourceSize string, namespace string, metrics []string, aggregations string, dimensions []Dimension, timegrain string) Metric {
	met := Metric{
		Resource: Resource{
			SubId:        selectedResourceID,
			Id:           *resource.ID,
			Name:         *resource.Name,
			Location:     *resource.Location,
			Type:         *resource.Type,
			Group:        getResourceGroupFromId(*resource.ID),
			Tags:         mapTags(resource.Tags),
			Subscription: client.Config.SubscriptionId,
			Size:         resourceSize,
		},
		Namespace:    namespace,
		Names:        metrics,
		Dimensions:   dimensions,
		Aggregations: aggregations,
		TimeGrain:    timegrain,
	}
	for _, prevMet := range client.Resources.Metrics {
		if len(prevMet.Values) != 0 && matchMetrics(prevMet, met) {
			met.Values = prevMet.Values
		}
	}
	return met
}

// MapMetricByPrimaryAggregation will map the primary aggregation of the metric definition to the client metric
func (client *Client) MapMetricByPrimaryAggregation(metrics []insights.MetricDefinition, resource resources.GenericResource, selectedResourceID string, resourceSize string, namespace string, dim []Dimension, timegrain string) []Metric {
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
		if selectedResourceID == "" {
			selectedResourceID = *resource.ID
		}
		clientMetrics = append(clientMetrics, client.CreateMetric(selectedResourceID, resource, resourceSize, namespace, metricNames, key, dim, timegrain))
	}
	return clientMetrics
}
