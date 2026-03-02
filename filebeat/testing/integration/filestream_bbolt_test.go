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

//go:build integration

package integration

import (
	"context"
	"fmt"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/testing/integration"
)

func TestFilestreamBboltBackendResumesIngestion(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	EnsureCompiled(ctx, t)

	reportOptions := integration.ReportOptions{
		PrintLinesOnFail:  50,
		PrintConfigOnFail: true,
	}

	dataDir := t.TempDir()

	inputDir := t.TempDir()
	logFile := filepath.Join(inputDir, "test.log")

	lineCount := 100
	prefix := "bbolt-test-log-line-prefix"
	generator := NewPlainTextGenerator(prefix)
	GenerateLogFile(t, logFile, lineCount, generator)

	config := fmt.Sprintf(`
filebeat.inputs:
  - type: filestream
    id: "test-bbolt-backend"
    paths:
      - %s

filebeat.registry:
  backend: bbolt
  flush: 10ms

output.console:
  enabled: true
  bulk_max_size: 0 # required for the output counters

path.data: %s # reusing the same registry

logging:
  level: debug
`, logFile, dataDir)

	t.Run("ingest to EOF", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		test := NewTest(t, TestOptions{
			Config: config,
		})

		processedCount := &atomic.Int64{}

		test.
			ExpectIngestedToConsole(logFile, 0, lineCount).
			ExpectEOF(logFile).
			CountOutput(processedCount, prefix).
			WithReportOptions(reportOptions).
			ExpectStart().
			Start(ctx).
			Wait()

		assert.Equalf(t, int64(lineCount), processedCount.Load(), "%d lines should be processed at this point", lineCount)
	})

	extraLineCount := 10
	AppendLogFile(t, logFile, extraLineCount, generator)

	t.Run("resume from saved offset and hit EOF again", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		test := NewTest(t, TestOptions{
			Config: config,
		})

		processedCount := &atomic.Int64{}

		test.
			ExpectIngestedToConsole(logFile, lineCount, lineCount+extraLineCount).
			ExpectEOF(logFile).
			CountOutput(processedCount, prefix).
			WithReportOptions(reportOptions).
			Start(ctx).
			Wait()

		assert.Equalf(t, int64(extraLineCount), processedCount.Load(), "%d additional lines should be processed", extraLineCount)
	})
}
