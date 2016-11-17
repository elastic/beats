package consumer

import (
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/kafka"

	"github.com/Shopify/sarama"
	"github.com/wvanbergen/kazoo-go"
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
	client sarama.Client
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
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right format
// It returns the event which is then forward to the output. In case of an error, a
// descriptive error must be returned.
func (m *MetricSet) Fetch() ([]common.MapStr, error) {

	var err error
	m.client, err = kafka.GetClient(m.client, m)
	if err != nil {
		return nil, err
	}

	// Create config for it
	hosts := []string{"kafka:2181"}

	zookeeperClient, err := kazoo.NewKazoo(hosts, nil)
	if err != nil {
		return nil, err
	}

	groups, err := zookeeperClient.Consumergroups()
	if err != nil {
		return nil, err
	}

	topics, err := m.client.Topics()
	if err != nil {
		return nil, err
	}

	events := []common.MapStr{}
	for _, group := range groups {
		broker, err := m.client.Coordinator(group.Name)
		if err != nil {
			logp.Err("Broker error: %s", err)
			continue
		}

		offsetRequest := &sarama.OffsetFetchRequest{
			ConsumerGroup: group.Name,
			Version:       0,
		}
		response, err := broker.FetchOffset(offsetRequest)
		for _, topic := range topics {
			partitions, err := m.client.Partitions(topic)
			if err != nil {
				logp.Err("Fetch partition info for topic %s: %s", topic, err)
			}

			for _, partition := range partitions {

				// Could we use group.FetchOffset() instead?
				offset := response.GetBlock(topic, partition)

				event := common.MapStr{
					"@timestamp": common.Time(time.Now()),
					"type":       "consumer",
					"partition":  partition,
					"topic":      topic,
					"group":      group.Name,
					"offset":     offset,
				}
				events = append(events, event)
			}
		}
	}

	return events, nil
}
