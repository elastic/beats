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

				corruptedGZIP := gziptest.CraftCorruptedGzip(t, content, tc.corruption)
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

				data, err := os.ReadFile(files[0])
				require.NoError(t, err, "could not open output file")

				gotLines := strings.Split(strings.TrimSpace(string(data)), "\n")
				assert.Equal(t, len(lines), len(gotLines), "unexpected number of events")

				for i, l := range lines {
					assert.Contains(t,
						gotLines[i],
						fmt.Sprintf(`"message":"%s"`, l),
						"expected output to match input")
				}
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
