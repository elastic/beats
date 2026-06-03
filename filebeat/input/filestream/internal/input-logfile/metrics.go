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
	"sync"
	"sync/atomic"

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

	FilesMatched        *monitoring.Int // Number of files matched by the configured paths (gauge).
	FilesUnique         *monitoring.Int // Number of unique ingestible files found by the scanner (gauge).
	FilesNoIngestTarget *monitoring.Int // Number of matched non-empty files without an ingest target, too small, or other internal errors (gauge).
	FilesIgnored        *monitoring.Int // Number of ingestible files ignored by filestream settings (gauge).
	FilesEmpty          *monitoring.Int // Number of empty files found by the scanner (gauge).

	FilesIngestedPercent100    *monitoring.Int // Number of active harvesters that have fully ingested their files (gauge).
	FilesIngestedPercent95To99 *monitoring.Int // Number of active harvesters that have ingested 95-99% of their files (gauge).
	FilesIngestedPercentLt95   *monitoring.Int // Number of active harvesters that have ingested less than 95% of their files (gauge).

	harvesterMetricsMu sync.Mutex
	harvesterOffsets   map[string]*atomic.Int64

	lastFileScanMetrics  FileScanMetrics
	lastHarvesterMetrics HarvesterMetrics
}

// FileScanMetrics contains one filestream scanner snapshot for an input.
type FileScanMetrics struct {
	FilesMatched        int64
	FilesUnique         int64
	FilesNoIngestTarget int64
	FilesIgnored        int64
	FilesEmpty          int64
}

// HarvesterMetrics contains the harvester progress snapshot for an input.
type HarvesterMetrics struct {
	FilesIngestedPercent100    int64
	FilesIngestedPercent95To99 int64
	FilesIngestedPercentLt95   int64
}

// HarvesterFile contains the scanner-observed file state needed to calculate
// harvester progress metrics.
type HarvesterFile struct {
	ID   string
	Size int64
}

func (m *HarvesterMetrics) addFile(offset, size int64) {
	switch {
	case offset >= size:
		m.FilesIngestedPercent100++
	case offset >= size-size/20:
		m.FilesIngestedPercent95To99++
	default:
		m.FilesIngestedPercentLt95++
	}
}

func NewMetrics(reg *monitoring.Registry, logger *logp.Logger) *Metrics {
	// The log input creates the `filebeat.harvester` registry as a package
	// variable, so it should always exist before this function runs.
	// However, at least on testing scenarios this does not hold true, so
	// if needed, we create the registry ourselves.
	harvesterMetrics := monitoring.Default.GetOrCreateRegistry("filebeat.harvester")
	filestreamMetrics := monitoring.Default.GetOrCreateRegistry("filebeat.filestream")

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

		FilesMatched:        monitoring.NewInt(filestreamMetrics, "files_matched"),
		FilesUnique:         monitoring.NewInt(filestreamMetrics, "files_unique"),
		FilesNoIngestTarget: monitoring.NewInt(filestreamMetrics, "files_no_ingest_target"),
		FilesIgnored:        monitoring.NewInt(filestreamMetrics, "files_ignored"),
		FilesEmpty:          monitoring.NewInt(filestreamMetrics, "files_empty"),

		FilesIngestedPercent100:    monitoring.NewInt(filestreamMetrics, "files_ingested_percent_100"),
		FilesIngestedPercent95To99: monitoring.NewInt(filestreamMetrics, "files_ingested_percent_95_99"),
		FilesIngestedPercentLt95:   monitoring.NewInt(filestreamMetrics, "files_ingested_percent_lt_95"),

		harvesterOffsets: map[string]*atomic.Int64{},
	}
	_ = adapter.NewGoMetrics(reg, "processing_time", logger, adapter.Accept).
		Register("histogram", metrics.NewHistogram(m.ProcessingTime))
	_ = adapter.NewGoMetrics(reg, "gzip_processing_time", logger, adapter.Accept).
		Register("histogram", metrics.NewHistogram(m.ProcessingGZIPTime))

	return &m
}

// UpdateFileScanMetrics updates the aggregate filestream scan gauges with this
// input's delta since the previous scan.
func (m *Metrics) UpdateFileScanMetrics(current FileScanMetrics) {
	if m == nil {
		return
	}

	m.FilesMatched.Add(current.FilesMatched - m.lastFileScanMetrics.FilesMatched)
	m.FilesUnique.Add(current.FilesUnique - m.lastFileScanMetrics.FilesUnique)
	m.FilesNoIngestTarget.Add(current.FilesNoIngestTarget - m.lastFileScanMetrics.FilesNoIngestTarget)
	m.FilesIgnored.Add(current.FilesIgnored - m.lastFileScanMetrics.FilesIgnored)
	m.FilesEmpty.Add(current.FilesEmpty - m.lastFileScanMetrics.FilesEmpty)

	m.lastFileScanMetrics = current
}

// CleanupFileScanMetrics removes this input's last file scan contribution from
// the shared aggregate scan gauges. Call this during input/prospector shutdown
// so stale scan counts are not left behind after an input stops or restarts.
func (m *Metrics) CleanupFileScanMetrics() {
	m.UpdateFileScanMetrics(FileScanMetrics{})
}

// RegisterHarvesterOffset registers an active harvester offset and returns the
// atomic value the harvester must update while reading, plus a cleanup function.
//
// The cleanup function only removes the offset if it still points to the same
// value registered by this call. This protects a restarted harvester for the
// same source ID from an older harvester's deferred cleanup: if the new
// harvester has already registered a replacement offset, the old cleanup must
// not delete it.
func (m *Metrics) RegisterHarvesterOffset(id string, offset int64) (*atomic.Int64, func()) {
	if m == nil {
		return nil, func() {}
	}

	activeOffset := &atomic.Int64{}
	activeOffset.Store(offset)

	m.harvesterMetricsMu.Lock()
	defer m.harvesterMetricsMu.Unlock()

	m.harvesterOffsets[id] = activeOffset

	cleanup := func() {
		m.harvesterMetricsMu.Lock()
		defer m.harvesterMetricsMu.Unlock()

		if m.harvesterOffsets[id] != activeOffset {
			return
		}
		delete(m.harvesterOffsets, id)
	}

	return activeOffset, cleanup
}

// UpdateHarvesterBuckets updates the aggregate harvester progress gauges from
// the current scanner-observed file snapshot.
func (m *Metrics) UpdateHarvesterBuckets(files []HarvesterFile) {
	if m == nil {
		return
	}

	m.harvesterMetricsMu.Lock()
	defer m.harvesterMetricsMu.Unlock()

	current := HarvesterMetrics{}
	for _, file := range files {
		if file.Size <= 0 {
			continue
		}

		offset, ok := m.harvesterOffsets[file.ID]
		if !ok {
			continue
		}

		current.addFile(offset.Load(), file.Size)
	}

	m.updateHarvesterBucketsLocked(current)
}

// updateHarvesterBucketsLocked updates the harvester metrics bucket,
// the caller MUST hold the lock on harvesterMetricsMu when calling this
// method.
func (m *Metrics) updateHarvesterBucketsLocked(current HarvesterMetrics) {
	m.FilesIngestedPercent100.Add(current.FilesIngestedPercent100 - m.lastHarvesterMetrics.FilesIngestedPercent100)
	m.FilesIngestedPercent95To99.Add(current.FilesIngestedPercent95To99 - m.lastHarvesterMetrics.FilesIngestedPercent95To99)
	m.FilesIngestedPercentLt95.Add(current.FilesIngestedPercentLt95 - m.lastHarvesterMetrics.FilesIngestedPercentLt95)

	m.lastHarvesterMetrics = current
}

// CleanupHarvesterMetrics removes this input's harvester metric contribution
// from the shared aggregate gauges and clears active harvester offsets.
func (m *Metrics) CleanupHarvesterMetrics() {
	if m == nil {
		return
	}

	m.harvesterMetricsMu.Lock()
	defer m.harvesterMetricsMu.Unlock()

	m.updateHarvesterBucketsLocked(HarvesterMetrics{})
	clear(m.harvesterOffsets)
}
