package partition

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/mb/parse"

	"github.com/Shopify/sarama"
)

// init registers the partition MetricSet with the central registry.
func init() {
	if err := mb.Registry.AddMetricSet("kafka", "partition", New, parse.PassThruHostParser); err != nil {
		panic(err)
	}
}

// MetricSet type defines all fields of the partition MetricSet
type MetricSet struct {
	mb.BaseMetricSet
	client sarama.Client
}

// New creates a new instance of the partition MetricSet
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	logp.Warn("EXPERIMENTAL: The %v %v metricset is experimental", base.Module().Name(), base.Name())

	return &MetricSet{BaseMetricSet: base}, nil
}

// Fetch partition stats list from kafka
func (m *MetricSet) Fetch() ([]common.MapStr, error) {
	if m.client == nil {
		config := sarama.NewConfig()
		config.Net.DialTimeout = m.Module().Config().Timeout
		config.Net.ReadTimeout = m.Module().Config().Timeout
		config.ClientID = "metricbeat"

		client, err := sarama.NewClient([]string{m.Host()}, config)
		if err != nil {
			return nil, err
		}
		m.client = client
	}

	topics, err := m.client.Topics()
	if err != nil {
		return nil, err
	}

	events := []common.MapStr{}
	for _, topic := range topics {
		partitions, err := m.client.Partitions(topic)
		if err != nil {
			logp.Err("Fetch partition info for topic %s: %s", topic, err)
		}

		for _, partition := range partitions {
			newestOffset, err := m.client.GetOffset(topic, partition, sarama.OffsetNewest)
			if err != nil {
				logp.Err("Fetching newest offset information for partition %s in topic %s: %s", partition, topic, err)
			}

			oldestOffset, err := m.client.GetOffset(topic, partition, sarama.OffsetOldest)
			if err != nil {
				logp.Err("Fetching oldest offset information for partition %s in topic %s: %s", partition, topic, err)
			}

			broker, err := m.client.Leader(topic, partition)
			if err != nil {
				logp.Err("Fetching brocker for partition %s in topic %s: %s", partition, topic, err)
			}

			event := common.MapStr{
				"topic":     topic,
				"partition": partition,
				"offset": common.MapStr{
					"oldest": oldestOffset,
					"newest": newestOffset,
				},
				"broker": common.MapStr{
					"id":      broker.ID(),
					"address": broker.Addr(),
				},
			}

			events = append(events, event)
		}
	}

	return events, nil
}
