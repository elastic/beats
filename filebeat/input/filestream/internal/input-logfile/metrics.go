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

package input_logfile

import (
	"github.com/rcrowley/go-metrics"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/elastic-agent-libs/monitoring"
	"github.com/elastic/elastic-agent-libs/monitoring/adapter"
)

// Metrics defines a set of metrics for the filestream input.
type Metrics struct {
	unregister func()

	FilesOpened       *monitoring.Uint // Number of files that have been opened.
	FilesClosed       *monitoring.Uint // Number of files closed.
	FilesActive       *monitoring.Uint // Number of files currently open (gauge).
	MessagesRead      *monitoring.Uint // Number of messages read.
	MessagesTruncated *monitoring.Uint // Number of messages truncated.
	BytesProcessed    *monitoring.Uint // Number of bytes processed.
	EventsProcessed   *monitoring.Uint // Number of events processed.
	ProcessingErrors  *monitoring.Uint // Number of processing errors.
	ProcessingTime    metrics.Sample   // Histogram of the elapsed time for processing an event.

	// Those metrics use the same registry/keys as the log input uses
	HarvesterStarted   *monitoring.Int
	HarvesterClosed    *monitoring.Int
	HarvesterRunning   *monitoring.Int
	HarvesterOpenFiles *monitoring.Int
}

func (m *Metrics) Close() {
	if m == nil {
		return
	}

	m.unregister()
}

func NewMetrics(ctx v2.Context) *Metrics {
	// The log input creates the `filebeat.harvester` registry as a package
	// variable, so it should always exist before this function runs.
	// However at least on testing scenarios this does not hold true, so
	// if needed, we create the registry ourselves.
	harvesterMetrics := monitoring.Default.GetRegistry("filebeat.harvester")
	if harvesterMetrics == nil {
		harvesterMetrics = monitoring.Default.NewRegistry("filebeat.harvester")
	}

	m := Metrics{
		unregister:        ctx.MetricsRegistryCancel,
		FilesOpened:       monitoring.NewUint(ctx.MetricsRegistry, "files_opened_total"),
		FilesClosed:       monitoring.NewUint(ctx.MetricsRegistry, "files_closed_total"),
		FilesActive:       monitoring.NewUint(ctx.MetricsRegistry, "files_active"),
		MessagesRead:      monitoring.NewUint(ctx.MetricsRegistry, "messages_read_total"),
		MessagesTruncated: monitoring.NewUint(ctx.MetricsRegistry, "messages_truncated_total"),
		BytesProcessed:    monitoring.NewUint(ctx.MetricsRegistry, "bytes_processed_total"),
		EventsProcessed:   monitoring.NewUint(ctx.MetricsRegistry, "events_processed_total"),
		ProcessingErrors:  monitoring.NewUint(ctx.MetricsRegistry, "processing_errors_total"),
		ProcessingTime:    metrics.NewUniformSample(1024),

		HarvesterStarted:   monitoring.NewInt(harvesterMetrics, "started"),
		HarvesterClosed:    monitoring.NewInt(harvesterMetrics, "closed"),
		HarvesterRunning:   monitoring.NewInt(harvesterMetrics, "running"),
		HarvesterOpenFiles: monitoring.NewInt(harvesterMetrics, "open_files"),
	}
	_ = adapter.NewGoMetrics(ctx.MetricsRegistry, "processing_time", adapter.Accept).
		Register("histogram", metrics.NewHistogram(m.ProcessingTime))

	return &m
}
