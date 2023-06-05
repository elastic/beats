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

	"github.com/elastic/elastic-agent-libs/monitoring"
	"github.com/elastic/elastic-agent-libs/monitoring/adapter"

	"github.com/elastic/beats/v7/libbeat/monitoring/inputmon"
)

// Metrics defines a set of metrics for the filestream input.
type Metrics struct {
	unregister func()

	FilesOpened      *monitoring.Uint // Number of files that have been opened.
	FilesClosed      *monitoring.Uint // Number of files closed.
	FilesCreated     *monitoring.Uint // Number of new file events found by the prospector.
	FilesUpdated     *monitoring.Uint // Number of file update events detected by the prospector.
	FilesIgnored     *monitoring.Uint // Number of ignored file events detected by the prospector.
	FilesTruncated   *monitoring.Uint // Number of truncated file events detected by the prospector.
	FilesDeleted     *monitoring.Uint // Number of deleted file events detected by the prospector.
	FilesRenamed     *monitoring.Uint // Number of renamed file events detected by the prospector.
	FilesActive      *monitoring.Uint // Number of files currently open (gauge).
	MessagesRead     *monitoring.Uint // Number of messages read.
	BytesProcessed   *monitoring.Uint // Number of bytes processed.
	EventsProcessed  *monitoring.Uint // Number of events processed.
	ProcessingErrors *monitoring.Uint // Number of processing errors.
	ProcessingTime   metrics.Sample   // Histogram of the elapsed time for processing an event.
}

func (m *Metrics) Close() {
	if m == nil {
		return
	}

	m.unregister()
}

func NewMetrics(id string) *Metrics {
	reg, unreg := inputmon.NewInputRegistry("filestream", id, nil)
	m := Metrics{
		unregister:       unreg,
		FilesOpened:      monitoring.NewUint(reg, "files_opened_total"),
		FilesClosed:      monitoring.NewUint(reg, "files_closed_total"),
		FilesCreated:     monitoring.NewUint(reg, "files_created_total"),
		FilesUpdated:     monitoring.NewUint(reg, "files_updated_total"),
		FilesIgnored:     monitoring.NewUint(reg, "files_ignored_total"),
		FilesTruncated:   monitoring.NewUint(reg, "files_truncated_total"),
		FilesDeleted:     monitoring.NewUint(reg, "files_deleted_total"),
		FilesRenamed:     monitoring.NewUint(reg, "files_renamed_total"),
		FilesActive:      monitoring.NewUint(reg, "files_active"),
		MessagesRead:     monitoring.NewUint(reg, "messages_read_total"),
		BytesProcessed:   monitoring.NewUint(reg, "bytes_processed_total"),
		EventsProcessed:  monitoring.NewUint(reg, "events_processed_total"),
		ProcessingErrors: monitoring.NewUint(reg, "processing_errors_total"),
		ProcessingTime:   metrics.NewUniformSample(1024),
	}
	_ = adapter.NewGoMetrics(reg, "processing_time", adapter.Accept).
		Register("histogram", metrics.NewHistogram(m.ProcessingTime))

	return &m
}
