package partition

import (
	"crypto/tls"
	"errors"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/mb/parse"
	"github.com/elastic/beats/metricbeat/module/kafka"

	"github.com/Shopify/sarama"
)

// init registers the partition MetricSet with the central registry.
func init() {
	mb.Registry.MustAddMetricSet("kafka", "partition", New,
		mb.WithHostParser(parse.PassThruHostParser),
		mb.DefaultMetricSet(),
	)
}

// MetricSet type defines all fields of the partition MetricSet
type MetricSet struct {
	mb.BaseMetricSet

	broker *kafka.Broker
	topics []string
}

const noID int32 = -1

var errFailQueryOffset = errors.New("operation failed")

var debugf = logp.MakeDebug("kafka")

// New creates a new instance of the partition MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The kafka partition metricset is beta")

	config := defaultConfig
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	var tls *tls.Config
	tlsCfg, err := outputs.LoadTLSConfig(config.TLS)
	if err != nil {
		return nil, err
	}
	if tlsCfg != nil {
		tls = tlsCfg.BuildModuleConfig("")
	}

	timeout := base.Module().Config().Timeout
	cfg := kafka.BrokerSettings{
		MatchID:     true,
		DialTimeout: timeout,
		ReadTimeout: timeout,
		ClientID:    config.ClientID,
		Retries:     config.Retries,
		Backoff:     config.Backoff,
		TLS:         tls,
		Username:    config.Username,
		Password:    config.Password,
	}

	return &MetricSet{
		BaseMetricSet: base,
		broker:        kafka.NewBroker(base.Host(), cfg),
		topics:        config.Topics,
	}, nil
}

func (m *MetricSet) connect() (*kafka.Broker, error) {
	err := m.broker.Connect()
	return m.broker, err
}

// Fetch partition stats list from kafka
func (m *MetricSet) Fetch() ([]common.MapStr, error) {
	b, err := m.connect()
	if err != nil {
		return nil, err
	}

	defer b.Close()
	topics, err := b.GetTopicsMetadata(m.topics...)
	if err != nil {
		return nil, err
	}

	events := []common.MapStr{}
	evtBroker := common.MapStr{
		"id":      b.ID(),
		"address": b.Addr(),
	}

	for _, topic := range topics {
		debugf("fetch events for topic: ", topic.Name)
		evtTopic := common.MapStr{
			"name": topic.Name,
		}

		if topic.Err != 0 {
			evtTopic["error"] = common.MapStr{
				"code": topic.Err,
			}
		}

		for _, partition := range topic.Partitions {
			// partition offsets can be queried from leader only
			if b.ID() != partition.Leader {
				debugf("broker is not leader (broker=%v, leader=%v)", b.ID(), partition.Leader)
				continue
			}

			// collect offsets for all replicas
			for _, id := range partition.Replicas {

				// Get oldest and newest available offsets
				offOldest, offNewest, offOK, err := queryOffsetRange(b, id, topic.Name, partition.ID)

				if !offOK {
					if err == nil {
						err = errFailQueryOffset
					}

					logp.Err("Failed to query kafka partition (%v:%v) offsets: %v",
						topic.Name, partition.ID, err)
					continue
				}

				partitionEvent := common.MapStr{
					"id":             partition.ID,
					"leader":         partition.Leader,
					"replica":        id,
					"insync_replica": hasID(id, partition.Isr),
				}

				if partition.Err != 0 {
					partitionEvent["error"] = common.MapStr{
						"code": partition.Err,
					}
				}

				// create event
				event := common.MapStr{
					"topic":     evtTopic,
					"broker":    evtBroker,
					"partition": partitionEvent,
					"offset": common.MapStr{
						"newest": offNewest,
						"oldest": offOldest,
					},
				}

				events = append(events, event)
			}
		}
	}

	return events, nil
}

// queryOffsetRange queries the broker for the oldest and the newest offsets in
// a kafka topics partition for a given replica.
func queryOffsetRange(
	b *kafka.Broker,
	replicaID int32,
	topic string,
	partition int32,
) (int64, int64, bool, error) {
	oldest, err := b.PartitionOffset(replicaID, topic, partition, sarama.OffsetOldest)
	if err != nil {
		return -1, -1, false, err
	}

	newest, err := b.PartitionOffset(replicaID, topic, partition, sarama.OffsetNewest)
	if err != nil {
		return -1, -1, false, err
	}

	okOld := oldest != -1
	okNew := newest != -1
	return oldest, newest, okOld && okNew, nil
}

func hasID(id int32, lst []int32) bool {
	for _, other := range lst {
		if id == other {
			return true
		}
	}
	return false
}
