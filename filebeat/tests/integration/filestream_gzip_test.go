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
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/filebeat/testing/gziptest"
	"github.com/elastic/beats/v7/libbeat/tests/integration"
	"github.com/elastic/elastic-agent-libs/iobuf"
)

// TestFilestreamGZIPIncompleteFilesAreFullyRead ensures filestream correctly
// handles GZIP files if it finds the file while the file is being written to
// disks.
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
		withStop    bool
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
			// This test verifies that Filebeat can stop, update its registry,
			// and later resume reading a GZIP file from the correct offset.
			//
			// Stopping Filebeat while it is still reading a file is hard to do
			// deterministically: once Filebeat reaches the end of a
			// fully-written GZIP file it hits EOF and stops reading, so we
			// would have to rely on timing tricks that make the test
			// flaky, especially in CI.
			//
			// Instead, we omit the GZIP footer. The missing footer prevents
			// Filebeat from marking the file as fully ingested. We stop
			// Filebeat, append the footer (completing the file), restart
			// Filebeat, and assert that it resumes reading from the previous
			// offset.
			name:     "full header and incomplete data: stop filebeat in the middle",
			data:     gzData[:len(gzData)-20],
			restData: gzData[len(gzData)-20:],
			initialLogs: func(lofFile string) []string {
				return []string{
					fmt.Sprintf("Unexpected state reading from %s; error: unexpected EOF", lofFile),
					"Error stopping filestream reader: could not close gzip reader: unexpected EOF",
				}
			},
			withStop: true,
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
    compression: auto
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
				filebeat.WaitLogsContainsFromBeginning(
					want,
					30*time.Second,
					"Filebeat did not log: '%s'", want,
				)
			}

			if tc.withStop {
				filebeat.Stop()
			}
			// Write the rest of the file.
			f, err := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY, 0644)
			require.NoError(t, err)
			_, err = f.Write(tc.restData)
			require.NoError(t, err)
			require.NoError(t, f.Close())

			if tc.withStop {
				filebeat.Start()
			}
			// Wait for filebeat to continue reading the file now it's fully
			// written.
			for _, log := range tc.furtherLogs {
				want := fmt.Sprintf(log, logPath)
				filebeat.WaitLogsContainsFromBeginning(
					fmt.Sprintf(log, logPath),
					30*time.Second,
					"Filebeat did not log: '%s'", want,
				)
			}

			// Ensure the file is fully read
			filebeat.WaitLogsContainsFromBeginning(fmt.Sprintf(
				"EOF has been reached. Closing. Path='%s'", logPath),
				30*time.Second,
				"Filebeat did not finish reading the log file")

			filebeat.Stop()

			matchPublishedLinesFromFile(t,
				filepath.Join(tempDir, outputFilename), lines)
		})
	}
}

// TestFilestreamGZIPCompressionNotSetWithGZIPFile ensures filestream reads a
// gzip file as plain text (raw bytes) when compression is not set or empty.
func TestFilestreamGZIPCompressionNotSetWithGZIPFile(t *testing.T) {
	var plainContent []byte
	for i := range 500 {
		plainContent = append(plainContent, []byte(fmt.Sprintf("%d: a log line\n", i))...)
	}
	gzContent := gziptest.Compress(t, plainContent, gziptest.CorruptNone)

	tcs := []struct {
		name        string
		compression string
	}{
		{
			name:        "compression not set",
			compression: "",
		},
		{
			name:        "compression empty string",
			compression: "compression: \"\"",
		},
		{
			name:        "compression empty string with gzip_experimental set",
			compression: "compression: \"\"\n    gzip_experimental: true",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			filebeat := integration.NewBeat(
				t,
				"filebeat",
				"../../filebeat.test",
			)
			workDir := filebeat.TempDir()

			logFilepath := filepath.Join(workDir, "log.gz")
			cfg := fmt.Sprintf(`
filebeat.inputs:
  - type: filestream
    id: "test-no-compression-gzip-file"
    paths:
      - %s
    %s
path.home: %s
output.file:
  enabled: true
  path: %s
  filename: "output"
logging.level: debug
`, logFilepath, tc.compression, workDir, workDir)
			require.NoError(t, os.WriteFile(logFilepath, gzContent, 0644))

			filebeat.WriteConfigFile(cfg)
			filebeat.Start()

			eofLog := fmt.Sprintf("End of file reached: %s; Backoff now.", logFilepath)
			filebeat.WaitLogsContains(
				eofLog,
				30*time.Second,
				"Filebeat did not reach EOF for '%s'",
				logFilepath,
			)

			filebeat.Stop()

			type event struct {
				Message string `json:"message"`
			}
			events := integration.GetEventsFromFileOutput[event](filebeat, 0, true)
			require.Len(t, events, 1, "expected one event")

			// assert the message is not the decompressed content
			assert.NotEqual(t, "0: a log line", events[0].Message,
				"message should be raw gzip bytes, not decompressed content")
		})
	}
}

// TestFilestreamGZIPCompressioAndInvalidFileIdentity
func TestFilestreamGZIPCompressionAutoFileIdentityNativeErrors(t *testing.T) {
	tcs := []struct {
		name         string
		compression  string
		fileIdentity string
	}{
		{
			name:         "auto compression with native file_identity",
			compression:  "auto",
			fileIdentity: "file_identity.native: ~",
		},
		{
			name:         "gzip compression with native file_identity",
			compression:  "gzip",
			fileIdentity: "file_identity.native: ~",
		},
		{
			name:         "auto compression with path file_identity",
			compression:  "auto",
			fileIdentity: "file_identity.path: ~",
		},
		{
			name:         "gzip compression with path file_identity",
			compression:  "gzip",
			fileIdentity: "file_identity.path: ~",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			filebeat := integration.NewBeat(
				t,
				"filebeat",
				"../../filebeat.test",
			)
			workDir := filebeat.TempDir()

			logFilepath := filepath.Join(workDir, "log.gz")
			cfg := fmt.Sprintf(`
filebeat.inputs:
  - type: filestream
    id: "test-compression-file-identity-error"
    paths:
      - %s
    compression: %s
    %s
path.home: %s
output.file:
  enabled: true
  path: %s
  filename: "output"
logging.level: debug
`, logFilepath, tc.compression, tc.fileIdentity, workDir, workDir)

			filebeat.WriteConfigFile(cfg)
			filebeat.Start()

			filebeat.WaitLogsContains(
				fmt.Sprintf("compression='%s' requires 'file_identity' to be 'fingerprint'",
					tc.compression),
				30*time.Second,
				"Filebeat did not log expected config validation error",
			)
			filebeat.Stop()
		})
	}
}

// TestFilestreamGZIPCompressionOnPlainFile ensures filestream correctly handles
// the case when compression: gzip is explicitly set but the file is a plain
// text file (not gzip compressed). It should fail to read the file and log an
// appropriate error.
func TestFilestreamGZIPCompressionGZIPWithPlainFile(t *testing.T) {
	var content []byte
	for i := range 100 {
		content = append(content, []byte(fmt.Sprintf("%d: a log line\n", i))...)
	}

	filebeat := integration.NewBeat(
		t,
		"filebeat",
		"../../filebeat.test",
	)
	workDir := filebeat.TempDir()

	// Write a plain text file (not gzip compressed)
	logFilepath := filepath.Join(workDir, "log.log")
	cfg := fmt.Sprintf(`
filebeat.inputs:
  - type: filestream
    id: "test-gzip-on-plain-file"
    paths:
      - %s
    compression: gzip
path.home: %s
output.file:
  enabled: true
  path: %s
  filename: "output"
logging.level: debug
`, logFilepath, workDir, workDir)

	// Write plain text content (not compressed)
	require.NoError(t, os.WriteFile(logFilepath, content, 0644))

	filebeat.WriteConfigFile(cfg)
	filebeat.Start()

	filebeat.WaitLogsContains(
		fmt.Sprintf("cannot create a file descriptor for an ingest target \\\"%s\\\": failed to create gzip seeker: could not create gzip reader: gzip: invalid header", logFilepath),
		30*time.Second,
		"Filebeat did not log expected gzip error for plain file '%s'",
		logFilepath,
	)

	filebeat.Stop()

	// output file isn't created because no event has been published. Thus,
	// check it manually.
	path := filepath.Join(filebeat.TempDir(), "output-*.ndjson")
	files, err := filepath.Glob(path)
	require.NoError(t, err, "failed to glob output files")
	require.Len(t, files, 0, "expected no output file to be created as no event should have been published")
}

// TestFilestreamGZIPEOF ensures, for GZIP files, filestream:
//   - sets EOF on the registry when it reached the end of the file
//   - if EOF is set, it does not read the file again.
func TestFilestreamGZIPEOF(t *testing.T) {
	var content []byte
	for i := range 100 {
		content = append(content, []byte(fmt.Sprintf("%d: a log line\n", i))...)
	}

	filebeat := integration.NewBeat(
		t,
		"filebeat",
		"../../filebeat.test",
	)
	workDir := filebeat.TempDir()

	logFilepath := filepath.Join(workDir, "log.gz")
	cfg := fmt.Sprintf(`
filebeat.inputs:
  - type: filestream
    id: "test-gzip-eof"
    paths:
      - %s
    compression: auto
path.home: %s
filebeat.registry.flush: 1s
output.discard:
  enabled: true
logging.level: debug
`, logFilepath, workDir)

	compressedContent := gziptest.Compress(t, content, gziptest.CorruptNone)
	require.NoError(t, os.WriteFile(logFilepath, compressedContent, 0644))

	filebeat.WriteConfigFile(cfg)
	filebeat.Start()

	filebeat.WaitLogsContainsFromBeginning(
		fmt.Sprintf("EOF has been reached. Closing. Path='%s'", logFilepath),
		30*time.Second,
		"Filebeat did not reach EOF. Did not find log [%s]",
		logFilepath,
	)
	filebeat.Stop()

	registryLogFile := filepath.Join(workDir,
		"data", "registry", "filebeat", "log.json")
	entries, _ := readFilestreamRegistryLog(t, registryLogFile)

	var lastEntry *registryEntry
	for i := range entries {
		entry := &entries[i]
		if entry.Filename == logFilepath {
			lastEntry = entry
		}
	}
	require.NotNil(t, lastEntry,
		"state for log file not found in registry for %s", logFilepath)

	// ================ Verify offset and EOF are correctly set ================
	assert.Equal(t, len(content), lastEntry.Offset, "offset is not correct")
	assert.True(t, lastEntry.EOF, "EOF is not true")

	filebeat.Start()
	wantLog := fmt.Sprintf("GZIP file already read to EOF, not reading it again, file name '%s'", logFilepath)
	filebeat.WaitLogsContainsFromBeginning(
		wantLog,
		30*time.Second,
		"Filebeat did find log '%s'",
		wantLog,
	)

	// =============== Verify file read to EOF isn't read again ================
	gotEntries, _ := readFilestreamRegistryLog(t, registryLogFile)
	// when the harvester starts, before attempting to open the log file, it
	// updates the registry, thus reading it again will bring one more entry
	assert.Equal(t, entries, gotEntries[:len(gotEntries)-1],
		"the registry for should not have changed")
	// ensure the new entry is the same as the previous.
	assert.Equal(t, entries[len(entries)-1], gotEntries[len(gotEntries)-1],
		"expected the last entry of the registry to be the same as previous entry")
}

// TestFilestreamGZIPConcatenatedFiles verifies that filestream can read a
// gzip file produced by appending multiple gzip-compressed files. Per the gzip
// spec, decompressing this concatenation must yield the same bytes as first
// concatenating the plain data of both files and then gzipping the result.
func TestFilestreamGZIPConcatenatedFiles(t *testing.T) {
	lines := make([]string, 0, 200)
	var content []byte
	for i := range 100 {
		l := fmt.Sprintf("%d: 1st file log line", i)
		lines = append(lines, l)
		content = append(content, []byte(l+"\n")...)
	}
	gzData1 := gziptest.Compress(t, content, gziptest.CorruptNone)

	content = nil
	for i := range 100 {
		l := fmt.Sprintf("%d: 2nd file log line", i)
		lines = append(lines, l)
		content = append(content, []byte(l+"\n")...)
	}
	gzData2 := gziptest.Compress(t, content, gziptest.CorruptNone)

	filebeat := integration.NewBeat(
		t,
		"filebeat",
		"../../filebeat.test",
	)
	tempDir := filebeat.TempDir()
	logPath := filepath.Join(tempDir, "2gzipFilesConcatenated.gz")
	outputFilename := "output-file"

	err := os.WriteFile(
		logPath,
		append(gzData1, gzData2...), 0644)
	require.NoError(t, err, "could not write gzip file to disk")

	cfg := fmt.Sprintf(`
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
logging.level: debug
`, logPath, filebeat.TempDir(), outputFilename)

	filebeat.WriteConfigFile(cfg)
	filebeat.Start()

	filebeat.WaitLogsContainsFromBeginning(
		fmt.Sprintf("EOF has been reached. Closing. Path='%s'", logPath),
		30*time.Second,
		"Filebeat did not reach EOF. Did not find log [%s]",
		logPath,
	)
	filebeat.Stop()

	matchPublishedLinesFromFile(t,
		filepath.Join(tempDir, outputFilename), lines)
}

func TestFilestreamGZIPMetrics(t *testing.T) {
	lines := make([]string, 0, 200)
	var dataPlain []byte
	for i := range 100 {
		l := fmt.Sprintf("%d: %s", i, strings.Repeat("a", 100))
		lines = append(lines, l)
		dataPlain = append(dataPlain, []byte(l+"\n")...)
	}

	var gzPlainData []byte
	for i := range 100 {
		l := fmt.Sprintf("%d: %s", i, strings.Repeat("b", 100))
		lines = append(lines, l)
		gzPlainData = append(gzPlainData, []byte(l+"\n")...)
	}
	dataGZ := gziptest.Compress(t, gzPlainData, gziptest.CorruptNone)

	// sanity check
	require.Equal(t, len(dataPlain), len(gzPlainData),
		"data for both plain and gzip file should have the same size")

	filebeat := integration.NewBeat(
		t,
		"filebeat",
		"../../filebeat.test",
	)
	tempDir := filebeat.TempDir()
	t.Log("tempdir:", tempDir)
	logPathBase := filepath.Join(tempDir, "log")
	logPathPlain := logPathBase
	logPathGZ := logPathBase + ".gz"
	outputFilePattern := "output-file"

	err := os.WriteFile(
		logPathGZ, dataGZ, 0644)
	require.NoError(t, err, "could not write gzip file to disk")
	err = os.WriteFile(
		logPathPlain, dataPlain, 0644)
	require.NoError(t, err, "could not write plain file to disk")

	cfg := fmt.Sprintf(`
http:
  enabled: true
  port: 4242
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
logging.level: debug
`, logPathBase+"*", filebeat.TempDir(), outputFilePattern)

	filebeat.WriteConfigFile(cfg)
	filebeat.Start()

	eofLine := fmt.Sprintf("End of file reached: %s; Backoff now.", logPathPlain)
	filebeat.WaitLogsContainsFromBeginning(
		eofLine,
		30*time.Second,
		"Filebeat did not reach EOF. Did not find log [%s]",
		eofLine,
	)

	eofLine = fmt.Sprintf("EOF has been reached. Closing. Path='%s'", logPathGZ)
	filebeat.WaitLogsContainsFromBeginning(
		eofLine,
		time.Minute,
		"Filebeat did not reach EOF. Did not find log [%s]",
		eofLine,
	)

	//nolint:noctx // It's fine on a test
	resp, err := http.Get("http://localhost:4242/inputs/")
	require.NoError(t, err, "failed to get input metrics")
	defer resp.Body.Close()

	filebeat.Stop()

	matchPublishedLinesFromFile(t,
		filepath.Join(tempDir, outputFilePattern), lines)

	// ============================ assert metrics =============================

	type Histogram struct {
		Count  int     `json:"count"`
		Max    float64 `json:"max"`
		Mean   float64 `json:"mean"`
		Median float64 `json:"median"`
		Min    float64 `json:"min"`
		P75    float64 `json:"p75"`
		P95    float64 `json:"p95"`
		P99    float64 `json:"p99"`
		P999   float64 `json:"p999"`
		Stddev float64 `json:"stddev"`
	}

	type ProcessingTime struct {
		Histogram Histogram `json:"histogram"`
	}

	type InputMetrics struct {
		BytesProcessedTotal      int            `json:"bytes_processed_total"`
		EventsProcessedTotal     int            `json:"events_processed_total"`
		FilesClosedTotal         int            `json:"files_closed_total"`
		FilesOpenedTotal         int            `json:"files_opened_total"`
		ProcessingTime           ProcessingTime `json:"processing_time"`
		GzipBytesProcessedTotal  int            `json:"gzip_bytes_processed_total"`
		GzipEventsProcessedTotal int            `json:"gzip_events_processed_total"`
		GzipFilesClosedTotal     int            `json:"gzip_files_closed_total"`
		GzipFilesOpenedTotal     int            `json:"gzip_files_opened_total"`
		GzipProcessingTime       ProcessingTime `json:"gzip_processing_time"`
	}

	var metrics []InputMetrics
	body, err := iobuf.ReadAll(resp.Body)
	require.NoError(t, err, "could not read response body")
	err = json.Unmarshal(body, &metrics)
	require.NoError(t, err, "failed to Unmarshal JSON response")
	defer func() {
		if t.Failed() {
			t.Log("raw input metrics response:", string(body))
		}
	}()
	require.Len(t, metrics, 1, "expected exactly one input metric")
	metric := metrics[0]
	totalHist := metric.ProcessingTime.Histogram

	// Assert GZIP metrics
	assert.Equal(t, 100, metric.GzipEventsProcessedTotal,
		"gzip events processed should be 100")
	assert.Equal(t, len(gzPlainData), metric.GzipBytesProcessedTotal,
		"gzip bytes processed should be greater than 0")
	assert.Equal(t, 1, metric.GzipFilesClosedTotal,
		"gzip files closed should be 1")
	assert.Equal(t, 1, metric.GzipFilesOpenedTotal,
		"gzip files opened should be 1")
	assert.Equal(t, 100, metric.GzipProcessingTime.Histogram.Count,
		"gzip processing time histogram count should be 100")

	// Assert GZIP metrics are half of total metrics
	// (since we have 100 lines in gzip file vs 200 total lines)
	assert.Equal(t, metric.EventsProcessedTotal/2, metric.GzipEventsProcessedTotal,
		"gzip events should be half of total events")
	assert.Equal(t, totalHist.Count/2, metric.GzipProcessingTime.Histogram.Count,
		"gzip processing count should be half of total processing count")

	// Assert GZIP bytes processed are half of total
	assert.Equal(t, metric.BytesProcessedTotal/2, metric.GzipBytesProcessedTotal,
		"gzip bytes processed should be greater than 0")

	// File counts: we have 1 gzip file + 1 plain file = 2 total files opened
	assert.Equal(t, metric.FilesOpenedTotal, metric.GzipFilesOpenedTotal+1,
		"total files opened should be gzip files + 1 plain file")

	// =================== Assert histogram has valid values ===================
	assert.Equal(t, 200, totalHist.Count,
		"total processing time histogram count should be 200")
	assert.Greater(t, totalHist.Max, float64(0),
		"total processing time max should be greater than 0")
	assert.Greater(t, totalHist.Mean, float64(0),
		"total processing time mean should be greater than 0")
	assert.Greater(t, totalHist.Median, float64(0),
		"total processing time median should be greater than 0")
	assert.Greater(t, totalHist.Min, float64(0),
		"total processing time min should be greater than 0")
	assert.Greater(t, totalHist.P75, float64(0),
		"total processing time p75 should be greater than 0")
	assert.Greater(t, totalHist.P95, float64(0),
		"total processing time p95 should be greater than 0")
	assert.Greater(t, totalHist.P99, float64(0),
		"total processing time p99 should be greater than 0")
	assert.Greater(t, totalHist.P999, float64(0),
		"total processing time p999 should be greater than 0")
	assert.Greater(t, totalHist.Stddev, float64(0),
		"total processing time stddev should be greater than 0")

	assert.Greater(t, metric.GzipProcessingTime.Histogram.Max, float64(0),
		"gzip processing time max should be greater than 0")
	assert.Greater(t, metric.GzipProcessingTime.Histogram.Mean, float64(0),
		"gzip processing time mean should be greater than 0")
	assert.Greater(t, metric.GzipProcessingTime.Histogram.Median, float64(0),
		"gzip processing time median should be greater than 0")
	assert.Greater(t, metric.GzipProcessingTime.Histogram.Min, float64(0),
		"gzip processing time min should be greater than 0")
	assert.Greater(t, metric.GzipProcessingTime.Histogram.P75, float64(0),
		"gzip processing time p75 should be greater than 0")
	assert.Greater(t, metric.GzipProcessingTime.Histogram.P95, float64(0),
		"gzip processing time p95 should be greater than 0")
	assert.Greater(t, metric.GzipProcessingTime.Histogram.P99, float64(0),
		"gzip processing time p99 should be greater than 0")
	assert.Greater(t, metric.GzipProcessingTime.Histogram.P999, float64(0),
		"gzip processing time p999 should be greater than 0")
	assert.Greater(t, metric.GzipProcessingTime.Histogram.Stddev, float64(0),
		"gzip processing time stddev should be greater than 0")
}

func TestFilestreamGZIPFingerprintOnDecompressedData(t *testing.T) {
	lines := make([]string, 0, 100)
	var dataPlain []byte
	for i := range 100 {
		l := fmt.Sprintf("%d: 1st file log line", i)
		lines = append(lines, l)
		dataPlain = append(dataPlain, []byte(l+"\n")...)
	}
	dataGZ := gziptest.Compress(t, dataPlain, gziptest.CorruptNone)

	filebeat := integration.NewBeat(
		t,
		"filebeat",
		"../../filebeat.test",
	)
	tempDir := filebeat.TempDir()
	logFileBaseName := "plain.log"
	logPathPlain := filepath.Join(tempDir, logFileBaseName)
	logPathGZ := filepath.Join(tempDir, logFileBaseName+".gz")

	err := os.WriteFile(logPathPlain, dataPlain, 0644)
	require.NoError(t, err, "could not write gzip file to disk")

	outputFilename := "output-file"
	cfg := fmt.Sprintf(`
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
logging.level: debug
`, logPathPlain+"*", filebeat.TempDir(), outputFilename)

	filebeat.WriteConfigFile(cfg)
	filebeat.Start()

	eofLine := fmt.Sprintf("End of file reached: %s; Backoff now.", logPathPlain)
	filebeat.WaitLogsContainsFromBeginning(
		eofLine,
		30*time.Second,
		"Filebeat did not reach EOF. Did not find log [%s]",
		eofLine,
	)

	// 1st file is ingested, add the GZ file
	err = os.WriteFile(logPathGZ, dataGZ, 0644)
	require.NoError(t, err, "could not write gzip file to disk")

	// wait filebeat to pick up the file and see it's the same as the plain file.
	wantLine := fmt.Sprintf("\\\"%s\\\" points to an already known ingest target \\\"%s\\\" [e64ff2da367b082e1dcc38ec48215bff55925bd408f718f107e50ecf426fe3c3==e64ff2da367b082e1dcc38ec48215bff55925bd408f718f107e50ecf426fe3c3]. Skipping",
		logPathGZ, logPathPlain)
	filebeat.WaitLogsContainsFromBeginning(
		wantLine,
		30*time.Second,
		"Did not find log '%s'",
		wantLine,
	)

	filebeat.Stop()
	matchPublishedLinesFromFile(t,
		filepath.Join(tempDir, outputFilename), lines)
}

func TestFilestreamGZIPLogRotation_1_rotated_file(t *testing.T) {
	want1stLines := make([]string, 0, 100)
	want2ndLines := make([]string, 0, 150)
	var dataPlain1stHalf []byte
	for i := range 100 {
		l := fmt.Sprintf("%d: 1st 1/2 file before rotation log line", i)
		want1stLines = append(want1stLines, l)
		dataPlain1stHalf = append(dataPlain1stHalf, []byte(l+"\n")...)
	}
	var dataPlain2ndHalf []byte
	for i := range 100 {
		l := fmt.Sprintf("%d: 2nd 1/2 file after rotation log line", i)
		want2ndLines = append(want2ndLines, l)
		dataPlain2ndHalf = append(dataPlain2ndHalf, []byte(l+"\n")...)
	}
	dataGZ := gziptest.Compress(t,
		append(dataPlain1stHalf, dataPlain2ndHalf...), gziptest.CorruptNone)

	var dataPlainAfterRotation []byte
	for i := range 50 { // ensure it's smaller than the original
		l := fmt.Sprintf("%d: new logs after rotation", i)
		want2ndLines = append(want2ndLines, l)
		dataPlainAfterRotation = append(dataPlainAfterRotation, []byte(l+"\n")...)
	}

	filebeat := integration.NewBeat(
		t,
		"filebeat",
		"../../filebeat.test",
	)
	tempDir := filebeat.TempDir()
	logFileBaseName := "plain.log"
	logPathPlain := filepath.Join(tempDir, logFileBaseName)
	logPathGZ := filepath.Join(tempDir, logFileBaseName+"1.gz")

	// 1st half of the file to simulate the rotation before filebeat finishes
	// reading the file
	err := os.WriteFile(logPathPlain, dataPlain1stHalf, 0644)
	require.NoError(t, err, "could not write gzip file to disk")

	outputFilePattern := "output-file"
	cfg := fmt.Sprintf(`
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
logging.level: debug
`, logPathPlain+"*", filebeat.TempDir(), outputFilePattern)

	filebeat.WriteConfigFile(cfg)
	filebeat.Start()

	eofLine := fmt.Sprintf("End of file reached: %s; Backoff now.", logPathPlain)
	filebeat.WaitLogsContainsFromBeginning(
		eofLine,
		30*time.Second,
		"Filebeat did not reach EOF. Did not find log [%s]",
		eofLine,
	)
	// 1st file is ingested, stop filebeat and do the log rotation
	filebeat.Stop()

	// rotate the plain file "with data not yet read"
	err = os.WriteFile(logPathGZ, dataGZ, 0644)
	require.NoError(t, err, "could not write gzip file to disk")

	// truncate the original file and add new data
	err = os.WriteFile(logPathPlain, dataPlainAfterRotation, 0644)
	require.NoError(t, err, "could not truncate original log file and add new data")

	filebeat.Start()

	// Wait filebeat to finish the gzipped file
	eofLog := fmt.Sprintf("EOF has been reached. Closing. Path='%s'", logPathGZ)
	filebeat.WaitLogsContainsFromBeginning(
		eofLog,
		30*time.Second,
		"Filebeat did not reach EOF. Did not find log [%s]",
		eofLog,
	)

	// Wait filebeat to finish the original file with new content
	eofLine = fmt.Sprintf("End of file reached: %s; Backoff now.", logPathPlain)
	filebeat.WaitLogsContainsFromBeginning(
		eofLine,
		30*time.Second,
		"Filebeat did not reach EOF. Did not find log [%s]",
		eofLine,
	)

	filebeat.Stop()

	// So far so good. Now check the output

	globPattern := outputFilePattern + "-*.ndjson"
	files, err := filepath.Glob(filepath.Join(tempDir, globPattern))
	require.NoError(t, err, "could not glob output file pattern")
	require.Lenf(t, files, 2,
		"expected only 2 output files. Glob pattern '%s'", globPattern)

	slices.SortFunc(files, func(a, b string) int {
		if len(a) < len(b) {
			return -1
		}
		if len(a) > len(b) {
			return 1
		}
		if len(a) == len(b) {
			return 0
		}

		panic("unreachable")
	})

	got, err := os.ReadFile(files[0])
	require.NoError(t, err, "could not open output file")
	// 1st file: check that all lines have been published
	matchPublishedLines(t, got, want1stLines)

	got, err = os.ReadFile(files[1])
	require.NoError(t, err, "could not open output file")
	matchPublishedLines(t, got, want2ndLines)
}

func TestFilestreamGZIPLogRotation_2_rotated_files(t *testing.T) {
	want1stLines := make([]string, 0, 100)
	want2ndLines := make([]string, 0, 150)
	var dataPlain1stHalf []byte
	for i := range 100 {
		l := fmt.Sprintf("%d: 1st 1/2 file before rotation log line", i)
		want1stLines = append(want1stLines, l)
		dataPlain1stHalf = append(dataPlain1stHalf, []byte(l+"\n")...)
	}
	var dataPlain2ndHalf []byte
	for i := range 100 {
		l := fmt.Sprintf("%d: 2nd 1/2 file after rotation log line", i)
		want2ndLines = append(want2ndLines, l)
		dataPlain2ndHalf = append(dataPlain2ndHalf, []byte(l+"\n")...)
	}

	var dataPlainLatestRotation []byte
	for i := range 50 { // ensure it's smaller than the original
		l := fmt.Sprintf("%d: latest rotated file", i)
		want2ndLines = append(want2ndLines, l)
		dataPlainLatestRotation = append(dataPlainLatestRotation, []byte(l+"\n")...)
	}

	var dataPlainNewActive []byte
	for i := range 50 { // ensure it's smaller than the original
		l := fmt.Sprintf("%d: new plain file after truncation", i)
		want2ndLines = append(want2ndLines, l)
		dataPlainNewActive = append(dataPlainNewActive, []byte(l+"\n")...)
	}

	dataGZOldestRotation := gziptest.Compress(t,
		append(dataPlain1stHalf, dataPlain2ndHalf...), gziptest.CorruptNone)
	dataGZLatestRotation := gziptest.Compress(t, dataPlainLatestRotation, gziptest.CorruptNone)

	filebeat := integration.NewBeat(
		t,
		"filebeat",
		"../../filebeat.test",
	)
	tempDir := filebeat.TempDir()
	logFileBaseName := "plain.log"
	logPathActive := filepath.Join(tempDir, logFileBaseName)
	logPathLatestRotation := filepath.Join(tempDir, logFileBaseName+".1.gz")
	logPathGZOldestRotation := filepath.Join(tempDir, logFileBaseName+".2.gz")

	// 1st half of the file to simulate the rotation before filebeat finishes
	// reading the file
	err := os.WriteFile(logPathActive, dataPlain1stHalf, 0644)
	require.NoError(t, err, "could not write gzip file to disk")

	outputFilePattern := "output-file"
	cfg := fmt.Sprintf(`
filebeat.inputs:
  - type: filestream
    id: "test-filestream"
    paths:
      - %s
    compression: auto
    #rotation.external.strategy.copytruncate.suffix_regex: \.\d+(\.gz)?$
output.file:
  enabled: true
  path: %s
  filename: "%s"
logging.level: debug
`, logPathActive+"*", filebeat.TempDir(), outputFilePattern)

	filebeat.WriteConfigFile(cfg)
	filebeat.Start()

	eofLine := fmt.Sprintf("End of file reached: %s; Backoff now.", logPathActive)
	filebeat.WaitLogsContainsFromBeginning(
		eofLine,
		30*time.Second,
		"Filebeat did not reach EOF. Did not find log [%s]",
		eofLine,
	)
	// 1st file is ingested, stop filebeat and do the log rotation
	filebeat.Stop()

	// rotate the plain file "with data not yet read"
	err = os.WriteFile(logPathGZOldestRotation, dataGZOldestRotation, 0644)
	require.NoError(t, err, "could not write gzip file to disk")

	// add new rotated file never seen by filebeat
	err = os.WriteFile(logPathLatestRotation, dataGZLatestRotation, 0644)
	require.NoError(t, err, "could not write gzip file to disk")

	// truncate the original file and add new data
	err = os.WriteFile(logPathActive, dataPlainNewActive, 0644)
	require.NoError(t, err, "could not truncate original log file and add new data")

	filebeat.Start()

	// Wait filebeat to finish the gzipped files
	eofLog := fmt.Sprintf("EOF has been reached. Closing. Path='%s'", logPathLatestRotation)
	filebeat.WaitLogsContainsFromBeginning(
		eofLog,
		30*time.Second,
		"Filebeat did not reach EOF. Did not find log [%s]",
		eofLog,
	)

	eofLog = fmt.Sprintf("EOF has been reached. Closing. Path='%s'", logPathGZOldestRotation)
	filebeat.WaitLogsContainsFromBeginning(
		eofLog,
		30*time.Second,
		"Filebeat did not reach EOF. Did not find log [%s]",
		eofLog,
	)

	// Wait filebeat to finish the original file with new content
	eofLine = fmt.Sprintf("End of file reached: %s; Backoff now.", logPathActive)
	filebeat.WaitLogsContainsFromBeginning(
		eofLine,
		30*time.Second,
		"Filebeat did not reach EOF. Did not find log [%s]",
		eofLine,
	)

	filebeat.Stop()

	// So far so good. Now check the output

	globPattern := outputFilePattern + "-*.ndjson"
	files, err := filepath.Glob(filepath.Join(tempDir, globPattern))
	require.NoError(t, err, "could not glob output file pattern")
	require.Lenf(t, files, 2,
		"expected only 2 output files. Glob pattern '%s'", globPattern)

	slices.SortFunc(files, func(a, b string) int {
		if len(a) < len(b) {
			return -1
		}
		if len(a) > len(b) {
			return 1
		}
		if len(a) == len(b) {
			return 0
		}

		panic("unreachable")
	})

	got, err := os.ReadFile(files[0])
	require.NoError(t, err, "could not open output file")
	// 1st file: check that all lines have been published
	matchPublishedLines(t, got, want1stLines)

	got, err = os.ReadFile(files[1])
	require.NoError(t, err, "could not open output file")
	matchPublishedLines(t, got, want2ndLines)
}

// TestFilestreamGZIPLogRotation_2_rotations test filebeat experiencing 2 file
// rotations. It simulates 3 log files, 2 of them will be rotated:
//   - 1st: 'a' content
//   - 2nd: 'b' content
//   - 3rd: 'c' content
//
// Filebeat will "see the files" in 3 different moments, by "see" it means,
// filebeat will start with the files written to disk, read then until their end
// and then stop. While filebeat is stopped, the logs are rotated, then filebeat
// is started again.
// This test simulates the following moments:
//   - 1st: only one active log file, 1/2 of the content
//     active file: 'a' content, only 1/2 of the logs
//   - 2nd: 1st log rotation
//     active file: 'b' content
//     *.1.gz file: 'a' content. Full content
//   - 3rd: 2nd log rotation
//     active file: 'c' content
//     *.1.gz file: 'b' content
//     *.2.gz file: 'a' content
func TestFilestreamGZIPLogRotation_2_rotations(t *testing.T) {
	want1stRunLines := make([]string, 0, 50)
	want2ndRunLines := make([]string, 0, 150)
	want3rdRunLines := make([]string, 0, 100)

	var dataPlainA1stHalf []byte
	for i := range 50 {
		l := fmt.Sprintf("%d: 1st 1/2 aaaaaaaaaaaaaaaaaaaaaaaaa", i)
		want1stRunLines = append(want1stRunLines, l)
		dataPlainA1stHalf = append(dataPlainA1stHalf, []byte(l+"\n")...)
	}

	var dataPlainA2ndHalf []byte
	for i := range 50 {
		l := fmt.Sprintf("%d: 2nd 1/2 aaaaaaaaaaaaaaaaaaaaaaaaa", i)
		want2ndRunLines = append(want2ndRunLines, l)
		dataPlainA2ndHalf = append(dataPlainA2ndHalf, []byte(l+"\n")...)
	}

	var dataPlainB []byte
	for i := range 100 {
		l := fmt.Sprintf("%d: bbbbbbbbbbbbbbbbbbbbbbbb", i)
		want2ndRunLines = append(want2ndRunLines, l)
		dataPlainB = append(dataPlainB, []byte(l+"\n")...)
	}

	var dataPlainC []byte
	for i := range 100 {
		l := fmt.Sprintf("%d: cccccccccccccccccccccccccccc", i)
		want3rdRunLines = append(want3rdRunLines, l)
		dataPlainC = append(dataPlainC, []byte(l+"\n")...)
	}

	dataGZA := gziptest.Compress(t,
		append(dataPlainA1stHalf, dataPlainA2ndHalf...), gziptest.CorruptNone)
	dataGZB := gziptest.Compress(t, dataPlainB, gziptest.CorruptNone)

	filebeat := integration.NewBeat(
		t,
		"filebeat",
		"../../filebeat.test",
	)
	tempDir := filebeat.TempDir()

	logFileBaseName := "plain.log"
	logPathActive := filepath.Join(tempDir, logFileBaseName)
	logPath1stRotation := filepath.Join(tempDir, logFileBaseName+".1.gz")
	logPath2ndRotation := filepath.Join(tempDir, logFileBaseName+".2.gz")

	outputFilePattern := "output-file"
	cfg := fmt.Sprintf(`
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
logging.level: debug
`, logPathActive+"*", filebeat.TempDir(), outputFilePattern)
	filebeat.WriteConfigFile(cfg)

	// 1st: only one active log file, 1/2 of the content
	err := os.WriteFile(logPathActive, dataPlainA1stHalf, 0644)
	require.NoError(t, err, "could not write 'a' file to disk")

	filebeat.Start()
	eofLine := fmt.Sprintf("End of file reached: %s; Backoff now.", logPathActive)
	filebeat.WaitLogsContainsFromBeginning(
		eofLine,
		30*time.Second,
		"Filebeat did not reach EOF. Did not find log [%s]",
		eofLine,
	)
	filebeat.Stop()

	// 2nd: 1st log rotation
	// finish writing the 'a' file
	f, err := os.OpenFile(logPathActive, os.O_APPEND|os.O_WRONLY, 0644)
	require.NoError(t, err, "could not open 'a' file to append")
	_, err = f.Write(dataPlainA2ndHalf)
	require.NoError(t, err, "could not append to 'a' file")
	require.NoError(t, f.Close(), "could not close 'a' file after appending")

	// "copy" and gzip the 'a' file
	err = os.WriteFile(logPath1stRotation, dataGZA, 0644)
	require.NoError(t, err, "could not write gzipped 'a' file")

	// truncate active and write 'b' file
	err = os.WriteFile(logPathActive, dataPlainB, 0644)
	require.NoError(t, err, "could not write 'b' file")

	// at this point there is:
	// - an active file with 'b' content
	// - a '.1.gz' file with the full 'a' content

	filebeat.Start()

	waitForLatestOutput(t, outputFilePattern, tempDir, len(want2ndRunLines))
	// check the output
	files := getOutputFilesSorted(t, outputFilePattern, tempDir)
	require.Len(t, files, 2, "expected 2 output files")
	got, err := os.ReadFile(files[1])
	require.NoError(t, err, "could not open output file")
	matchPublishedLines(t, got, want2ndRunLines)

	filebeat.Stop()

	// 3rd: 2nd log rotation
	// move '.1.gz' to '.2.gz'
	err = os.Rename(logPath1stRotation, logPath2ndRotation)
	require.NoError(t, err, "could not move 'a' gzipped file")

	// "copy" and gzip the 'b' file
	err = os.WriteFile(logPath1stRotation, dataGZB, 0644)
	require.NoError(t, err, "could not write gzipped 'b' file")

	// truncate active and write 'c' file
	err = os.WriteFile(logPathActive, dataPlainC, 0644)
	require.NoError(t, err, "could not write 'c' file")

	// at this point there is:
	// - an active file with 'c' content
	// - a '.1.gz' file with 'b' content
	// - a '.2.gz' file with 'a' content

	filebeat.Start()
	waitForLatestOutput(t, outputFilePattern, tempDir, len(want3rdRunLines))
	filebeat.Stop()

	// So far so good. Now check all the output files
	files = getOutputFilesSorted(t, outputFilePattern, tempDir)

	got, err = os.ReadFile(files[0])
	require.NoError(t, err, "could not open output file")
	matchPublishedLines(t, got, want1stRunLines)

	got, err = os.ReadFile(files[1])
	require.NoError(t, err, "could not open output file")
	matchPublishedLines(t, got, want2ndRunLines)

	got, err = os.ReadFile(files[2])
	require.NoError(t, err, "could not open output file")
	matchPublishedLines(t, got, want3rdRunLines)
}

func TestFilestreamGZIPReadsCorruptedFileUntilEOF(t *testing.T) {
	// For future reference, this is the code used to generate
	// testdata/gzip/corrupted.gz
	//
	// lines := make([]string, 0, 200)
	// var content []byte
	// for i := range 100 {
	// 	l := fmt.Sprintf("%d: 1st file log line", i)
	// 	lines = append(lines, l)
	// 	content = append(content, []byte(l+"\n")...)
	// }
	// gzData := gziptest.Compress(t, content, gziptest.CorruptNone)
	// gzData[50] = ^gzData[50]

	filebeat := integration.NewBeat(
		t,
		"filebeat",
		"../../filebeat.test",
	)
	logPath := filepath.Join("testdata", "gzip", "corrupted.gz")
	logPath, err := filepath.Abs(logPath)
	require.NoError(t, err, "could not find absolute path for log file")
	outputFilePattern := "output-file"

	workDir := filebeat.TempDir()
	cfg := fmt.Sprintf(`
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
logging.level: debug
`, logPath, workDir, outputFilePattern)

	filebeat.WriteConfigFile(cfg)
	filebeat.Start()

	eofLog := fmt.Sprintf("EOF has been reached. Closing. Path='%s'", logPath)
	filebeat.WaitLogsContainsFromBeginning(
		eofLog,
		30*time.Second,
		"Filebeat did not reach EOF. Did not find log [%s]",
		eofLog,
	)
	filebeat.Stop()

	// assert EOF is set on registry
	registryLogFile := filepath.Join(workDir,
		"data", "registry", "filebeat", "log.json")
	entries, _ := readFilestreamRegistryLog(t, registryLogFile)
	var lastEntry *registryEntry
	for i := range entries {
		entry := &entries[i]
		if entry.Filename == logPath {
			lastEntry = entry
		}
	}
	require.NotNil(t, lastEntry,
		"state for log file not found in registry for %s", logPath)
	assert.True(t, lastEntry.EOF, "EOF is not true")

	pattern := outputFilePattern + "-*.ndjson"
	files, err := filepath.Glob(filepath.Join(workDir, pattern))
	require.NoError(t, err, "could not glob output file pattern")
	require.Len(t, files, 1, "expected only 1 output file, file glob pattern: '%s'",
		pattern)

	assertLogFieldsEqual(t,
		filepath.Join("testdata", "gzip", "output-file-for-corrupted.gz.ndjson"),
		files[0],
	)
}

func getOutputFilesSorted(t *testing.T, outputFilePattern string, tempDir string) []string {
	globPattern := outputFilePattern + "-*.ndjson"
	files, err := filepath.Glob(filepath.Join(tempDir, globPattern))
	require.NoError(t, err, "could not glob output file pattern")

	slices.SortFunc(files, func(a, b string) int {
		if len(a) < len(b) {
			return -1
		}
		if len(a) > len(b) {
			return 1
		}
		if len(a) == len(b) {
			return strings.Compare(a, b)
		}

		panic("unreachable")
	})
	return files
}

func matchPublishedLinesFromFile(t *testing.T, outputFilePattern string, lines []string) {
	pattern := outputFilePattern + "-*.ndjson"
	files, err := filepath.Glob(pattern)
	require.NoError(t, err, "could not glob output file pattern")
	require.Len(t, files, 1, "expected only 1 output file")

	got, err := os.ReadFile(files[0])
	require.NoError(t, err, "could not open output file")

	// check that all lines have been published
	matchPublishedLines(t, got, lines)
}

func matchPublishedLines(t *testing.T, got []byte, want []string) {
	gotLinesJSON := strings.Split(strings.TrimSpace(string(got)), "\n")
	assert.Equal(t, len(want), len(gotLinesJSON), "unexpected number of events")

	gotLines := make([]string, len(gotLinesJSON))

	logLine := struct {
		Message string `json:"message"`
	}{}
	for i, line := range gotLinesJSON {
		err := json.Unmarshal([]byte(line), &logLine)
		require.NoError(t, err, "could not Unmarshal log line")
		gotLines[i] = logLine.Message
	}

	slices.Sort(gotLines)
	slices.Sort(want)

	assert.Equal(t, want, gotLines, "not all lines match")
}

func assertLogFieldsEqual(t *testing.T, wantPath, gotPath string) {
	t.Helper()

	type event struct {
		Message string `json:"message"`
		Log     struct {
			Offset int64 `json:"offset"`
		} `json:"log"`
	}

	open := func(path string) *bufio.Scanner {
		f, err := os.Open(path)
		require.NoError(t, err, "opening file %s", path)
		t.Cleanup(func() { _ = f.Close() })
		return bufio.NewScanner(f)
	}

	wantScanner := open(wantPath)
	gotScanner := open(gotPath)

	line := 1
	for {
		wantOK := wantScanner.Scan()
		gotOK := gotScanner.Scan()

		if !wantOK || !gotOK {
			assert.Equal(t, wantOK, gotOK,
				"different number of lines: want EOF=%v, got EOF=%v at line %d",
				!wantOK, !gotOK, line,
			)
			return
		}

		var wantEv, gotEv event
		if err := json.Unmarshal(wantScanner.Bytes(), &wantEv); err != nil {
			t.Fatalf("failed to unmarshal want JSON at line %d: %v", line, err)
		}
		if err := json.Unmarshal(gotScanner.Bytes(), &gotEv); err != nil {
			t.Fatalf("failed to unmarshal got JSON at line %d: %v", line, err)
		}

		if wantEv.Message != gotEv.Message ||
			wantEv.Log.Offset != gotEv.Log.Offset {
			t.Errorf("line %d mismatch:\n\tmessage: want '%q got %q\n\toffset:  want %d got %d",
				line, wantEv.Message, gotEv.Message, wantEv.Log.Offset, gotEv.Log.Offset)
		}
		line++
	}
}

func waitForLatestOutput(t *testing.T, outputFilePattern string, tempDir string, want int) {
	t.Helper()

	// wait for all lines in the output
	msg := &strings.Builder{}
	var files []string
	condition := func() bool {
		// writeMsg was intended to avoid the message being reset, the function
		// being executed, and before the function completes and writes the new
		// message, the timeout elapses and the error message is written with an
		// empty msg. But it fails too often, so the whole `waitForLatestOutput`
		// is the real fix. It duplicates the error, but bette that than
		// missing information :/
		writeMsg := func(format string, a ...any) {
			msg.Reset()
			fmt.Fprintf(msg, format, a...)
		}
		msg.Reset()

		files = getOutputFilesSorted(t, outputFilePattern, tempDir)
		if len(files) == 0 {
			writeMsg("no output file")
			return false
		}

		got, _ := os.ReadFile(files[len(files)-1])
		lines := strings.Split(strings.TrimSuffix(string(got), "\n"), "\n")

		if len(lines) != want {
			writeMsg("want %d lines, got %d",
				want, len(lines))
			return false
		}

		return true
	}

	if !assert.Eventuallyf(t, condition, 45*time.Second, 500*time.Millisecond,
		"output file isn't what we expect: %s", msg) {
		// call the condition one last time to ensure the msg isn't reset and not
		// yet written when it's called as part of the eventually message
		condition()
		require.Failf(t, "condition never satisfied",
			"output file isn't what we expect: %s", msg)
	}
}
