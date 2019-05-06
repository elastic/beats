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

package connection

import (
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/rabbitmq"
)

func init() {
	mb.Registry.MustAddMetricSet("rabbitmq", "connection", New,
		mb.WithHostParser(rabbitmq.HostParser),
		mb.DefaultMetricSet(),
	)
}

// MetricSet for fetching RabbitMQ connections.
type MetricSet struct {
	*rabbitmq.MetricSet
}

// New creates new instance of MetricSet
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	ms, err := rabbitmq.NewMetricSet(base, rabbitmq.ConnectionsPath)
	if err != nil {
		return nil, err
	}
	return &MetricSet{ms}, nil
}

// Fetch makes an HTTP request to fetch connections metrics from the connections endpoint.
func (m *MetricSet) Fetch(r mb.ReporterV2) {
	content, err := m.HTTP.FetchContent()

	if err != nil {
		r.Error(err)
		return
	}

	eventsMapping(content, r)
}
