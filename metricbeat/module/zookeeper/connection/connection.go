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
	"github.com/pkg/errors"

	"github.com/menderesk/beats/v7/metricbeat/mb"
	"github.com/menderesk/beats/v7/metricbeat/mb/parse"
	"github.com/menderesk/beats/v7/metricbeat/module/zookeeper"
)

// init registers the MetricSet with the central registry as soon as the program
// starts. The New function will be called later to instantiate an instance of
// the MetricSet for each host defined in the module's configuration. After the
// MetricSet has been created then Fetch will begin to be called periodically.
func init() {
	mb.Registry.MustAddMetricSet("zookeeper", "connection", New,
		mb.WithHostParser(parse.PassThruHostParser),
	)
}

// MetricSet holds any configuration or state information. It must implement
// the mb.MetricSet interface. And this is best achieved by embedding
// mb.BaseMetricSet because it implements all of the required mb.MetricSet
// interface methods except for Fetch.
type MetricSet struct {
	mb.BaseMetricSet
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	config := struct{}{}
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	return &MetricSet{
		BaseMetricSet: base,
	}, nil
}

// Fetch fetches metrics from ZooKeeper by making a tcp connection to the
// command port and sending the "cons" command and parsing the output.
func (m *MetricSet) Fetch(reporter mb.ReporterV2) error {
	outputReader, err := zookeeper.RunCommand("cons", m.Host(), m.Module().Config().Timeout)
	if err != nil {
		return errors.Wrap(err, "'cons' command failed")
	}

	events, err := m.parseCons(outputReader)
	if err != nil {
		return errors.Wrap(err, "error parsing response from zookeeper")
	}

	serverID, err := zookeeper.ServerID(m.Host(), m.Module().Config().Timeout)
	if err != nil {
		return errors.Wrap(err, "error obtaining server id")
	}

	for _, event := range events {
		event.RootFields.Put("service.node.name", serverID)
		reporter.Event(event)
	}

	return nil
}
