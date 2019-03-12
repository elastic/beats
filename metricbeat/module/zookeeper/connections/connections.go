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

package connections

import (
	"github.com/elastic/beats/metricbeat/mb/parse"
	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/logp"

	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/zookeeper"
)

var logger = logp.NewLogger("zookeeper.connections")

// init registers the MetricSet with the central registry as soon as the program
// starts. The New function will be called later to instantiate an instance of
// the MetricSet for each host defined in the module's configuration. After the
// MetricSet has been created then Fetch will begin to be called periodically.
func init() {
	mb.Registry.MustAddMetricSet("zookeeper", "connections", New,
		mb.WithHostParser(parse.PassThruHostParser),
		mb.DefaultMetricSet(),
	)
}

// MetricSet holds any configuration or state information. It must implement
// the mb.MetricSet interface. And this is best achieved by embedding
// mb.BaseMetricSet because it implements all of the required mb.MetricSet
// interface methods except for Fetch.
type MetricSet struct {
	mb.BaseMetricSet
	counter int
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The zookeeper connections metricset is beta.")

	config := struct{}{}
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	return &MetricSet{
		BaseMetricSet: base,
		counter:       1,
	}, nil
}

// Fetch fetches metrics from ZooKeeper by making a tcp connection to the
// command port and sending the "cons" command and parsing the output.
func (m *MetricSet) Fetch(reporter mb.ReporterV2) {
	outputReader, err := zookeeper.RunCommand("cons", m.Host(), m.Module().Config().Timeout)
	if err != nil {
		reporter.Error(errors.Wrap(err, "'cons' command failed"))
		return
	}

	events, err := parseCons(outputReader)
	if err != nil {
		reporter.Error(err)
		return
	}

	for _, event := range events {
		reporter.Event(mb.Event{
			MetricSetFields: event,
		})
	}
}
