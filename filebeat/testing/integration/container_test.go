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

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/testing/integration"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestContainerInput(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Minute)
	defer cancel()
	EnsureCompiled(ctx, t)

	reportOptions := integration.ReportOptions{
		PrintLinesOnFail:  100,
		PrintConfigOnFail: false,
	}

	config := `
filebeat.inputs:
- type: filestream
  id: test-container
  file_identity.native: ~
  prospector.scanner.fingerprint.enabled: false	
  paths:
  - %s
  parsers:
  - container:
      stream: stdout
output.console:
  enabled: true
filebeat.registry.flush: 0s
queue.mem.flush.timeout: 0s
`

	// get current working director
	path, err := os.Getwd()
	require.NoError(t, err)

	t.Run("test container input", func(t *testing.T) {

		dockerLogPath := filepath.Join(path, "files", "logs", "docker.log")
		test := NewTest(t, TestOptions{
			Config: fmt.Sprintf(config, dockerLogPath),
		})

		test.ExpectJSONFields(mapstr.M{
			"message":    "Moving binaries to host...\n",
			"stream":     "stdout",
			"input.type": "filestream",
		})

		test.
			ExpectEOF(dockerLogPath).
			WithReportOptions(reportOptions).
			ExpectStart().
			Start(ctx).
			Wait()
	})

	t.Run(" Test container input with CRI format", func(t *testing.T) {
		criLogPath := filepath.Join(path, "files", "logs", "cri.log")
		test := NewTest(t, TestOptions{
			Config: fmt.Sprintf(config, criLogPath),
		})

		test.ExpectJSONFields(mapstr.M{
			"stream":     "stdout",
			"input.type": "filestream",
		})

		test.
			ExpectEOF(criLogPath).
			WithReportOptions(reportOptions).
			ExpectStart().
			Start(ctx).
			Wait()
	})

	t.Run(" Test container input properly updates registry offset in case of unparsable lines", func(t *testing.T) {
		dockerCorruptedPath := filepath.Join(path, "files", "logs", "docker_corrupted.log")
		test := NewTest(t, TestOptions{
			Config: fmt.Sprintf(config, dockerCorruptedPath),
		})

		test.ExpectJSONFields(mapstr.M{
			"message":    "Moving binaries to host...\n",
			"stream":     "stdout",
			"input.type": "filestream",
		})

		//expect parse line error
		test.ExpectOutput("Parse line error")

		test.
			ExpectEOF(dockerCorruptedPath).
			WithReportOptions(reportOptions).
			ExpectStart().
			Start(ctx).
			Wait()

		registryLogFile := filepath.Join(test.GetTempDir(), "data/registry/filebeat/log.json")

		// bytes of healthy file are 2244 so for the corrupted one should
		// be 2244-1=2243 since we removed one character
		require.Eventually(
			t,
			func() bool {
				return AssertLastOffset(t, registryLogFile, 2243)
			},
			20*time.Second,
			250*time.Millisecond, "did not find the expected registry offset of 2243")
	})
}
