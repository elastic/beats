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
