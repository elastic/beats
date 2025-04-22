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

func TestFilebeat(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	EnsureCompiled(ctx, t)

	messagePrefix := "sample test message"
	fileCount := 5
	lineCount := 128

	reportOptions := integration.ReportOptions{
		PrintLinesOnFail:  10,
		PrintConfigOnFail: true,
	}

	t.Run("Filebeat starts and ingests files", func(t *testing.T) {
		configTemplate := `
filebeat.inputs:
  - type: filestream
    id: "test-filestream"
    paths:
      - %s
# we want to check that all messages are ingested
# without using an external service, this is an easy way
output.console:
  enabled: true
`
		// we can generate any amount of expectations
		// they are light-weight
		expectIngestedFiles := func(test Test, files []string) {
			// ensuring we ingest every line from every file
			for _, filename := range files {
				for i := 1; i <= lineCount; i++ {
					line := fmt.Sprintf("%s %s:%d", messagePrefix, filepath.Base(filename), i)
					test.ExpectOutput(line)
				}
			}
		}

		t.Run("plain text files", func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()

			generator := NewPlainTextGenerator(messagePrefix)
			path, files := GenerateLogFiles(t, fileCount, lineCount, generator)
			config := fmt.Sprintf(configTemplate, path)
			test := NewTest(t, TestOptions{
				Config: config,
			})

			expectIngestedFiles(test, files)

			test.
				// we expect to read all generated files to EOF
				ExpectEOF(files...).
				WithReportOptions(reportOptions).
				// we should observe the start message of the Beat
				ExpectStart().
				// check that the first and the last line of the file get ingested
				Start(ctx).
				// wait until all the expectations are met
				// or we hit the timeout set by the context
				Wait()
		})

		t.Run("JSON files", func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()

			generator := NewJSONGenerator(messagePrefix)
			path, files := GenerateLogFiles(t, fileCount, lineCount, generator)
			config := fmt.Sprintf(configTemplate, path)
			test := NewTest(t, TestOptions{
				Config: config,
			})

			expectIngestedFiles(test, files)

			test.
				ExpectEOF(files...).
				WithReportOptions(reportOptions).
				ExpectStart().
				Start(ctx).
				Wait()
		})
	})

	t.Run("Filebeat crashes due to incorrect config", func(t *testing.T) {
		t.Skip("Flaky test: https://github.com/elastic/beats/issues/42778")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// path items are required, this config is invalid
		config := `
filebeat.inputs:
  - type: filestream
    id: "test-filestream"
output.console:
  enabled: true
`
		test := NewTest(t, TestOptions{
			Config: config,
		})

		test.
			WithReportOptions(reportOptions).
			ExpectStart().
			ExpectOutput("Exiting: Failed to start crawler: starting input failed: error while initializing input: no path is configured").
			ExpectStop(1).
			Start(ctx).
			Wait()
	})
}
