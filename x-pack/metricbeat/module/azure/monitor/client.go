package monitor

import (
	"context"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/services/preview/monitor/mgmt/2018-03-01/insights"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-03-01/resources"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/elastic/beats/x-pack/metricbeat/module/azure"
	"github.com/pkg/errors"
	"strings"
	"sync"
	"time"
)

// MonitorClient represents local client which will use the azure sdk go metricsclient
type Client struct {
	metricsClient          *insights.MetricsClient
	metricDefinitionClient *insights.MetricDefinitionsClient
	resourceClient         *resources.Client
	config                 azure.Config
	resources              ResourceList
}

type ResourceList struct {
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
	Type string
}

type Metric struct {
	resource     Resource
	namespace    string
	names        []string
	aggregations []string
	dimensions   []Dimension
	values       []MetricValue
}

type Dimension struct {
	name  string
	value string
}

type MetricValue struct {
	name    string
	average *float64
	min     *float64
	max     *float64
	total   *float64
	count   *int64
}

// New instantiates the an Azure monitoring client
func (client *Client) New(config azure.Config) error {
	clientConfig := auth.NewClientCredentialsConfig(config.ClientId, config.ClientSecret, config.TenantId)
	authorizer, err := clientConfig.Authorizer()
	if err != nil {
		return err
	}
	metricsClient := insights.NewMetricsClient(config.SubscriptionId)
	metricsDefinitionClient := insights.NewMetricDefinitionsClient(config.SubscriptionId)
	resourceClient := resources.NewClient(config.SubscriptionId)
	metricsClient.Authorizer = authorizer
	metricsDefinitionClient.Authorizer = authorizer
	resourceClient.Authorizer = authorizer
	client.metricDefinitionClient = &metricsDefinitionClient
	client.metricsClient = &metricsClient
	client.resourceClient = &resourceClient
	client.config = config
	client.resources.refreshInterval = config.RefreshListInterval
	return nil
}

// InitResources returns the list of resources and maps them.
func (client *Client) InitResources() error {
	if !client.resources.expired() {
		return nil
	}
	var metrics []Metric
	for _, resource := range client.config.Resources {
		if resource.Group != "" {
			var top int32 = 20
			resourceList, err := client.resourceClient.ListByResourceGroup(context.Background(), resource.Group, fmt.Sprintf("resourceType eq '%s'", resource.Type), "true", &top)
			if err != nil {
				return errors.Wrapf(err, "error while listing resources by resource group %s  and filter %s", resource.Group, resource.Type)
			}
			for _, res := range resourceList.Values() {
				for _, namespace := range resource.Namespace {
					metrics = append(metrics, mapNamespace(namespace, res)...)

				}
			}
		}
		if resource.Id != "" {
			for _, namespace := range resource.Namespace {
				re, err := client.resourceClient.GetByID(context.Background(), resource.Id)
				if err != nil {
					return errors.Wrapf(err, "error while retrieving resource by id  %s ", resource.Id)
				}
				metrics = append(metrics, mapNamespace(namespace, re)...)
			}
		}
	}
	client.resources.metrics = metrics
	return nil
}

func mapNamespace(namespace azure.NamespaceConfig, resource resources.GenericResource) []Metric {
	var metrics []Metric
	for _, metric := range namespace.Metrics {
		var dim []Dimension
		if len(metric.Dimensions) > 0 {
			for _, dimension := range metric.Dimensions {
				//dim = append(dim, dimension.Name+" eq '"+dimension.Value+"'")
				dim = append(dim, Dimension{name: dimension.Name, value: dimension.Value})
			}
		}
		metrics = append(metrics, Metric{resource: Resource{Id: *resource.ID, Name: *resource.Name, Location: *resource.Location, Type: *resource.Type},
			namespace: namespace.Name, names: metric.Name, aggregations: metric.Aggregations, dimensions: dim})
	}
	return metrics
}

func (p *ResourceList) expired() bool {
	if p.refreshInterval <= 0 {
		return true
	}

	p.lastUpdate.Lock()
	defer p.lastUpdate.Unlock()

	if p.lastUpdate.Add(p.refreshInterval).After(time.Now()) {
		return false
	}
	p.lastUpdate.Time = time.Now()
	return true
}

// ListMetricDefinitions returns the list of metrics available for the specified resource in the form "Localized Name (metric name)".
func (client *Client) ListMetricDefinitions(resourceURI string) ([]string, error) {
	result, err := client.metricDefinitionClient.List(context.Background(), resourceURI, "")
	if err != nil {
		return nil, err
	}
	metrics := make([]string, len(*result.Value))
	for i := range *result.Value {
		metrics[i] = fmt.Sprintf("%s (%s)", *(*result.Value)[i].Name.LocalizedValue, *(*result.Value)[i].Name.Value)
	}
	return metrics, nil
}

// GetMetricValues returns the specified metric data points for the specified resource ID/namespace.
func (client *Client) GetMetricValues() error {
	for i, metric := range client.resources.metrics {
		endTime := time.Now().UTC()
		startTime := endTime.Add(client.config.Period * (-1))
		timespan := fmt.Sprintf("%s/%s", startTime.Format(time.RFC3339), endTime.Format(time.RFC3339))
		interval := "PT1M" //to do : calculate interval
		var filter string
		if len(metric.dimensions) > 0 {
			var filterList []string
			for _, dim := range metric.dimensions {
				filterList = append(filterList, dim.name+" eq '"+dim.value+"'")
			}
			filter = strings.Join(filterList, " AND ")
		}
		resp, err := client.metricsClient.List(context.Background(), metric.resource.Id, timespan, &interval, strings.Join(metric.names, ","),
			strings.Join(metric.aggregations, ","), nil, "", filter, insights.Data, metric.namespace)
		if err != nil {
			return err
		}
		for _, v := range *resp.Value {
			for _, t := range *v.Timeseries {
				for _, mv := range *t.Data {
					var val MetricValue
					val.name = *v.Name.Value
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
