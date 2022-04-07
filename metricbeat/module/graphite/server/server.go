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

package server

import (
	"github.com/pkg/errors"

	"github.com/elastic/beats/v8/libbeat/logp"
	serverhelper "github.com/elastic/beats/v8/metricbeat/helper/server"
	"github.com/elastic/beats/v8/metricbeat/helper/server/tcp"
	"github.com/elastic/beats/v8/metricbeat/helper/server/udp"
	"github.com/elastic/beats/v8/metricbeat/mb"
)

// init registers the MetricSet with the central registry.
// The New method will be called after the setup of the module and before starting to fetch data
func init() {
	mb.Registry.MustAddMetricSet("graphite", "server", New,
		mb.DefaultMetricSet(),
	)
}

// MetricSet type defines all fields of the MetricSet
// As a minimum it must inherit the mb.BaseMetricSet fields, but can be extended with
// additional entries. These variables can be used to persist data or configuration between
// multiple fetch calls.
type MetricSet struct {
	mb.BaseMetricSet
	server    serverhelper.Server
	processor *metricProcessor
}

// New create a new instance of the MetricSet
// Part of new is also setting up the configuration by processing additional
// configuration entries if needed.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {

	config := DefaultGraphiteCollectorConfig()
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	var s serverhelper.Server
	var err error
	if config.Protocol == "tcp" {
		s, err = tcp.NewTcpServer(base)
	} else {
		s, err = udp.NewUdpServer(base)
	}

	if err != nil {
		return nil, err
	}

	processor := NewMetricProcessor(config.Templates, config.DefaultTemplate)

	return &MetricSet{
		BaseMetricSet: base,
		server:        s,
		processor:     processor,
	}, nil
}

// Run method provides the Graphite server with a reporter with which events can be reported.
func (m *MetricSet) Run(reporter mb.PushReporter) {
	// Start event watcher
	if err := m.server.Start(); err != nil {
		err = errors.Wrap(err, "failed to start graphite server")
		logp.Err("%v", err)
		reporter.Error(err)
		return
	}

	for {
		select {
		case <-reporter.Done():
			m.server.Stop()
			return
		case msg := <-m.server.GetEvents():
			input := msg.GetEvent()
			bytesRaw, ok := input[serverhelper.EventDataKey]
			if ok {
				bytes, ok := bytesRaw.([]byte)
				if ok && len(bytes) != 0 {
					event, err := m.processor.Process(string(bytes))
					if err != nil {
						reporter.Error(err)
					} else {
						reporter.Event(event)
					}
				}
			}

		}
	}
}
