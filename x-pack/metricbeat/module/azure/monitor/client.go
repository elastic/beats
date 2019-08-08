package monitor

import (
	"context"
	"fmt"
	"github.com/elastic/beats/libbeat/logp"
	"strings"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/preview/monitor/mgmt/2019-06-01/insights"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-03-01/resources"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/elastic/beats/x-pack/metricbeat/module/azure"
	"github.com/pkg/errors"
)

// MonitorClient represents local client which will use the azure sdk go metricsclient
type Client struct {
	metricsClient          *insights.MetricsClient
	metricDefinitionClient *insights.MetricDefinitionsClient
	resourceClient         *resources.Client
	config                 azure.Config
	resourceConfig         ResourceConfiguration
	log                    *logp.Logger
}

type ResourceConfiguration struct {
	metrics         []Metric
	refreshInterval time.Duration
	lastUpdate      struct {
		time.Time
		sync.Mutex
	}
}

type Resource struct {
	Id       string
	Name     string
	Location string
	Type     string
}

type Metric struct {
	resource     Resource
	namespace    string
	names        string
	aggregations string
	dimensions   []Dimension
	values       []MetricValue
	timeGrain    string
}

type Dimension struct {
	name  string
	value string
}

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
	clientConfig := auth.NewClientCredentialsConfig(config.ClientId, config.ClientSecret, config.TenantId)
	authorizer, err := clientConfig.Authorizer()
	if err != nil {
		return nil, err
	}
	metricsClient := insights.NewMetricsClient(config.SubscriptionId)
	metricsDefinitionClient := insights.NewMetricDefinitionsClient(config.SubscriptionId)
	resourceClient := resources.NewClient(config.SubscriptionId)
	metricsClient.Authorizer = authorizer
	metricsDefinitionClient.Authorizer = authorizer
	resourceClient.Authorizer = authorizer
	client := &Client{
		metricDefinitionClient: &metricsDefinitionClient,
		metricsClient:          &metricsClient,
		resourceClient:         &resourceClient,
		config:                 config,
	}
	client.resourceConfig.refreshInterval = config.RefreshListInterval

	return client, nil
}

// InitResources returns the list of resources and maps them.
func (client *Client) InitResources() error {
	if !client.resourceConfig.expired() {
		return nil
	}
	var metrics []Metric
	for _, resource := range client.config.Resources {
		//check for all options to identify resources : resourceid, resource group and resource query
		if resource.Id != "" {
			for _, metric := range resource.Metrics {
				re, err := client.resourceClient.GetByID(context.Background(), resource.Id)
				if err != nil {
					client.log.Errorf(" error while retrieving resource by id  %s : %s", resource.Id, err)
				} else {
					met, err := client.mapMetric(metric, re)
					if err != nil {
						return err
					}
					metrics = append(metrics, met)
				}
			}
		}
		if resource.Group != "" {
			var top int32 = 200
			resourceList, err := client.resourceClient.ListByResourceGroup(context.Background(), resource.Group, fmt.Sprintf("resourceType eq '%s'", resource.Type), "true", &top)
			if err != nil {
				return errors.Wrapf(err, "error while listing resources by resource group %s  and filter %s", resource.Group, resource.Type)
			}
			for _, res := range resourceList.Values() {
				for _, metric := range resource.Metrics {
					met, err := client.mapMetric(metric, res)
					if err != nil {
						return err
					}
					metrics = append(metrics, met)

				}
			}
		}
		if resource.Query != "" {
			var top int32 = 200
			resourceList, err := client.resourceClient.List(context.Background(), resource.Query, "true", &top)
			if err != nil {
				return errors.Wrapf(err, "error while listing resources by filter %s", resource.Query)
			}
			for _, res := range resourceList.Values() {
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
	client.resourceConfig.metrics = metrics
	return nil
}

func (client *Client) mapMetric(metric azure.MetricConfig, resource resources.GenericResource) (Metric, error) {
	var dim []Dimension
	var supportedMetricNames []string
	var unsupportedMetricNames []string
	var supportedAggregations []string
	var unsupportedAggregations []string
	metricDefinitions, err := client.metricDefinitionClient.List(context.Background(), *resource.ID, metric.Namespace)
	if err != nil {
		return Metric{}, err
	}

	// validate metric names
	// if all metric names are selected (*)
	if stringInSlice("*", metric.Name) {
		for _, definition := range *metricDefinitions.Value {
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
	if len(metric.Aggregations) != 0 {
		metricDefs := getMetricDefinitionsByNames(metricDefinitions, supportedMetricNames)
		supportedAggregations, unsupportedAggregations = filterAggregations(metric.Aggregations, metricDefs)
		if len(unsupportedAggregations) > 0 {
			client.log.Errorf("The aggregations configured : %s are not supported for some of the metrics selected %s ", strings.Join(unsupportedAggregations, ","), strings.Join(supportedMetricNames, ","))
		}
	}
	// map dimensions
	if len(metric.Dimensions) > 0 {
		for _, dimension := range metric.Dimensions {
			dim = append(dim, Dimension{name: dimension.Name, value: dimension.Value})
		}
	}
	return Metric{resource: Resource{Id: *resource.ID, Name: *resource.Name, Location: *resource.Location, Type: *resource.Type},
		namespace: metric.Namespace, names: strings.Join(supportedMetricNames, ","), aggregations: strings.Join(supportedAggregations, ","), dimensions: dim}, nil

}

// GetMetricValues returns the specified metric data points for the specified resource ID/namespace.
func (client *Client) GetMetricValues() error {
	for i, metric := range client.resourceConfig.metrics {
		client.resourceConfig.metrics[i].values = nil
		endTime := time.Now().UTC()
		startTime := endTime.Add(client.config.Period * (-1))
		timespan := fmt.Sprintf("%s/%s", startTime.Format(time.RFC3339), endTime.Format(time.RFC3339))
		//interval := "PT1M"
		//interval=""
		var filter string
		if len(metric.dimensions) > 0 {
			var filterList []string
			for _, dim := range metric.dimensions {
				filterList = append(filterList, dim.name+" eq '"+dim.value+"'")
			}
			filter = strings.Join(filterList, " AND ")
		}
		resp, err := client.metricsClient.List(context.Background(), metric.resource.Id, timespan, nil, metric.names,
			metric.aggregations, nil, "", filter, insights.Data, metric.namespace)
		if err != nil {
			return err
		}
		for _, v := range *resp.Value {
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
					client.resourceConfig.metrics[i].values = append(client.resourceConfig.metrics[i].values, val)
				}
			}
		}
	}
	return nil
}
