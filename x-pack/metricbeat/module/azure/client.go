package azure

import (
	"fmt"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-03-01/resources"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/pkg/errors"
)

// Client represents the azure client which will make use of the azure sdk go metrics related clients
type Client struct {
	AzureMonitorService AzureService
	Config              Config
	Resources           ResourceConfiguration
	Log                 *logp.Logger
}

// NewClient instantiates the an Azure monitoring client
func NewClient(config Config) (*Client, error) {
	azureMonitorService, err := NewAzureService(config.ClientID, config.ClientSecret, config.TenantID, config.SubscriptionID)
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
			current, err := mapMetricValues(resp, client.Resources.Metrics[i].Values, endTime.Truncate(time.Minute).Add(client.Config.Period * (-1)), endTime.Truncate(time.Minute) )
			if err != nil {
				client.LogError(report, err)
			}
			client.Resources.Metrics[i].Values = current
		}
	}
	return nil
}

// logError is used to reduce the number of lines written when logging errors
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
