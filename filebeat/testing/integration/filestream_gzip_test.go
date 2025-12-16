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
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/klauspost/compress/gzip"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/filebeat/testing/gziptest"
	"github.com/elastic/beats/v7/libbeat/testing/integration"
)

func TestFilestreamGZIP(t *testing.T) {
	lines := make([]string, 0, 100)
	var content []byte
	for i := range 100 {
		l := fmt.Sprintf("%d: a log line", i)
		lines = append(lines, l)
		content = append(content, []byte(l+"\n")...)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	EnsureCompiled(ctx, t)

	reportOptions := integration.ReportOptions{
		PrintLinesOnFail:  1000,
		PrintConfigOnFail: true,
	}

	t.Run("integrity", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		tcs := []struct {
			name       string
			corruption gziptest.Corruption
		}{
			{
				name:       "CRC",
				corruption: gziptest.CorruptCRC,
			},
			{
				name:       "size",
				corruption: gziptest.CorruptSize,
			},
		}

		for _, tc := range tcs {
			t.Run(tc.name, func(t *testing.T) {

				tempDir := t.TempDir()
				outputFilename := "output-file"
				logPath := filepath.Join(tempDir, "input.log.gz")

				corruptedGZIP := gziptest.Compress(t, content, tc.corruption)
				err := os.WriteFile(logPath, corruptedGZIP, 0644)
				require.NoError(t, err)

				config := fmt.Sprintf(`
filebeat.inputs:
  - type: filestream
    id: "test-filestream"
    paths:
      - %s
    compression: auto
output.file:
  enabled: true
  path: %s
  filename: "%s"
`, logPath, tempDir, outputFilename)

				fbt := NewTest(t, TestOptions{
					Config: config,
				})

				fbt.
					ExpectEOF(logPath).
					WithReportOptions(reportOptions).
					ExpectStart().
					// CRC and size validation return the same error :/
					ExpectOutput(fmt.Sprintf(
						"Unexpected state reading from %s; error: gzip: invalid checksum",
						logPath)).
					// Wait for events to be published
					ExpectOutput("ackloop: return ack to broker loop:100").
					Start(ctx).
					Wait()

				pattern := filepath.Join(tempDir, outputFilename) + "-*.ndjson"
				files, err := filepath.Glob(pattern)
				require.NoError(t, err, "could not glob output file pattern")
				require.Len(t, files, 1, "expected only 1 output file")

				got, err := os.ReadFile(files[0])
				require.NoError(t, err, "could not open output file")

				matchPublishedLines(t, got, lines)
			})
		}
	})
	t.Run("GzipExperimentalDeprecationWarning", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		td := t.TempDir()
		logPath := filepath.Join(td, "input.log.gz")

		var gzBuff bytes.Buffer
		gw := gzip.NewWriter(&gzBuff)
		_, err := gw.Write([]byte("hello world"))
		require.NoError(t, err)
		require.NoError(t, gw.Close())

		err = os.WriteFile(logPath, gzBuff.Bytes(), 0644)
		require.NoError(t, err)

		config := fmt.Sprintf(`
filebeat.inputs:
  - type: filestream
    id: "test-filestream"
    paths:
      - %s
    gzip_experimental: true
output.console:
  enabled: true
`, logPath)

		test := NewTest(t, TestOptions{
			Config: config,
		})

		test.
			WithReportOptions(reportOptions).
			ExpectStart().
			ExpectOutput(
				"'gzip_experimental' is deprecated and ignored, set 'compression' instead").
			Start(ctx).
			Wait()
	})

	t.Run("GzipExperimentalDeprecationWarningWithCompressionSet", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		td := t.TempDir()
		logPath := filepath.Join(td, "input.log.gz")

		var gzBuff bytes.Buffer
		gw := gzip.NewWriter(&gzBuff)
		_, err := gw.Write([]byte("hello world"))
		require.NoError(t, err)
		require.NoError(t, gw.Close())

		err = os.WriteFile(logPath, gzBuff.Bytes(), 0644)
		require.NoError(t, err)

		config := fmt.Sprintf(`
filebeat.inputs:
  - type: filestream
    id: "test-filestream"
    paths:
      - %s
    compression: auto
    gzip_experimental: true
output.console:
  enabled: true
`, logPath)

		test := NewTest(t, TestOptions{
			Config: config,
		})

		test.
			WithReportOptions(reportOptions).
			ExpectStart().
			ExpectOutput(
				"'gzip_experimental' is deprecated and ignored. 'compression' is set, using it instead").
			Start(ctx).
			Wait()
	})
}

func matchPublishedLines(t *testing.T, got []byte, want []string) {
	gotLines := strings.Split(strings.TrimSpace(string(got)), "\n")
	assert.Equal(t, len(want), len(gotLines), "unexpected number of events")

	linesToMatch := min(len(want), len(gotLines))

	var unmatched []int
	for i := range linesToMatch {
		if !strings.Contains(
			gotLines[i],
			fmt.Sprintf(`"message":"%s"`, want[i])) {
			unmatched = append(unmatched, i)
		}
	}
	if len(unmatched) > 0 {
		t.Logf("\n\t%d lines not matched on the output:", len(unmatched))
		for _, i := range unmatched {
			fmt.Printf("\t\t\tgot: %s\n", gotLines[i])
			fmt.Printf("\t\t\twant containing: '%s'\n", want[i])
		}
	}
	notFound := len(want) - len(gotLines)
	if notFound > 0 {
		t.Logf("\n\t%d lines not found on the output:", notFound)
		fmt.Printf("\t\t\t%s", strings.Join(want[len(gotLines):], "\n\t\t\t"))
		fmt.Print("\n\n")
	}
	if notFound < 0 { // extra lines in the output
		t.Logf("\n\t%d extra lines found on the output:", notFound*-1)
		fmt.Printf("\t\t\t%s", strings.Join(gotLines[len(want):], "\n\t\t\t"))
		fmt.Print("\n\n")
	}
}
