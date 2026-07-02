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
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

func TestFileScanMetricsUpdate(t *testing.T) {
	metrics := NewMetrics(monitoring.NewRegistry(), logp.NewNopLogger())

	// Create an "empty" baseline because these gauges live in the shared
	// filebeat.filestream registry and may already contain values.
	baseline := FileScanMetrics{
		FilesMatched:        metrics.FilesMatched.Get(),
		FilesUnique:         metrics.FilesUnique.Get(),
		FilesNoIngestTarget: metrics.FilesNoIngestTarget.Get(),
		FilesIgnored:        metrics.FilesIgnored.Get(),
		FilesEmpty:          metrics.FilesEmpty.Get(),
	}

	firstScan := FileScanMetrics{
		FilesMatched:        10,
		FilesUnique:         6,
		FilesNoIngestTarget: 3,
		FilesIgnored:        1,
		FilesEmpty:          2,
	}
	metrics.UpdateFileScanMetrics(firstScan)
	assert.Equal(t, firstScan, metrics.lastFileScanMetrics, "file scan metrics after first update")
	assert.Equal(t, baseline.FilesMatched+10, metrics.FilesMatched.Get(), "files_matched")
	assert.Equal(t, baseline.FilesUnique+6, metrics.FilesUnique.Get(), "files_unique")
	assert.Equal(t, baseline.FilesNoIngestTarget+3, metrics.FilesNoIngestTarget.Get(), "files_no_ingest_target")
	assert.Equal(t, baseline.FilesIgnored+1, metrics.FilesIgnored.Get(), "files_ignored")
	assert.Equal(t, baseline.FilesEmpty+2, metrics.FilesEmpty.Get(), "files_empty")

	secondScan := FileScanMetrics{
		FilesMatched:        12,
		FilesUnique:         5,
		FilesNoIngestTarget: 4,
		FilesIgnored:        0,
		FilesEmpty:          1,
	}
	metrics.UpdateFileScanMetrics(secondScan)
	assert.Equal(t, secondScan, metrics.lastFileScanMetrics, "file scan metrics after second update")
	assert.Equal(t, baseline.FilesMatched+12, metrics.FilesMatched.Get(), "files_matched after second update")
	assert.Equal(t, baseline.FilesUnique+5, metrics.FilesUnique.Get(), "files_unique after second update")
	assert.Equal(t, baseline.FilesNoIngestTarget+4, metrics.FilesNoIngestTarget.Get(), "files_no_ingest_target after second update")
	assert.Equal(t, baseline.FilesIgnored, metrics.FilesIgnored.Get(), "files_ignored after second update")
	assert.Equal(t, baseline.FilesEmpty+1, metrics.FilesEmpty.Get(), "files_empty after second update")
}

func TestHarvesterMetricsUpdate(t *testing.T) {
	metrics := NewMetrics(monitoring.NewRegistry(), logp.NewNopLogger())

	// Simulate the Harvester registering files and the ingested offset.
	// The offsets are all at boundary value.
	completeOffset, _ := metrics.RegisterHarvesterOffset("complete", 100)
	nearOffset, _ := metrics.RegisterHarvesterOffset("near", 95)
	laggingOffset, _ := metrics.RegisterHarvesterOffset("lagging", 94)

	// The odd one: harvester has offset 100, but the file is zero bytes.
	// This is here to ensure we don't track metrics for empty files, even
	// if registered by a harvester.
	metrics.RegisterHarvesterOffset("zero-size", 100)

	// Simulate the scanner calling UpdateHarvesterBuckets with the file sizes.
	metrics.UpdateHarvesterBuckets([]HarvesterFile{
		{ID: "complete", Size: 100},
		{ID: "near", Size: 100},
		{ID: "lagging", Size: 100},
		{ID: "not-active", Size: 100}, // aka: not registered with 'RegisterHarvesterOffset'
		{ID: "zero-size", Size: 0},
	})
	assert.Equal(t, HarvesterMetrics{
		FilesIngestedPercent100:    1,
		FilesIngestedPercent95To99: 1,
		FilesIngestedPercentLt95:   1,
	}, metrics.lastHarvesterMetrics, "harvester metrics after first update")

	// Simulate harvester updating the offset after publishing an event
	completeOffset.Store(100) // continues 100% ingested
	nearOffset.Store(100)     // moves from 95to99 bucket to 100 bucket
	laggingOffset.Store(99)   // moves from Lt95 bucket to 95to99 bucket

	// Simulate the scanner calling UpdateHarvesterBuckets with the file sizes.
	// No change in file sizes, however some files don't exist any more.
	metrics.UpdateHarvesterBuckets([]HarvesterFile{
		{ID: "complete", Size: 100},
		{ID: "near", Size: 100},
		{ID: "lagging", Size: 100},
	})
	assert.Equal(t, HarvesterMetrics{
		FilesIngestedPercent100:    2,
		FilesIngestedPercent95To99: 1,
	}, metrics.lastHarvesterMetrics, "harvester metrics after second update")
}

func TestHarvesterMetricsAddFile(t *testing.T) {
	tests := map[string]struct {
		offset   int64
		size     int64
		expected HarvesterMetrics
	}{
		"below 95 percent": {
			offset: 94,
			size:   100,
			expected: HarvesterMetrics{
				FilesIngestedPercentLt95: 1,
			},
		},
		"exactly 95 percent": {
			offset: 95,
			size:   100,
			expected: HarvesterMetrics{
				FilesIngestedPercent95To99: 1,
			},
		},
		"99 percent": {
			offset: 99,
			size:   100,
			expected: HarvesterMetrics{
				FilesIngestedPercent95To99: 1,
			},
		},
		"complete": {
			offset: 100,
			size:   100,
			expected: HarvesterMetrics{
				FilesIngestedPercent100: 1,
			},
		},
		"offset greater than size": {
			offset: 101,
			size:   100,
			expected: HarvesterMetrics{
				FilesIngestedPercent100: 1,
			},
		},
		"rounded 95 percent threshold": {
			offset: 96,
			size:   101,
			expected: HarvesterMetrics{
				FilesIngestedPercent95To99: 1,
			},
		},
		"small file below 95 percent": {
			offset: 18,
			size:   19,
			expected: HarvesterMetrics{
				FilesIngestedPercentLt95: 1,
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			actual := HarvesterMetrics{}
			actual.addFile(test.offset, test.size)
			assert.Equal(t, test.expected, actual, "unexpected harvester bucket")
		})
	}
}

func TestHarvesterOffsetRegistration(t *testing.T) {
	metrics := NewMetrics(monitoring.NewRegistry(), logp.NewNopLogger())

	firstOffset, cleanupFirstOffset := metrics.RegisterHarvesterOffset("test-id", 10)
	assert.NotNil(t, firstOffset, "registered harvester offset should not be nil")

	offset, ok := metrics.harvesterOffsets["test-id"]
	assert.True(t, ok, "registered harvester offset should be found")
	assert.Same(t, firstOffset, offset, "registered harvester offset should match returned offset")
	assert.EqualValues(t, 10, offset.Load(), "registered harvester offset")

	firstOffset.Store(42)
	offset, ok = metrics.harvesterOffsets["test-id"]
	assert.True(t, ok, "updated harvester offset should be found")
	assert.EqualValues(t, 42, offset.Load(), "updated harvester offset")

	secondOffset, cleanupSecondOffset := metrics.RegisterHarvesterOffset("test-id", 100)
	assert.NotSame(t, firstOffset, secondOffset, "re-registering should create a new active harvester offset")

	cleanupFirstOffset()
	offset, ok = metrics.harvesterOffsets["test-id"]
	assert.True(t, ok, "removing a stale harvester offset should keep the current one")
	assert.Same(t, secondOffset, offset, "stale harvester removal should not remove current offset")

	cleanupSecondOffset()
	_, ok = metrics.harvesterOffsets["test-id"]
	assert.False(t, ok, "removing the active harvester offset should clear it")
}

func TestMetricsCleanup(t *testing.T) {
	metrics := NewMetrics(monitoring.NewRegistry(), logp.NewNopLogger())

	metrics.UpdateFileScanMetrics(FileScanMetrics{
		FilesMatched:        5,
		FilesUnique:         4,
		FilesNoIngestTarget: 3,
		FilesIgnored:        2,
		FilesEmpty:          1,
	})
	metrics.RegisterHarvesterOffset("test-id", 10)
	metrics.UpdateHarvesterBuckets([]HarvesterFile{
		{ID: "test-id", Size: 10},
	})

	metrics.Cleanup()

	assert.Equal(t, FileScanMetrics{}, metrics.lastFileScanMetrics, "file scan metrics snapshot after cleanup")
	assert.Equal(t, HarvesterMetrics{}, metrics.lastHarvesterMetrics, "harvester metrics snapshot after cleanup")
	_, ok := metrics.harvesterOffsets["test-id"]
	assert.False(t, ok, "cleanup should remove active harvester offsets")
}
