// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package partition

import (
	"fmt"

	"github.com/Shopify/sarama"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/mb/parse"
	"github.com/elastic/beats/metricbeat/module/kafka"
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
	*kafka.MetricSet

	topics []string
}

var errFailQueryOffset = errors.New("operation failed")

var debugf = logp.MakeDebug("kafka")

// New creates a new instance of the partition MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	opts := kafka.MetricSetOptions{
		Version: "0.8.2.0",
	}

	ms, err := kafka.NewMetricSet(base, opts)
	if err != nil {
		return nil, err
	}

	config := struct {
		Topics []string `config:"topics"`
	}{}
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	return &MetricSet{
		MetricSet: ms,
		topics:    config.Topics,
	}, nil
}

// Fetch partition stats list from kafka
func (m *MetricSet) Fetch(r mb.ReporterV2) error {
	broker, err := m.Connect()
	if err != nil {
		return errors.Wrap(err, "error in connect")
	}
	defer broker.Close()

	topics, err := broker.GetTopicsMetadata(m.topics...)
	if err != nil {
		return errors.Wrap(err, "error getting topic metadata")
	}
	if len(topics) == 0 {
		debugf("no topic could be read, check ACLs")
		return nil
	}

	evtBroker := common.MapStr{
		"id":      broker.ID(),
		"address": broker.AdvertisedAddr(),
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
			if broker.ID() != partition.Leader {
				debugf("broker is not leader (broker=%v, leader=%v)", broker.ID(), partition.Leader)
				continue
			}

			// collect offsets for all replicas
			for _, id := range partition.Replicas {

				// Get oldest and newest available offsets
				offOldest, offNewest, offOK, err := queryOffsetRange(broker, id, topic.Name, partition.ID)

				if !offOK {
					if err == nil {
						err = errFailQueryOffset
					}

					msg := fmt.Errorf("Failed to query kafka partition (%v:%v) offsets: %v",
						topic.Name, partition.ID, err)
					m.Logger().Warn(msg)
					r.Error(msg)
					continue
				}

				partitionEvent := common.MapStr{
					"id":             partition.ID,
					"leader":         partition.Leader,
					"replica":        id,
					"is_leader":      partition.Leader == id,
					"insync_replica": hasID(id, partition.Isr),
				}

				if partition.Err != 0 {
					partitionEvent["error"] = common.MapStr{
						"code": partition.Err,
					}
				}

				// Helpful IDs to avoid scripts on queries
				partitionTopicID := fmt.Sprintf("%d-%s", partition.ID, topic.Name)
				partitionTopicBrokerID := fmt.Sprintf("%s-%d", partitionTopicID, id)

				// create event
				event := common.MapStr{
					// Common `kafka.partition` fields
					"id":              partition.ID,
					"topic_id":        partitionTopicID,
					"topic_broker_id": partitionTopicBrokerID,

					"topic":     evtTopic,
					"broker":    evtBroker,
					"partition": partitionEvent,
					"offset": common.MapStr{
						"newest": offNewest,
						"oldest": offOldest,
					},
				}

				// TODO (deprecation): Remove fields from MetricSetFields moved to ModuleFields
				sent := r.Event(mb.Event{
					ModuleFields: common.MapStr{
						"broker": evtBroker,
						"topic":  evtTopic,
					},
					MetricSetFields: event,
				})
				if !sent {
					return nil
				}
			}
		}
	}
	return nil
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
		return -1, -1, false, errors.Wrap(err, "failed to get oldest offset")
	}

	newest, err := b.PartitionOffset(replicaID, topic, partition, sarama.OffsetNewest)
	if err != nil {
		return -1, -1, false, errors.Wrap(err, "failed to get newest offset")
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
