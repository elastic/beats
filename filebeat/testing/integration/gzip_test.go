package integration

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/klauspost/compress/gzip"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/testing/integration"
)

type gzipCorruption int

const (
	corruptCRC gzipCorruption = 1 << iota
	corruptSize
)

func TestGZIP(t *testing.T) {
	messageSuffix :=
		"sample test message long enough for fingerprint to work. 'Nothing is " +
			"sad until it’s over. Then everything is', 'That's why I keep moving" +
			" on, to see the next thing, and the next, and the next. And " +
			"sometimes... It looks even better through your eyes.'"
	content := []byte("some log line\nanother line\n" + messageSuffix + "\n")
	content = append(
		content, []byte(strings.Repeat("a log line\n", 100))...)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	EnsureCompiled(ctx, t)

	reportOptions := integration.ReportOptions{
		PrintLinesOnFail:  100,
		PrintConfigOnFail: true,
	}

	t.Run("CRC integrity", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		td := t.TempDir()
		logPath := filepath.Join(td, "input.log.gz")

		err := os.WriteFile(
			logPath, craftCorruptedGzip(t, content, corruptCRC), 0644)
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

		fbt := NewTest(t, TestOptions{
			Config: config,
		})

		fbt.
			WithReportOptions(reportOptions).
			ExpectStart().
			ExpectOutput("Read line error: gzip: invalid checksum").
			Start(ctx).
			Wait()
	})

	t.Run("Size integrity", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		td := t.TempDir()
		logPath := filepath.Join(td, "input.log.gz")

		err := os.WriteFile(
			logPath, craftCorruptedGzip(t, content, corruptSize), 0644)
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
			// The gzip lib returns the same error for checksum and size
			// validation :/
			ExpectOutput("Read line error: gzip: invalid checksum").
			Start(ctx).
			Wait()
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

// craftCorruptedGzip takes input data, compresses it using gzip,
// and then intentionally corrupts parts of the footer (CRC32 and/or ISIZE)
// to simulate checksum/length errors upon decompression.
// It returns the corrupted, compressed GZIP data.
// Check the RFC1 952 for details https://www.rfc-editor.org/rfc/rfc1952.html.
func craftCorruptedGzip(t *testing.T, data []byte, corruption gzipCorruption) []byte {
	var gzBuff bytes.Buffer
	gw := gzip.NewWriter(&gzBuff)

	wrote, err := gw.Write(data)
	require.NoError(t, err, "failed to write data to gzip writer")
	// sanity check
	require.Equal(t, len(data), wrote, "written data is not equal to input data")
	require.NoError(t, gw.Close(), "failed to close gzip writer")

	compressedBytes := gzBuff.Bytes()

	// get the footer start index
	footerStartIndex := len(compressedBytes) - 8

	if corruption&corruptCRC != 0 {
		// CRC32 - first 4 bytes of footer
		originalCRC32 := binary.LittleEndian.Uint32(
			compressedBytes[footerStartIndex : footerStartIndex+4])
		// corrupted the CRC32, anything will do.
		corruptedCRC32 := originalCRC32 + 1
		binary.LittleEndian.PutUint32(
			compressedBytes[footerStartIndex:footerStartIndex+4], corruptedCRC32)
	}

	if corruption&corruptSize != 0 {
		// ISIZE - last 4 bytes of footer
		originalISIZE := binary.LittleEndian.Uint32(
			compressedBytes[footerStartIndex+4 : footerStartIndex+8])
		// corrupted the ISIZE, anything will do
		corruptedISIZE := originalISIZE + 1
		binary.LittleEndian.PutUint32(
			compressedBytes[footerStartIndex+4:footerStartIndex+8], corruptedISIZE)
	}

	return compressedBytes
}
