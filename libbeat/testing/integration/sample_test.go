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

package integration

import (
	"context"
	"testing"
	"time"
)

func TestFilebeat(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	EnsureCompiled(ctx, t, "filebeat")

	reportOptions := ReportOptions{
		PrintExpectationsBeforeStart: true,
		// last 10 output lines would suffice
		PrintLinesOnFail: 10,
	}

	t.Run("Filebeat starts", func(t *testing.T) {
		config := `
filebeat.inputs:
  - type: filestream
    id: "test-filestream"
    paths:
      - /var/log/*.log
# we want to check that all messages are ingested
# without using an external service, this is an easy way
output.console:
  enabled: true
`
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		test := NewBeatTest(t, BeatTestOptions{
			Beatname: "filebeat",
			Config:   config,
		})

		test.
			WithReportOptions(reportOptions).
			// we should observe the start message of the Beat
			ExpectStart().
			// check that the first and the last line of the file get ingested
			Start(ctx).
			// wait until all the expectations are met
			// or we hit the timeout set by the context
			Wait()
	})

	t.Run("Filebeat crashes due to incorrect config", func(t *testing.T) {
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
		test := NewBeatTest(t, BeatTestOptions{
			Beatname: "filebeat",
			Config:   config,
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
