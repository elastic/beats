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
	"path/filepath"
	"testing"
	"time"

	libbeatintegration "github.com/elastic/beats/v7/libbeat/testing/integration"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// TestBase covers the fundamental tests from test_base.py.
func TestBase(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Minute)
	defer cancel()
	EnsureCompiled(ctx, t)

	reportOptions := libbeatintegration.ReportOptions{
		PrintLinesOnFail:  100,
		PrintConfigOnFail: true,
	}

	t.Run("BasicFieldsExist", func(t *testing.T) {
		// Verify that core ECS fields are present in every event.
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		logDir := t.TempDir()
		logFile := filepath.Join(logDir, "test.log")
		WriteFile(t, logFile, "test message\n")

		config := FilestreamInputConfig("base-test", logFile, FilestreamOptions{})
		test := NewTest(t, TestOptions{Config: config})

		test.ExpectJSONFields(mapstr.M{"message": "test message"})
		test.ExpectJSONFields(mapstr.M{"input.type": "filestream"})
		test.WithReportOptions(reportOptions)
		test.
			ExpectEOF(logFile).
			ExpectStart().
			Start(ctx).
			Wait()
	})

	t.Run("ShutdownNoInputs", func(t *testing.T) {
		// When no inputs are configured, Filebeat should exit with an error.
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		config := `
filebeat.inputs: []
output.console:
  enabled: true
`
		test := NewTest(t, TestOptions{Config: config})
		test.
			WithReportOptions(reportOptions).
			ExpectOutput("no modules or inputs enabled").
			ExpectStop(1).
			Start(ctx).
			Wait()
	})
}
