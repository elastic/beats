package monitor

import (
	"context"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/services/preview/monitor/mgmt/2018-03-01/insights"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-03-01/resources"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/elastic/beats/x-pack/metricbeat/module/azure"
	"strings"
	"time"
)

// AzureMonitorClient represents local client which will use the azure sdk go metricsclient
type AzureMonitorClient struct {
	metricsClient               *insights.MetricsClient
	metricDefinitionClient *insights.MetricDefinitionsClient
	accessToken          string
	accessTokenExpiresOn time.Time
	resources []AzureMonitorResource
}

type AzureMonitorResource struct {
	name string
	uri string
	metricNamespace string
	metrics []AzureMonitorMetric
}

type AzureMonitorMetric struct {
	name string
	value int64
}

// New instantiates the an Azure monitoring client
func (client *AzureMonitorClient) New(config azure.Config) error{

	clientConfig := auth.NewClientCredentialsConfig(config.ClientId, config.ClientSecret, config.TenantId)
	authorizer, err := clientConfig.Authorizer()
	if err != nil {
		return err
	}
	metricsClient := insights.NewMetricsClient(config.SubscriptionId)
	metricsDefinitionClient := insights.NewMetricDefinitionsClient(config.SubscriptionId)
	metricsClient.Authorizer = authorizer
	metricsDefinitionClient.Authorizer= authorizer
	client.metricDefinitionClient= &metricsDefinitionClient
	client.metricsClient = &metricsClient


resourceClient := resources.NewClient(config.SubscriptionId)
resourceClient.Authorizer= authorizer
	for _, resource := range config.Resources{
    res:= GetResourceInfo(resourceClient, resource, config)
_= res
	}

	return nil
}


func GetResourceInfo(client resources.Client, name string, config azure.Config) AzureMonitorResource{
	var monitorResource AzureMonitorResource
	monitorResource.name= name
	test, err:= client.Get(context.Background(), "obs-infrastructure", "", "", "", name)
	if err!= nil{
		_= test
		monitorResource.uri= "dsfs"
	}
	return monitorResource
}

// ListMetricDefinitions returns the list of metrics available for the specified resource in the form "Localized Name (metric name)".
func (client *AzureMonitorClient)ListMetricDefinitions(resourceURI string) ([]string, error) {

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
func (client *AzureMonitorClient)GetMetricsData(resourceID string, metrics []string) ([]string, error) {

	endTime := time.Now().UTC()
	startTime := endTime.Add(time.Duration(-5) * time.Minute)
	timespan := fmt.Sprintf("%s/%s", startTime.Format(time.RFC3339), endTime.Format(time.RFC3339))

	resp, err := client.metricsClient.List(context.Background(), resourceID, timespan, nil, strings.Join(metrics, ","), "minimum,maximum", nil, "", "", insights.Data, "")
	if err != nil {
		return nil, err
	}
	metricData := []string{}
	for _, v := range *resp.Value {
		for _, t := range *v.Timeseries {
			for _, mv := range *t.Data {
				min := float64(0.0)
				max := float64(0.0)
				if mv.Minimum != nil {
					min = *mv.Minimum
				}
				if mv.Maximum != nil {
					max = *mv.Maximum
				}
				metricData = append(metricData, fmt.Sprintf("%s @ %s - min: %f, max: %f", *v.Name.LocalizedValue, *mv.TimeStamp, min, max))
			}
		}
	}
	return metricData, nil
}

