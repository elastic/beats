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

func TestFilebeatDeprecated(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Minute)
	defer cancel()
	EnsureCompiled(ctx, t)

	messagePrefix := "sample test message"
	fileCount := 1
	lineCount := 1

	reportOptions := integration.ReportOptions{
		PrintLinesOnFail:  10,
		PrintConfigOnFail: false,
	}

	t.Run("check that harvesting works with deprecated input_type", func(t *testing.T) {

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		config := `
filebeat.inputs:
  - input_type: filestream
    id: "test-filestream"
    paths:
     - %s
    scan_frequency: 0.1s
output.console:
  enabled: true
`
		generator := NewPlainTextGenerator(messagePrefix)
		path, file := GenerateLogFiles(t, fileCount, lineCount, generator)
		test := NewTest(t, TestOptions{
			Config: fmt.Sprintf(config, path),
		})

		line := fmt.Sprintf("%s:%d", filepath.Base(file[0]), 1)
		test.ExpectOutput(line)
		test.ExpectOutput("DEPRECATED: input_type input config is deprecated")

		test.
			WithReportOptions(reportOptions).
			ExpectStart().
			Start(ctx).
			Wait()
	})

	t.Run("check that harvesting works with deprecated input_type", func(t *testing.T) {

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		config := `
filebeat.config.modules:
  path: ${path.config}/modules.d/*.yml
  reload.enabled: true
output.console:
  enabled: true  
`

		test := NewTest(t, TestOptions{
			Config: config,
			Args:   []string{"-E", "filebeat.prospectors=anything", "-E", "filebeat.config.prospectors=anything"},
		})

		test.ExpectOutput(`setting 'filebeat.prospectors' has been removed`)
		test.ExpectOutput(`setting 'filebeat.config.prospectors' has been removed`)

		test.
			WithReportOptions(reportOptions).
			Start(ctx).
			Wait()
	})
}
