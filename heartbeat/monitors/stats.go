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

type statsRecorder interface {
	startMonitor(endpoints int64)
	stopMonitor(endpoints int64)
}

// multiStatsRecorder composes multiple statsRecorders.
type multiStatsRecorder struct {
	recorders []statsRecorder
}

func (msr multiStatsRecorder) startMonitor(endpoints int64) {
	for _, recorder := range msr.recorders {
		recorder.startMonitor(endpoints)
	}
}

func (msr multiStatsRecorder) stopMonitor(endpoints int64) {
	for _, recorder := range msr.recorders {
		recorder.stopMonitor(endpoints)
	}
}

// globalStats recorder is for recording to the shared global monitorStarts counter.
type globalMonitorsRecorder struct {
	// globalMonitors is a reference to the global count of all monitoring plugins.
	// A pointer to it is stored here for convenience
	globalMonitors *monitoring.Int
}

func (gsr globalMonitorsRecorder) startMonitor(endpoints int64) {
	gsr.globalMonitors.Inc()
}

func (gsr globalMonitorsRecorder) stopMonitor(endpoints int64) {
	gsr.globalMonitors.Dec()
}

// singleStats is used to record start/stop events for a single monitor/plugin
// to a single registry.
type pluginStatsRecorder struct {
	monitorStarts  *monitoring.Int
	monitorStops   *monitoring.Int
	endpointStarts *monitoring.Int
	endpointStops  *monitoring.Int
}

func newPluginStatsRecorder(pluginName string, rootRegistry *monitoring.Registry) statsRecorder {
	pluginRegistry := rootRegistry.NewRegistry(pluginName)
	return pluginStatsRecorder{
		monitoring.NewInt(pluginRegistry, "monitor_starts"),
		monitoring.NewInt(pluginRegistry, "monitor_stops"),
		monitoring.NewInt(pluginRegistry, "endpoint_starts"),
		monitoring.NewInt(pluginRegistry, "endpoint_stops"),
	}
}

func (ssr pluginStatsRecorder) startMonitor(endpoints int64) {
	ssr.monitorStarts.Inc()
	ssr.endpointStarts.Add(endpoints)
}

func (ssr pluginStatsRecorder) stopMonitor(endpoints int64) {
	ssr.monitorStops.Inc()
	ssr.endpointStops.Add(endpoints)
}
