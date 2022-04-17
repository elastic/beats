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

package kafka

import (
	"crypto/tls"

	"github.com/menderesk/beats/v7/libbeat/common/transport/tlscommon"
	"github.com/menderesk/beats/v7/metricbeat/mb"
)

// MetricSet is the base metricset for all Kafka metricsets
type MetricSet struct {
	mb.BaseMetricSet
	broker *Broker
}

// MetricSetOptions are the options of a Kafka metricset
type MetricSetOptions struct {
	Version string
}

// NewMetricSet creates a base metricset for Kafka metricsets
func NewMetricSet(base mb.BaseMetricSet, options MetricSetOptions) (*MetricSet, error) {
	config := defaultConfig
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	tlsCfg, err := tlscommon.LoadTLSConfig(config.TLS)
	if err != nil {
		return nil, err
	}

	var tls *tls.Config
	if tlsCfg != nil {
		tls = tlsCfg.BuildModuleClientConfig("")
	}

	timeout := base.Module().Config().Timeout
	cfg := BrokerSettings{
		MatchID:     true,
		DialTimeout: timeout,
		ReadTimeout: timeout,
		ClientID:    config.ClientID,
		Retries:     config.Retries,
		Backoff:     config.Backoff,
		TLS:         tls,
		Username:    config.Username,
		Password:    config.Password,
		Version:     Version(options.Version),
		Sasl:        config.Sasl,
	}

	return &MetricSet{
		BaseMetricSet: base,
		broker:        NewBroker(base.Host(), cfg),
	}, nil

}

// Connect connects with a kafka broker
func (m *MetricSet) Connect() (*Broker, error) {
	err := m.broker.Connect()
	return m.broker, err
}
