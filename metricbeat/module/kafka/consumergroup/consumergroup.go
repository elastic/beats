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
	"crypto/tls"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/libbeat/common/transport/tlscommon"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/kafka"
)

// init registers the MetricSet with the central registry.
func init() {
	mb.Registry.MustAddMetricSet("kafka", "consumergroup", New,
		mb.DefaultMetricSet(),
	)
}

// MetricSet type defines all fields of the MetricSet
type MetricSet struct {
	mb.BaseMetricSet

	broker *kafka.Broker
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
	cfgwarn.Beta("The kafka consumergroup metricset is beta")

	config := defaultConfig
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	var tls *tls.Config
	tlsCfg, err := tlscommon.LoadTLSConfig(config.TLS)
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

		// consumer groups API requires at least 0.9.0.0
		Version: kafka.Version{String: "0.9.0.0"},
	}

	return &MetricSet{
		BaseMetricSet: base,
		broker:        kafka.NewBroker(base.Host(), cfg),
		groups:        makeNameSet(config.Groups...),
		topics:        makeNameSet(config.Topics...),
	}, nil
}

func (m *MetricSet) Fetch() ([]common.MapStr, error) {
	if err := m.broker.Connect(); err != nil {
		logp.Err("broker connect failed: %v", err)
		return nil, err
	}

	b := m.broker
	defer b.Close()

	brokerInfo := common.MapStr{
		"id":      b.ID(),
		"address": b.AdvertisedAddr(),
	}

	var events []common.MapStr
	emitEvent := func(event common.MapStr) {
		event["broker"] = brokerInfo
		events = append(events, event)
	}
	err := fetchGroupInfo(emitEvent, b, m.groups.pred(), m.topics.pred())
	return events, err
}
