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

package plugin

import (
	"github.com/elastic/beats/v7/libbeat/monitoring"
)

type StartStopRegistryRecorder interface {
	StartMonitor(endpoints int64)
	StopMonitor(endpoints int64)
}

type MultiRegistry struct {
	startStopRecorders []StartStopRegistryRecorder
	durationRecorders  []DurationRegistryRecorder
}

// MultiRegistryRecorder composes multiple statsRecorders.
type MultiRegistryRecorder interface {
	StartMonitor(endpoints int64)
	StopMonitor(endpoints int64)
	RecordDuration(duration int64)
}

func (mr MultiRegistry) StartMonitor(endpoints int64) {
	for _, recorder := range mr.startStopRecorders {
		recorder.StartMonitor(endpoints)
	}
}

func (mr MultiRegistry) StopMonitor(endpoints int64) {
	for _, recorder := range mr.startStopRecorders {
		recorder.StopMonitor(endpoints)
	}
}

func (mr MultiRegistry) RecordDuration(duration int64) {
	for _, durationRecorder := range mr.durationRecorders {
		durationRecorder.RecordDuration(duration)
	}
}

// countersRecorder is used to record start/stop events for a single monitor/plugin
// to a single registry as counters.
type CountersRecorder struct {
	monitorStarts  *monitoring.Int
	monitorStops   *monitoring.Int
	endpointStarts *monitoring.Int
	endpointStops  *monitoring.Int
}

func NewPluginCountersRecorder(pluginName string, rootRegistry *monitoring.Registry) StartStopRegistryRecorder {
	pluginRegistry := rootRegistry.NewRegistry(pluginName)
	return CountersRecorder{
		monitoring.NewInt(pluginRegistry, "monitor_starts"),
		monitoring.NewInt(pluginRegistry, "monitor_stops"),
		monitoring.NewInt(pluginRegistry, "endpoint_starts"),
		monitoring.NewInt(pluginRegistry, "endpoint_stops"),
	}
}

func (r CountersRecorder) StartMonitor(endpoints int64) {
	r.monitorStarts.Inc()
	r.endpointStarts.Add(endpoints)
}

func (r CountersRecorder) StopMonitor(endpoints int64) {
	r.monitorStops.Inc()
	r.endpointStops.Add(endpoints)
}

// countersRecorder is used to record start/stop events for a single monitor/plugin
// to a single registry as gauges.
type gaugeRecorder struct {
	monitors  *monitoring.Int
	endpoints *monitoring.Int
}

func newRootGaugeRecorder(r *monitoring.Registry) StartStopRegistryRecorder {
	return gaugeRecorder{
		monitoring.NewInt(r, "monitors"),
		monitoring.NewInt(r, "endpoints"),
	}
}

func newPluginGaugeRecorder(pluginName string, rootRegistry *monitoring.Registry) StartStopRegistryRecorder {
	pluginRegistry := rootRegistry.NewRegistry(pluginName)
	return newRootGaugeRecorder(pluginRegistry)
}

func (r gaugeRecorder) StartMonitor(endpoints int64) {
	r.monitors.Inc()
	r.endpoints.Add(endpoints)
}

func (r gaugeRecorder) StopMonitor(endpoints int64) {
	r.monitors.Dec()
	r.endpoints.Sub(endpoints)
}
