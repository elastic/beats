package consumer

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"
)

// init registers the MetricSet with the central registry.
// The New method will be called after the setup of the module and before starting to fetch data
func init() {
	if err := mb.Registry.AddMetricSet("kafka", "consumer", New); err != nil {
		panic(err)
	}
}

// MetricSet type defines all fields of the MetricSet
// As a minimum it must inherit the mb.BaseMetricSet fields, but can be extended with
// additional entries. These variables can be used to persist data or configuration between
// multiple fetch calls.
type MetricSet struct {
	mb.BaseMetricSet
	counter int
}

// New create a new instance of the MetricSet
// Part of new is also setting up the configuration by processing additional
// configuration entries if needed.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {

	config := struct{}{}

	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	return &MetricSet{
		BaseMetricSet: base,
		counter:       1,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right format
// It returns the event which is then forward to the output. In case of an error, a
// descriptive error must be returned.
func (m *MetricSet) Fetch() ([]common.MapStr, error) {

	/*
		Returns a list of consumer with the following fields


		- partition id
		- topic
		- group
		- offset
		- size-offset (optional)

		Do we need to open a consumer to get the data?
		Connection to zookeeper need to fetch list of consumers?

		https://github.com/linkedin/Burrow/blob/master/protocol/protocol.go#L70
	*/

	/*var err error
	config := sarama.NewConfig()
	client, err := sarama.NewClient([]string{"localhost:9092"}, config)*/

	/*logp.Debug("test", "TOPICS: %+v", topics)

	if err != nil {

			logp.Err("ERRR: %s", err)
	}*/

	events := []common.MapStr{}
	return events, nil

}
