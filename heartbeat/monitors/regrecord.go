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

package monitors

import (
	"github.com/elastic/beats/libbeat/monitoring"
)

type registryRecorder interface {
	startMonitor(endpoints int64)
	stopMonitor(endpoints int64)
}

// multiRegistryRecorder composes multiple statsRecorders.
type multiRegistryRecorder struct {
	recorders []registryRecorder
}

func (mr multiRegistryRecorder) startMonitor(endpoints int64) {
	for _, recorder := range mr.recorders {
		recorder.startMonitor(endpoints)
	}
}

func (mr multiRegistryRecorder) stopMonitor(endpoints int64) {
	for _, recorder := range mr.recorders {
		recorder.stopMonitor(endpoints)
	}
}

// countersRecorder is used to record start/stop events for a single monitor/plugin
// to a single registry as counters.
type countersRecorder struct {
	monitorStarts  *monitoring.Int
	monitorStops   *monitoring.Int
	endpointStarts *monitoring.Int
	endpointStops  *monitoring.Int
}

func newPluginCountersRecorder(pluginName string, rootRegistry *monitoring.Registry) registryRecorder {
	pluginRegistry := rootRegistry.NewRegistry(pluginName)
	return countersRecorder{
		monitoring.NewInt(pluginRegistry, "monitor_starts"),
		monitoring.NewInt(pluginRegistry, "monitor_stops"),
		monitoring.NewInt(pluginRegistry, "endpoint_starts"),
		monitoring.NewInt(pluginRegistry, "endpoint_stops"),
	}
}

func (r countersRecorder) startMonitor(endpoints int64) {
	r.monitorStarts.Inc()
	r.endpointStarts.Add(endpoints)
}

func (r countersRecorder) stopMonitor(endpoints int64) {
	r.monitorStops.Inc()
	r.endpointStops.Add(endpoints)
}

// countersRecorder is used to record start/stop events for a single monitor/plugin
// to a single registry as gauges.
type gaugeRecorder struct {
	monitors  *monitoring.Int
	endpoints *monitoring.Int
}

func newRootGaugeRecorder(r *monitoring.Registry) registryRecorder {
	return gaugeRecorder{
		monitoring.NewInt(r, "monitors"),
		monitoring.NewInt(r, "endpoints"),
	}
}

func newPluginGaugeRecorder(pluginName string, rootRegistry *monitoring.Registry) registryRecorder {
	pluginRegistry := rootRegistry.NewRegistry(pluginName)
	return newRootGaugeRecorder(pluginRegistry)
}

func (r gaugeRecorder) startMonitor(endpoints int64) {
	r.monitors.Inc()
	r.endpoints.Add(endpoints)
}

func (r gaugeRecorder) stopMonitor(endpoints int64) {
	r.monitors.Dec()
	r.endpoints.Sub(endpoints)
}
