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
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/elastic/beats/v7/libbeat/testing/integration"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestCustomFields(t *testing.T) {
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
	path, file := GenerateLogFiles(t, fileCount, lineCount, generator)

	t.Run("tests that custom fields show up in the output dict and  agent.name defaults to hostname", func(t *testing.T) {

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		config := `
filebeat.inputs:
  - type: filestream
    id: "test-filestream"
    paths:
     - %s
    fields:
      hello: world
      number: 2
    file_identity.native: ~
    prospector.scanner.fingerprint.enabled: false	  
output.console:
  enabled: true
`

		test := NewTest(t, TestOptions{
			Config: fmt.Sprintf(config, path),
		})

		host, _ := os.Hostname()
		line := fmt.Sprintf("%s:%d", filepath.Base(file[0]), 1)

		test.ExpectJSONFields(mapstr.M{
			"message": fmt.Sprintf("sample test message %s", line),
			"fields": mapstr.M{
				"number": float64(2),
			},
			"fields.hello": "world",
			"hostname":     host,
		})

		test.
			ExpectEOF(file...).
			WithReportOptions(reportOptions).
			Start(ctx).
			Wait()
	})

	t.Run("tests that custom fields show up in the output dict when fields_under_root: true", func(t *testing.T) {

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		config := `
filebeat.inputs:
  - type: filestream
    id: "test-filestream"
    paths:
     - %s
    fields_under_root: true
    fields:
      hello: world
      number: 2
    file_identity.native: ~
    prospector.scanner.fingerprint.enabled: false	  
output.console:
  enabled: true
`

		test := NewTest(t, TestOptions{
			Config: fmt.Sprintf(config, path),
		})

		line := fmt.Sprintf("%s:%d", filepath.Base(file[0]), 1)
		test.ExpectJSONFields(mapstr.M{
			"message": fmt.Sprintf("sample test message %s", line),
			"number":  float64(2),
			"hello":   "world",
		})

		test.
			ExpectEOF(file...).
			WithReportOptions(reportOptions).
			Start(ctx).
			Wait()
	})

	t.Run("Checks that it's possible to set a custom agent name.", func(t *testing.T) {

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		config := `
filebeat.inputs:
  - type: filestream
    id: "test-filestream"
    paths:
     - %s
    file_identity.native: ~
    prospector.scanner.fingerprint.enabled: false
name: testShipperName
output.console:
  enabled: true
`

		test := NewTest(t, TestOptions{
			Config: fmt.Sprintf(config, path),
		})

		line := fmt.Sprintf("%s:%d", filepath.Base(file[0]), 1)
		test.ExpectJSONFields(mapstr.M{
			"message":    fmt.Sprintf("sample test message %s", line),
			"host.name":  "testShipperName",
			"agent.name": "testShipperName",
		})

		test.
			ExpectEOF(file...).
			WithReportOptions(reportOptions).
			Start(ctx).
			Wait()
	})

}
