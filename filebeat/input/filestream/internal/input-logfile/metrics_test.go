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

	// Create an "empty" baseline
	baseline := FileScanMetrics{
		FilesMatched:        metrics.FilesMatched.Get(),
		FilesUnique:         metrics.FilesUnique.Get(),
		FilesNoIngestTarget: metrics.FilesNoIngestTarget.Get(),
		FilesIgnored:        metrics.FilesIgnored.Get(),
		FilesEmpty:          metrics.FilesEmpty.Get(),
	}

	metrics.UpdateFileScanMetrics(FileScanMetrics{
		FilesMatched:        10,
		FilesUnique:         6,
		FilesNoIngestTarget: 3,
		FilesIgnored:        1,
		FilesEmpty:          2,
	})
	assert.Equal(t, baseline.FilesMatched+10, metrics.FilesMatched.Get(), "files_matched")
	assert.Equal(t, baseline.FilesUnique+6, metrics.FilesUnique.Get(), "files_unique")
	assert.Equal(t, baseline.FilesNoIngestTarget+3, metrics.FilesNoIngestTarget.Get(), "files_no_ingest_target")
	assert.Equal(t, baseline.FilesIgnored+1, metrics.FilesIgnored.Get(), "files_ignored")
	assert.Equal(t, baseline.FilesEmpty+2, metrics.FilesEmpty.Get(), "files_empty")

	metrics.UpdateFileScanMetrics(FileScanMetrics{
		FilesMatched:        12,
		FilesUnique:         5,
		FilesNoIngestTarget: 4,
		FilesIgnored:        0,
		FilesEmpty:          1,
	})
	assert.Equal(t, baseline.FilesMatched+12, metrics.FilesMatched.Get(), "files_matched after second update")
	assert.Equal(t, baseline.FilesUnique+5, metrics.FilesUnique.Get(), "files_unique after second update")
	assert.Equal(t, baseline.FilesNoIngestTarget+4, metrics.FilesNoIngestTarget.Get(), "files_no_ingest_target after second update")
	assert.Equal(t, baseline.FilesIgnored, metrics.FilesIgnored.Get(), "files_ignored after second update")
	assert.Equal(t, baseline.FilesEmpty+1, metrics.FilesEmpty.Get(), "files_empty after second update")
}

func TestHarvesterMetricsUpdate(t *testing.T) {
	metrics := NewMetrics(monitoring.NewRegistry(), logp.NewNopLogger())

	baseline := HarvesterMetrics{
		FilesIngestedPercent100:    metrics.FilesIngestedPercent100.Get(),
		FilesIngestedPercent95To99: metrics.FilesIngestedPercent95To99.Get(),
		FilesIngestedPercentLt95:   metrics.FilesIngestedPercentLt95.Get(),
	}

	metrics.UpdateHarvesterBuckets(HarvesterMetrics{
		FilesIngestedPercent100:    1,
		FilesIngestedPercent95To99: 2,
		FilesIngestedPercentLt95:   3,
	})
	assert.Equal(t, baseline.FilesIngestedPercent100+1, metrics.FilesIngestedPercent100.Get(), "files_ingested_percent_100")
	assert.Equal(t, baseline.FilesIngestedPercent95To99+2, metrics.FilesIngestedPercent95To99.Get(), "files_ingested_percent_95_99")
	assert.Equal(t, baseline.FilesIngestedPercentLt95+3, metrics.FilesIngestedPercentLt95.Get(), "files_ingested_percent_lt_95")

	metrics.UpdateHarvesterBuckets(HarvesterMetrics{
		FilesIngestedPercent100:    2,
		FilesIngestedPercent95To99: 1,
		FilesIngestedPercentLt95:   0,
	})
	assert.Equal(t, baseline.FilesIngestedPercent100+2, metrics.FilesIngestedPercent100.Get(), "files_ingested_percent_100 after second update")
	assert.Equal(t, baseline.FilesIngestedPercent95To99+1, metrics.FilesIngestedPercent95To99.Get(), "files_ingested_percent_95_99 after second update")
	assert.Equal(t, baseline.FilesIngestedPercentLt95, metrics.FilesIngestedPercentLt95.Get(), "files_ingested_percent_lt_95 after second update")
}

func TestHarvesterOffsetRegistration(t *testing.T) {
	metrics := NewMetrics(monitoring.NewRegistry(), logp.NewNopLogger())

	firstOffset, cleanupFirstOffset := metrics.RegisterHarvesterOffset("test-id", 10)
	assert.NotNil(t, firstOffset, "registered harvester offset should not be nil")

	offset, ok := metrics.FindHarvesterOffset("test-id")
	assert.True(t, ok, "registered harvester offset should be found")
	assert.Same(t, firstOffset, offset, "registered harvester offset should match returned offset")
	assert.EqualValues(t, 10, offset.Load(), "registered harvester offset")

	firstOffset.Store(42)
	offset, ok = metrics.FindHarvesterOffset("test-id")
	assert.True(t, ok, "updated harvester offset should be found")
	assert.EqualValues(t, 42, offset.Load(), "updated harvester offset")

	secondOffset, cleanupSecondOffset := metrics.RegisterHarvesterOffset("test-id", 100)
	assert.NotSame(t, firstOffset, secondOffset, "re-registering should create a new active harvester offset")

	cleanupFirstOffset()
	offset, ok = metrics.FindHarvesterOffset("test-id")
	assert.True(t, ok, "removing a stale harvester offset should keep the current one")
	assert.Same(t, secondOffset, offset, "stale harvester removal should not remove current offset")

	cleanupSecondOffset()
	_, ok = metrics.FindHarvesterOffset("test-id")
	assert.False(t, ok, "removing the active harvester offset should clear it")
}

func TestHarvesterMetricsCleanup(t *testing.T) {
	metrics := NewMetrics(monitoring.NewRegistry(), logp.NewNopLogger())

	baseline := HarvesterMetrics{
		FilesIngestedPercent100:    metrics.FilesIngestedPercent100.Get(),
		FilesIngestedPercent95To99: metrics.FilesIngestedPercent95To99.Get(),
		FilesIngestedPercentLt95:   metrics.FilesIngestedPercentLt95.Get(),
	}

	metrics.RegisterHarvesterOffset("test-id", 10)
	metrics.UpdateHarvesterBuckets(HarvesterMetrics{
		FilesIngestedPercent100:    1,
		FilesIngestedPercent95To99: 2,
		FilesIngestedPercentLt95:   3,
	})

	metrics.CleanupHarvesterMetrics()

	assert.EqualValues(t, baseline.FilesIngestedPercent100, metrics.FilesIngestedPercent100.Get(), "files_ingested_percent_100 after cleanup")
	assert.EqualValues(t, baseline.FilesIngestedPercent95To99, metrics.FilesIngestedPercent95To99.Get(), "files_ingested_percent_95_99 after cleanup")
	assert.EqualValues(t, baseline.FilesIngestedPercentLt95, metrics.FilesIngestedPercentLt95.Get(), "files_ingested_percent_lt_95 after cleanup")
	_, ok := metrics.FindHarvesterOffset("test-id")
	assert.False(t, ok, "cleanup should remove active harvester offsets")
}
