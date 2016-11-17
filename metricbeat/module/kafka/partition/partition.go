package partition

import (
	"errors"
	"fmt"

	"github.com/elastic/beats/libbeat/common"
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

	broker *sarama.Broker
	cfg    *sarama.Config
	id     int32
}

var noID int32 = -1

var errFailQueryOffset = errors.New("Failed to query offset for")

// New create a new instance of the partition MetricSet
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	config := struct{}{}
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	cfg := sarama.NewConfig()
	cfg.Net.DialTimeout = base.Module().Config().Timeout
	cfg.Net.ReadTimeout = base.Module().Config().Timeout
	cfg.ClientID = "metricbeat"

	broker := sarama.NewBroker(base.Host())
	return &MetricSet{
		BaseMetricSet: base,
		broker:        broker,
		cfg:           cfg,
		id:            noID,
	}, nil
}

func (m *MetricSet) connect() (*sarama.Broker, error) {
	b := m.broker
	if err := b.Open(m.cfg); err != nil {
		return nil, err
	}

	if m.id != noID {
		return b, nil
	}

	// current broker is bootstrap only. Get metadata to find id:
	meta, err := b.GetMetadata(&sarama.MetadataRequest{})
	if err != nil {
		return nil, err
	}

	addr := b.Addr()
	for _, other := range meta.Brokers {
		if other.Addr() == addr {
			m.id = other.ID()
			break
		}
	}

	if m.id == noID {
		b.Close()
		err = fmt.Errorf("No advertised broker with address %v found", addr)
		return nil, err
	}

	return b, nil
}

// Fetch partition stats list from kafka
func (m *MetricSet) Fetch() ([]common.MapStr, error) {
	b, err := m.connect()
	if err != nil {
		return nil, err
	}

	defer b.Close()
	response, err := b.GetMetadata(&sarama.MetadataRequest{})
	if err != nil {
		return nil, err
	}

	events := []common.MapStr{}
	evtBroker := common.MapStr{
		"id":      m.id,
		"address": b.Addr(),
	}

	for _, topic := range response.Topics {
		evtTopic := common.MapStr{
			"name": topic.Name,
		}
		if topic.Err != 0 {
			evtTopic["error"] = topic.Err
		}

		for _, partition := range topic.Partitions {
			// partition offsets can be queried from leader only
			if m.id != partition.Leader {
				continue
			}

			// collect offsets for all replicas
			for _, id := range partition.Replicas {

				// Get oldest and newest available offsets
				offOldest, offNewest, offOK, err := queryOffsetRange(b, id, topic.Name, partition.ID)

				var offsets common.MapStr
				if offOK {
					offsets = common.MapStr{
						"newest": offNewest,
						"oldest": offOldest,
					}
				} else {
					if err == nil {
						err = errFailQueryOffset
					}
					offsets = common.MapStr{
						"error": err,
					}
				}

				// create event
				event := common.MapStr{
					"topic":  evtTopic,
					"broker": evtBroker,
					"partition": common.MapStr{
						"id":             partition.ID,
						"error":          partition.Err,
						"leader":         partition.Leader,
						"replica":        id,
						"insync_replica": hasID(id, partition.Isr),
					},
					"offset": offsets,
				}

				events = append(events, event)

			}
		}
	}

	return events, nil
}

func hasID(id int32, lst []int32) bool {
	for _, other := range lst {
		if id == other {
			return true
		}
	}
	return false
}

func queryOffsetRange(
	b *sarama.Broker,
	replicaID int32,
	topic string,
	partition int32,
) (int64, int64, bool, error) {
	oldest, okOld, err := queryOffset(b, replicaID, topic, partition, sarama.OffsetOldest)
	if err != nil {
		return -1, -1, false, err
	}

	newest, okNew, err := queryOffset(b, replicaID, topic, partition, sarama.OffsetNewest)
	if err != nil {
		return -1, -1, false, err
	}

	return oldest, newest, okOld && okNew, nil
}

func queryOffset(
	b *sarama.Broker,
	replicaID int32,
	topic string,
	partition int32,
	time int64,
) (int64, bool, error) {
	req := &sarama.OffsetRequest{}
	if replicaID != noID {
		req.SetReplicaID(replicaID)
	}
	req.AddBlock(topic, partition, time, 1)
	resp, err := b.GetAvailableOffsets(req)
	if err != nil {
		return -1, false, err
	}

	block := resp.GetBlock(topic, partition)
	if len(block.Offsets) == 0 {
		return -1, false, nil
	}

	return block.Offsets[0], true, nil
}
