package broker

import (
	"github.com/Shopify/sarama"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"
	metrics "github.com/rcrowley/go-metrics"
)

// init adds broker metricset
func init() {
	if err := mb.Registry.AddMetricSet("kafka", "broker", New); err != nil {
		panic(err)
	}
}

// MetricSet type defines broker metricset
type MetricSet struct {
	mb.BaseMetricSet
}

// New creates new broker metricset
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {

	config := struct{}{}

	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	return &MetricSet{
		BaseMetricSet: base,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right format
// It returns the event which is then forward to the output. In case of an error, a
// descriptive error must be returned.
func (m *MetricSet) Fetch() (common.MapStr, error) {

	config := sarama.NewConfig()
	client, err := sarama.NewClient([]string{"localhost:9092"}, config)
	if err != nil {
		return nil, err
	}
	config.MetricRegistry.RunHealthchecks()

	client.RefreshMetadata()
	incomingByteRate := metrics.GetOrRegisterMeter("incoming-byte-rate", config.MetricRegistry)
	requestRate := metrics.GetOrRegisterMeter("request-rate", config.MetricRegistry)

	// One event for each broker
	// Question: Should overall data also be sent?
	event := common.MapStr{
		"incomingByteRate": incomingByteRate,
		"requestRate":      requestRate,
	}

	/*
		Broker for a group can be fetched through client.Coordinator(group)

		Broker metrics to be reported:
		+------------------------------------------------+------------+---------------------------------------------------------------+
		| Name                                           | Type       | Description                                                   |
		+------------------------------------------------+------------+---------------------------------------------------------------+
		| incoming-byte-rate                             | meter      | Bytes/second read off all brokers                             |
		| incoming-byte-rate-for-broker-<broker-id>      | meter      | Bytes/second read off a given broker                          |
		| outgoing-byte-rate                             | meter      | Bytes/second written off all brokers                          |
		| outgoing-byte-rate-for-broker-<broker-id>      | meter      | Bytes/second written off a given broker                       |
		| request-rate                                   | meter      | Requests/second sent to all brokers                           |
		| request-rate-for-broker-<broker-id>            | meter      | Requests/second sent to a given broker                        |
		| histogram request-size                         | histogram  | Distribution of the request size in bytes for all brokers     |
		| histogram request-size-for-broker-<broker-id>  | histogram  | Distribution of the request size in bytes for a given broker  |
		| response-rate                                  | meter      | Responses/second received from all brokers                    |
		| response-rate-for-broker-<broker-id>           | meter      | Responses/second received from a given broker                 |
		| histogram response-size                        | histogram  | Distribution of the response size in bytes for all brokers    |
		| histogram response-size-for-broker-<broker-id> | histogram  | Distribution of the response size in bytes for a given broker |
		+------------------------------------------------+------------+---------------------------------------------------------------+
	*/

	return event, nil
}
