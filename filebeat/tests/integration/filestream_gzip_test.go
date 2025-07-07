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
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/filebeat/testing/gziptest"
	"github.com/elastic/beats/v7/libbeat/tests/integration"
)

func TestFilestreamGZIPIncompleteFilesAreFullyRead(t *testing.T) {
	lines := make([]string, 0, 100)
	var content []byte
	for i := range 100 {
		l := fmt.Sprintf("%d: a log line", i)
		lines = append(lines, l)
		content = append(content, []byte(l+"\n")...)
	}
	gzData := gziptest.Compress(t, content, gziptest.CorruptNone)
	outputFilename := "output-file"
	tcs := []struct {
		name        string
		data        []byte
		restData    []byte
		initialLogs func(lofFile string) []string
		furtherLogs []string
	}{
		{
			name:     "incomplete header",
			data:     gzData[:5],
			restData: gzData[5:],
			initialLogs: func(lofFile string) []string {
				return []string{
					fmt.Sprintf("cannot create a file descriptor for an ingest target \\\"%s\\\": failed to create gzip seeker: could not create gzip reader: unexpected EOF", lofFile),
				}
			},
			furtherLogs: []string{
				"A new file %s has been found",
			},
		},
		{
			name:     "full header and incomplete data",
			data:     gzData[:len(gzData)-20],
			restData: gzData[len(gzData)-20:],
			initialLogs: func(lofFile string) []string {
				return []string{
					fmt.Sprintf("Unexpected state reading from %s; error: unexpected EOF", lofFile),
					"Error stopping filestream reader: could not close gzip reader: unexpected EOF",
				}
			},
			furtherLogs: []string{
				"File %s has been updated",
			},
		},
		{
			name:     "incomplete footer",
			data:     gzData[:len(gzData)-8],
			restData: gzData[len(gzData)-8:],
			initialLogs: func(lofFile string) []string {
				return []string{
					fmt.Sprintf("Unexpected state reading from %s; error: unexpected EOF", lofFile),
				}
			},
			furtherLogs: []string{
				"File %s has been updated",
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			filebeat := integration.NewBeat(
				t,
				"filebeat",
				"../../filebeat.test",
			)

			tempDir := filebeat.TempDir()
			logPath := filepath.Join(tempDir, "input.log.gz")

			err := os.WriteFile(logPath, tc.data, 0644)
			require.NoError(t, err)

			cfg := fmt.Sprintf(`
filebeat.inputs:
  - type: filestream
    id: "test-filestream"
    paths:
      - %s
    prospector.scanner.check_interval: 1s
    gzip_experimental: true
    rotation.external.strategy.copytruncate.suffix_regex: \.\d+(\.gz)?$
output.file:
  enabled: true
  path: %s
  filename: "%s"
logging.level: debug
`, logPath, filebeat.TempDir(), outputFilename)

			filebeat.WriteConfigFile(cfg)
			filebeat.Start()

			// wait for filebeat read the incomplete GZIP file and reach the
			// error.
			for _, want := range tc.initialLogs(logPath) {
				filebeat.WaitForLogs(
					want,
					30*time.Second,
					"Filebeat did not log: '%s'", want,
				)
			}

			// Write the rest of the file.
			f, err := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY, 0644)
			require.NoError(t, err)
			_, err = f.Write(tc.restData)
			require.NoError(t, err)
			require.NoError(t, f.Close())

			// Wait for filebeat to continue reading the file now it's fully
			// written.
			for _, log := range tc.furtherLogs {
				want := fmt.Sprintf(log, logPath)
				filebeat.WaitForLogs(
					fmt.Sprintf(log, logPath),
					30*time.Second,
					"Filebeat did not log: '%s'", want,
				)
			}

			// Ensure the file is fully read
			filebeat.WaitForLogs(fmt.Sprintf(
				"EOF has been reached. Closing. Path='%s'", logPath),
				30*time.Second,
				"Filebeat did not finish reading the log file")

			filebeat.Stop()

			pattern := filepath.Join(tempDir, outputFilename) + "-*.ndjson"
			files, err := filepath.Glob(pattern)
			require.NoError(t, err, "could not glob output file pattern")
			require.Len(t, files, 1, "expected only 1 output file")

			got, err := os.ReadFile(files[0])
			require.NoError(t, err, "could not open output file")

			// check that all lines have been published
			matchPublishedLines(t, got, lines)
		})
	}
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
