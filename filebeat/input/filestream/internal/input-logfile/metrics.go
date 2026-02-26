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

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
	"github.com/elastic/elastic-agent-libs/monitoring/adapter"
)

// Metrics defines a set of metrics for the filestream input.
type Metrics struct {
	// Total metrics: plain and GZIP files
	FilesOpened       *monitoring.Uint // Number of files that have been opened.
	FilesClosed       *monitoring.Uint // Number of files closed.
	FilesActive       *monitoring.Uint // Number of files currently open (gauge).
	MessagesRead      *monitoring.Uint // Number of messages read.
	MessagesTruncated *monitoring.Uint // Number of messages truncated.
	BytesProcessed    *monitoring.Uint // Number of bytes processed.
	EventsProcessed   *monitoring.Uint // Number of events processed.
	ProcessingErrors  *monitoring.Uint // Number of processing errors.
	ProcessingTime    metrics.Sample   // Histogram of the elapsed time for processing an event.

	// GZIP only metrics
	FilesGZIPOpened       *monitoring.Uint // Number of files that have been opened.
	FilesGZIPClosed       *monitoring.Uint // Number of files closed.
	FilesGZIPActive       *monitoring.Uint // Number of files currently open (gauge).
	MessagesGZIPRead      *monitoring.Uint // Number of messages read.
	MessagesGZIPTruncated *monitoring.Uint // Number of messages truncated.
	BytesGZIPProcessed    *monitoring.Uint // Number of bytes processed.
	EventsGZIPProcessed   *monitoring.Uint // Number of events processed.
	ProcessingGZIPErrors  *monitoring.Uint // Number of processing errors.
	ProcessingGZIPTime    metrics.Sample   // Histogram of the elapsed time for processing an event.

	// Those metrics use the same registry/keys as the log input uses
	// Total metrics: plain and GZIP files
	HarvesterStarted   *monitoring.Int
	HarvesterClosed    *monitoring.Int
	HarvesterRunning   *monitoring.Int
	HarvesterOpenFiles *monitoring.Int

	// GZIP only metrics
	HarvesterGZIPStarted   *monitoring.Int
	HarvesterGZIPClosed    *monitoring.Int
	HarvesterGZIPRunning   *monitoring.Int
	HarvesterOpenGZIPFiles *monitoring.Int
}

func NewMetrics(reg *monitoring.Registry, logger *logp.Logger) *Metrics {
	// The log input creates the `filebeat.harvester` registry as a package
	// variable, so it should always exist before this function runs.
	// However, at least on testing scenarios this does not hold true, so
	// if needed, we create the registry ourselves.
	harvesterMetrics := monitoring.Default.GetOrCreateRegistry("filebeat.harvester")

	m := Metrics{
		FilesOpened:       monitoring.NewUint(reg, "files_opened_total"),
		FilesClosed:       monitoring.NewUint(reg, "files_closed_total"),
		FilesActive:       monitoring.NewUint(reg, "files_active"),
		MessagesRead:      monitoring.NewUint(reg, "messages_read_total"),
		MessagesTruncated: monitoring.NewUint(reg, "messages_truncated_total"),
		BytesProcessed:    monitoring.NewUint(reg, "bytes_processed_total"),
		EventsProcessed:   monitoring.NewUint(reg, "events_processed_total"),
		ProcessingErrors:  monitoring.NewUint(reg, "processing_errors_total"),
		ProcessingTime:    metrics.NewUniformSample(1024),

		FilesGZIPOpened:       monitoring.NewUint(reg, "gzip_files_opened_total"),
		FilesGZIPClosed:       monitoring.NewUint(reg, "gzip_files_closed_total"),
		FilesGZIPActive:       monitoring.NewUint(reg, "gzip_files_active"),
		MessagesGZIPRead:      monitoring.NewUint(reg, "gzip_messages_read_total"),
		MessagesGZIPTruncated: monitoring.NewUint(reg, "gzip_messages_truncated_total"),
		BytesGZIPProcessed:    monitoring.NewUint(reg, "gzip_bytes_processed_total"),
		EventsGZIPProcessed:   monitoring.NewUint(reg, "gzip_events_processed_total"),
		ProcessingGZIPErrors:  monitoring.NewUint(reg, "gzip_processing_errors_total"),
		ProcessingGZIPTime:    metrics.NewUniformSample(1024),

		HarvesterStarted:   monitoring.NewInt(harvesterMetrics, "started"),
		HarvesterClosed:    monitoring.NewInt(harvesterMetrics, "closed"),
		HarvesterRunning:   monitoring.NewInt(harvesterMetrics, "running"),
		HarvesterOpenFiles: monitoring.NewInt(harvesterMetrics, "open_files"),

		HarvesterGZIPStarted:   monitoring.NewInt(harvesterMetrics, "gzip_started"),
		HarvesterGZIPClosed:    monitoring.NewInt(harvesterMetrics, "gzip_closed"),
		HarvesterGZIPRunning:   monitoring.NewInt(harvesterMetrics, "gzip_running"),
		HarvesterOpenGZIPFiles: monitoring.NewInt(harvesterMetrics, "gzip_open_files"),
	}
	_ = adapter.NewGoMetrics(reg, "processing_time", logger, adapter.Accept).
		Register("histogram", metrics.NewHistogram(m.ProcessingTime))
	_ = adapter.NewGoMetrics(reg, "gzip_processing_time", logger, adapter.Accept).
		Register("histogram", metrics.NewHistogram(m.ProcessingGZIPTime))

	return &m
}
