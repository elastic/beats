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

package node

import (
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/munin"
)

// init registers the MetricSet with the central registry as soon as the program
// starts. The New function will be called later to instantiate an instance of
// the MetricSet for each host defined in the module's configuration. After the
// MetricSet has been created then Fetch will begin to be called periodically.
func init() {
	mb.Registry.MustAddMetricSet("munin", "node", New,
		mb.DefaultMetricSet(),
	)
}

// MetricSet holds any configuration or state information. It must implement
// the mb.MetricSet interface. And this is best achieved by embedding
// mb.BaseMetricSet because it implements all of the required mb.MetricSet
// interface methods except for Fetch.
type MetricSet struct {
	mb.BaseMetricSet
	serviceName string
	serviceType string
	timeout     time.Duration
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The munin node metricset is beta.")

	config := defaultConfig
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	return &MetricSet{
		BaseMetricSet: base,
		serviceName:   config.ServiceName,
		serviceType:   config.ServiceType,
		timeout:       base.Module().Config().Timeout,
	}, nil
}

// Fetch method implements the data gathering
func (m *MetricSet) Fetch(r mb.ReporterV2) {
	node, err := munin.Connect(m.Host(), m.timeout)
	if err != nil {
		r.Error(err)
		return
	}
	defer node.Close()

	items, err := node.List()
	if err != nil {
		r.Error(err)
		return
	}

	metrics, err := node.Fetch(items...)
	if err != nil {
		r.Error(err)
	}

	// Even if there was some error, keep sending succesfully collected metrics if any
	if len(metrics) == 0 {
		return
	}
	event := mb.Event{
		Service: m.serviceType,
		RootFields: common.MapStr{
			"munin": common.MapStr{
				"metrics": metrics,
			},
		},
	}
	if m.serviceName != "" {
		event.RootFields.Put("service.name", m.serviceName)
	}
	r.Event(event)
}
