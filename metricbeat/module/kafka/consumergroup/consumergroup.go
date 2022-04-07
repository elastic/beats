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

package consumergroup

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/libbeat/logp"
	"github.com/elastic/beats/v8/metricbeat/mb"
	"github.com/elastic/beats/v8/metricbeat/module/kafka"
)

// init registers the MetricSet with the central registry.
func init() {
	mb.Registry.MustAddMetricSet("kafka", "consumergroup", New,
		mb.DefaultMetricSet(),
	)
}

// MetricSet type defines all fields of the MetricSet
type MetricSet struct {
	*kafka.MetricSet

	topics nameSet
	groups nameSet
}

type groupAssignment struct {
	clientID   string
	memberID   string
	clientHost string
}

var debugf = logp.MakeDebug("kafka")

// New creates a new instance of the MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	opts := kafka.MetricSetOptions{
		Version: "0.9.0.0",
	}

	ms, err := kafka.NewMetricSet(base, opts)
	if err != nil {
		return nil, err
	}

	config := struct {
		Groups []string `config:"groups"`
		Topics []string `config:"topics"`
	}{}
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	return &MetricSet{
		MetricSet: ms,
		groups:    makeNameSet(config.Groups...),
		topics:    makeNameSet(config.Topics...),
	}, nil
}

// Fetch consumer group metrics from kafka
func (m *MetricSet) Fetch(r mb.ReporterV2) error {
	broker, err := m.Connect()
	if err != nil {
		return errors.Wrap(err, "error in connect")
	}
	defer broker.Close()

	brokerInfo := common.MapStr{
		"id":      broker.ID(),
		"address": broker.AdvertisedAddr(),
	}

	emitEvent := func(event common.MapStr) {
		// Helpful IDs to avoid scripts on queries
		partitionTopicID := fmt.Sprintf("%d-%s", event["partition"], event["topic"])

		moduleFields := common.MapStr{
			"broker": brokerInfo,
			"topic": common.MapStr{
				"name": event["topic"],
			},
			"partition": common.MapStr{
				"id":       event["partition"],
				"topic_id": partitionTopicID,
			},
		}
		delete(event, "topic")
		delete(event, "partition")

		r.Event(mb.Event{
			ModuleFields:    moduleFields,
			MetricSetFields: event,
		})
	}
	err = fetchGroupInfo(emitEvent, broker, m.groups.pred(), m.topics.pred())
	if err != nil {
		return errors.Wrap(err, "error in fetch")
	}

	return nil
}
