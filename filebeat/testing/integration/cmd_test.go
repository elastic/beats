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
	"testing"
	"time"

	"github.com/elastic/beats/v7/libbeat/testing/integration"
)

// Tests filebeat --once command
func TestFileBeatOnceCommand(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Minute)
	defer cancel()
	EnsureCompiled(ctx, t)

	messagePrefix := "sample test message"
	fileCount := 1
	lineCount := 10

	reportOptions := integration.ReportOptions{
		PrintLinesOnFail:  100,
		PrintConfigOnFail: true,
	}

	generator := NewPlainTextGenerator(messagePrefix)
	path, files := GenerateLogFiles(t, fileCount, lineCount, generator)

	config := `
filebeat.inputs:
  - type: log
    enabled: true
    id: "test-log"
    allow_deprecated_use: true
    paths:
     - %s
output.console:
  enabled: true
`

	test := NewTest(t, TestOptions{
		Config: fmt.Sprintf(config, path),
		Args:   []string{"--once"},
	})

	// ensuring we ingest every line from every file
	for _, filename := range files {
		for i := 1; i <= lineCount; i++ {
			line := fmt.Sprintf("%s:%d", filepath.Base(filename), i)
			test.ExpectOutput(line)
		}
	}

	// // expect filebeat to exit
	// test.ExpectOutput("filebeat stopped")

	test.
		ExpectEOF(files...).
		WithReportOptions(reportOptions).
		ExpectStart().
		Start(ctx).
		Wait()
}
