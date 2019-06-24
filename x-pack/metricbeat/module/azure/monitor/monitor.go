package monitor

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/x-pack/metricbeat/module/azure"
	"github.com/pkg/errors"
)

// init registers the MetricSet with the central registry as soon as the program
// starts. The New function will be called later to instantiate an instance of
// the MetricSet for each host defined in the module's configuration. After the
// MetricSet has been created then Fetch will begin to be called periodically.
func init() {
	mb.Registry.MustAddMetricSet("azure", "monitor", New, mb.DefaultMetricSet())
}

// MetricSet holds any configuration or state information. It must implement
// the mb.MetricSet interface. And this is best achieved by embedding
// mb.BaseMetricSet because it implements all of the required mb.MetricSet
// interface methods except for Fetch.
type MetricSet struct {
	mb.BaseMetricSet
	client *AzureMonitorClient
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The azure monitor metricset is beta.")
	var config azure.Config
	err := base.Module().UnpackConfig(&config)
	if err != nil {
		return nil, errors.Wrap(err, "error unpack raw module config using UnpackConfig")
	}
	if len(config.Metrics) == 0 {
		return nil, errors.New("no metrics defined")
	}
	var monitorClient AzureMonitorClient
	monitorClient.New(config)
	monitorClient.InitResources()

	return &MetricSet{
		BaseMetricSet: base,
		client:        &monitorClient,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(report mb.ReporterV2) error {
	for _, metric := range m.client.metrics {
		values, err := m.client.GetMetricsData(metric)
		if err != nil {
			return nil
		}
		report.Event(mb.Event{
			MetricSetFields: common.MapStr{
				"resource": common.MapStr{
					"id": metric.resourcePath,
					"metric": common.MapStr{
						"name":  metric.name,
						"value": values[0].total,
					},
				},
			},
		})
	}
	return nil
}
