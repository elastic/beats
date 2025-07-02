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

func TestGZIP(t *testing.T) {
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

	t.Run("gzip files still being read", func(t *testing.T) {
		gzData := gziptest.Compress(t, content, gziptest.CorruptNone)
		tcs := []struct {
			name            string
			data            []byte
			restData        []byte
			addExpectations func(bt integration.BeatTest, logPath string) integration.BeatTest
		}{
			{
				// It cannot open the file, once the header is fully written, it
				// finally opens the file.
				name:     "incomplete header",
				data:     gzData[:5],
				restData: gzData[5:],
				addExpectations: func(bt integration.BeatTest, logPath string) integration.BeatTest {
					bt.ExpectOutput(fmt.Sprintf("cannot create a file descriptor for an ingest target \"%s\": failed to create gzip seeker: could not create gzip reader: unexpected EOF", logPath)).
						ExpectOutput(fmt.Sprintf("A new file %s has been found", logPath))

					return bt
				},
			},
			{
				// Reads all possible data, then once the file is updated, it
				// resumes from where it left off.
				name:     "full header and incomplete data",
				data:     gzData[:len(gzData)-20],
				restData: gzData[len(gzData)-20:],
				addExpectations: func(bt integration.BeatTest, logPath string) integration.BeatTest {
					bt.ExpectOutput(fmt.Sprintf("Unexpected state reading from %s; error: unexpected EOF", logPath)).
						ExpectOutput("Error stopping filestream reader could not close gzip reader: unexpected EOF").
						ExpectOutput(fmt.Sprintf("File %s has been updated", logPath))

					return bt
				},
			},
			{
				// It reads all lines, when the footer is fully read, it opens
				// the file again, sees it reached EOF and closes the file.
				name:     "incomplete footer",
				data:     gzData[:len(gzData)-8],
				restData: gzData[len(gzData)-8:],
				addExpectations: func(bt integration.BeatTest, logPath string) integration.BeatTest {
					bt.ExpectOutput(fmt.Sprintf("Unexpected state reading from %s; error: unexpected EOF", logPath)).
						ExpectOutput(fmt.Sprintf("File %s has been updated", logPath))

					return bt
				},
			},
		}

		for _, tc := range tcs {
			t.Run(tc.name, func(t *testing.T) {
				timeout := 40 * time.Second
				ctx, cancel := context.WithTimeout(context.Background(), timeout)
				defer cancel()

				tempDir := t.TempDir()
				outputFilename := "output-file"
				logPath := filepath.Join(tempDir, "input.log.gz")

				err := os.WriteFile(logPath, tc.data, 0644)
				require.NoError(t, err)

				// TODO(AndersonQ): I bet it'll become flaky in CI, so use the
				//   other frameworks to test this as it allows to wait for a
				//   log, the perform actions and wait for another log
				go func() {
					f, err := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY, 0644)
					if !assert.NoError(t, err, "failed to open log file to append leftover data") {
						return
					}
					time.Sleep(20 * time.Second)
					n, err := f.Write(tc.restData)
					assert.Equal(t, len(tc.restData), n, "unexpected amount of the leftover data written to file")
					assert.NoError(t, err, "failed to write leftover data to log file")
					assert.NoError(t, f.Close(), "failed to close log file after writing leftover data")
				}()

				config := fmt.Sprintf(`
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
`, logPath, tempDir, outputFilename)

				ft := NewTest(t, TestOptions{
					Config: config,
				})

				bt := ft.
					ExpectEOF(logPath).
					WithReportOptions(reportOptions)
				bt.ExpectStart()
				tc.addExpectations(bt, logPath)
				// Wait for events to be published
				bt.ExpectOutput("ackloop: return ack to broker loop:100").
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
    gzip_experimental: true
    rotation.external.strategy.copytruncate.suffix_regex: \.\d+(\.gz)?$
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

				// TODO(AndersonQ): save the output and expected lines
			})
		}
	})

	t.Run("TechPreviewWarning", func(t *testing.T) {
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
    rotation.external.strategy.copytruncate.suffix_regex: \.\d+(\.gz)?$
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
				"EXPERIMENTAL: filestream: experimental gzip support enabled").
			Start(ctx).Wait()
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
