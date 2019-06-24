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
	"time"
)

// AzureMonitorClient represents local client which will use the azure sdk go metricsclient
type AzureMonitorClient struct {
	metricsClient          *insights.MetricsClient
	metricDefinitionClient *insights.MetricDefinitionsClient
	resourceClient         *resources.Client
	config                 azure.Config
	accessTokenExpiresOn   time.Time
	metrics                []AzureMonitorMetric
}

type AzureMonitorMetric struct {
	resourcePath string
	namespace    string
	name         string
	values       []MetricValue
}

type MetricValue struct {
	average float64
	min     float64
	max     float64
	total   float64
	count   int64
}

// New instantiates the an Azure monitoring client
func (client *AzureMonitorClient) New(config azure.Config) error {
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
	return nil
}

// InitResources returns the list of resources and maps them.
func (client *AzureMonitorClient) InitResources() error {
	for _, metric := range client.config.Metrics {
		if metric.ResourceGroup != "" {
			var top int32 = 20
			resourceList, err := client.resourceClient.ListByResourceGroup(context.Background(), metric.ResourceGroup, fmt.Sprintf("resourceType eq '%s'", metric.ResourceType), "true", &top)
			hell := resourceList.Values()
			_ = hell
			if err != nil {
				return errors.Wrapf(err, "error while listing resources by resource group %s  and filter %s", metric.ResourceGroup, metric.ResourceType)
			}
			for _, resource := range resourceList.Values() {
				client.metrics = append(client.metrics, AzureMonitorMetric{resourcePath: *resource.ID, namespace: metric.Namespace, name: metric.MetricName})
			}
		}
		if metric.ResourceId != "" {
			client.metrics = append(client.metrics, AzureMonitorMetric{resourcePath: metric.ResourceId, namespace: metric.Namespace, name: metric.MetricName})
		}
	}
	return nil
}

// ListMetricDefinitions returns the list of metrics available for the specified resource in the form "Localized Name (metric name)".
func (client *AzureMonitorClient) ListMetricDefinitions(resourceURI string) ([]string, error) {
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

// GetMetricsData returns the specified metric data points for the specified resource ID spanning the last five minutes.
func (client *AzureMonitorClient) GetMetricsData(metric AzureMonitorMetric) ([]MetricValue, error) {
	endTime := time.Now().UTC()
	startTime := endTime.Add(client.config.Period * (-1))
	timespan := fmt.Sprintf("%s/%s", startTime.Format(time.RFC3339), endTime.Format(time.RFC3339))
	metrics := []string{metric.name}
	interval := "PT1M" //to do : calculate interval
	resp, err := client.metricsClient.List(context.Background(), metric.resourcePath, timespan, &interval, strings.Join(metrics, ","), "average,maximum", nil, "", "", insights.Data, metric.namespace)
	if err != nil {
		return nil, err
	}
	var metricData []MetricValue
	for _, v := range *resp.Value {
		for _, t := range *v.Timeseries {
			for _, mv := range *t.Data {
				var val MetricValue
				if mv.Minimum != nil {
					val.min = *mv.Minimum
				}
				if mv.Maximum != nil {
					val.max = *mv.Maximum
				}
				if mv.Average != nil {
					val.average = *mv.Average
				}
				if mv.Total != nil {
					val.total = *mv.Total
				}
				if mv.Count != nil {
					val.count = *mv.Count
				}
				metricData = append(metricData, val)
			}
		}
	}
	return metricData, nil
}
