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

// This file was contributed to by generative AI

package filestream

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/filebeat/testing/gziptest"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/reader/readfile/encoding"
	"github.com/elastic/beats/v7/libbeat/statestore"
	"github.com/elastic/beats/v7/libbeat/statestore/storetest"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

func BenchmarkFilestream(b *testing.B) {
	logp.TestingSetup(logp.ToDiscardOutput())

	b.Run("single file", func(b *testing.B) {
		lineCount := 10000
		filename := generateFile(b, b.TempDir(), lineCount)
		b.ResetTimer()

		b.Run("inode throughput", func(b *testing.B) {
			cfg := `
type: filestream
prospector.scanner.check_interval: 1s
prospector.scanner.fingerprint.enabled: false
paths:
    - ` + filename + `
`
			for i := 0; i < b.N; i++ {
				runFilestreamBenchmark(b, fmt.Sprintf("one-file-inode-benchmark-%d", i), cfg, lineCount)
			}
		})

		b.Run("fingerprint throughput", func(b *testing.B) {
			cfg := `
type: filestream
prospector.scanner:
  fingerprint.enabled: true
  check_interval: 1s
file_identity.fingerprint: ~
paths:
  - ` + filename + `
`
			for i := 0; i < b.N; i++ {
				runFilestreamBenchmark(b, fmt.Sprintf("one-file-fp-benchmark-%d", i), cfg, lineCount)
			}
		})
	})

	b.Run("many files", func(b *testing.B) {
		lineCount := 1000
		fileCount := 100
		dir := b.TempDir()

		for i := 0; i < fileCount; i++ {
			_ = generateFile(b, dir, lineCount)
		}

		ingestPath := filepath.Join(dir, "*")
		expEvents := lineCount * fileCount
		b.ResetTimer()

		b.Run("inode throughput", func(b *testing.B) {
			cfg := `
type: filestream
prospector.scanner.check_interval: 1s
prospector.scanner.fingerprint.enabled: false
paths:
    - ` + ingestPath + `
`
			for i := 0; i < b.N; i++ {
				runFilestreamBenchmark(b, fmt.Sprintf("many-files-inode-benchmark-%d", i), cfg, expEvents)
			}
		})

		b.Run("fingerprint throughput", func(b *testing.B) {
			cfg := `
type: filestream
prospector.scanner:
  fingerprint.enabled: true
  check_interval: 1s
file_identity.fingerprint: ~
paths:
  - ` + ingestPath + `
`
			for i := 0; i < b.N; i++ {
				runFilestreamBenchmark(b, fmt.Sprintf("many-files-fp-benchmark-%d", i), cfg, expEvents)
			}
		})
	})
}

func TestTakeOverTags(t *testing.T) {
	testCases := []struct {
		name     string
		takeOver bool
		testFunc func(t *testing.T, event beat.Event)
	}{
		{
			name:     "test-take_over-true",
			takeOver: true,
			testFunc: func(t *testing.T, event beat.Event) {
				tags, err := event.GetValue("tags")
				require.NoError(t, err)
				require.Contains(t, tags, "take_over")
			},
		},
		{
			name:     "test-take_over-false",
			takeOver: false,
			testFunc: func(t *testing.T, event beat.Event) {
				_, err := event.GetValue("tags")
				require.ErrorIs(t, err, mapstr.ErrKeyNotFound)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			filename := generateFile(t, t.TempDir(), 5)
			cfg := fmt.Sprintf(`
type: filestream
id: foo
prospector.scanner.check_interval: 1s
prospector.scanner.fingerprint.enabled: false
take_over.enabled: %t
paths:
    - %s`, testCase.takeOver, filename)
			runner := createFilestreamTestRunner(context.Background(), t, testCase.name, cfg, 5, true)
			events := runner(t)
			for _, event := range events {
				testCase.testFunc(t, event)
			}
		})
	}
}

func TestNewFile(t *testing.T) {
	tempDir := t.TempDir()

	plainFileContent := "this is a plain file"
	plainFilePath := filepath.Join(tempDir, "plain.txt")
	err := os.WriteFile(plainFilePath, []byte(plainFileContent), 0644)
	require.NoError(t, err, "could not write plain file")

	gzipFileContent := "this is a gzipped file"
	var gzipBuf bytes.Buffer
	gzipWriter := gzip.NewWriter(&gzipBuf)
	_, err = gzipWriter.Write([]byte(gzipFileContent))
	require.NoError(t, err)
	err = gzipWriter.Close()
	require.NoError(t, err)
	gzippedFilePath := filepath.Join(tempDir, "test.gz")
	err = os.WriteFile(gzippedFilePath, gzipBuf.Bytes(), 0644)
	require.NoError(t, err)

	testCases := map[string]struct {
		compression   string
		filePath      string
		expectedType  interface{}
		expectError   bool
		errorContains string
		setup         func(t *testing.T, filePath string) *os.File
	}{
		"compression_none_returns_plain_file": {
			compression:  CompressionNone,
			filePath:     plainFilePath,
			expectedType: &plainFile{},
		},
		"compression_gzip_with_gzip_file_returns_gzip_reader": {
			compression:  CompressionGZIP,
			filePath:     gzippedFilePath,
			expectedType: &gzipSeekerReader{},
		},
		"compression_gzip_with_plain_file_returns_error": {
			compression:   CompressionGZIP,
			filePath:      plainFilePath,
			expectError:   true,
			errorContains: "failed to create gzip reader",
		},
		"compression_auto_with_plain_file_returns_plain_file": {
			compression:  CompressionAuto,
			filePath:     plainFilePath,
			expectedType: &plainFile{},
		},
		"compression_auto_with_gzip_file_returns_gzip_reader": {
			compression:  CompressionAuto,
			filePath:     gzippedFilePath,
			expectedType: &gzipSeekerReader{},
		},
		"compression_auto_with_unreadable_file_returns_error": {
			compression: CompressionAuto,
			filePath:    plainFilePath, // content doesn't matter
			setup: func(t *testing.T, filePath string) *os.File {
				// Return a file that is already closed to trigger a read error
				// in IsGZIP
				f, err := os.Open(filePath)
				require.NoError(t, err)
				f.Close()
				return f
			},
			expectError:   true,
			errorContains: "gzip detection error",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			inp := &filestream{
				compression:  tc.compression,
				readerConfig: defaultReaderConfig(),
			}

			var rawFile *os.File
			if tc.setup != nil {
				rawFile = tc.setup(t, tc.filePath)
			} else {
				var err error
				rawFile, err = os.Open(tc.filePath)
				require.NoError(t, err)
			}
			defer rawFile.Close()

			file, err := inp.newFile(rawFile)

			if tc.expectError {
				require.Error(t, err)
				if tc.errorContains != "" {
					assert.Contains(t, err.Error(), tc.errorContains)
				}
				assert.Nil(t, file)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, file)
				assert.IsType(t, tc.expectedType, file)
			}
		})
	}
}

func TestOpenFile_GZIPNeverTruncated(t *testing.T) {
	log := logptest.NewTestingLogger(t, "", zap.AddStacktrace(zapcore.ErrorLevel+1))

	tempDir := t.TempDir()
	plainData := []byte("some plain data\n")

	plainPath := filepath.Join(tempDir, "plain.txt")
	err := os.WriteFile(plainPath, plainData, 0644)
	require.NoError(t, err, "could not save plain file")

	data := gziptest.Compress(t, plainData, gziptest.CorruptNone)
	gzPath := filepath.Join(tempDir, "test.gz")
	err = os.WriteFile(gzPath, data, 0644)
	require.NoError(t, err, "could not save gzip file")

	tcs := []struct {
		name        string
		compression string
		path        string
		want        bool
		errMsg      string
	}{
		{
			name:        "plain file is truncated",
			compression: CompressionNone,
			path:        plainPath,
			want:        true,
			errMsg:      "plain file should be considered truncated",
		},
		{
			name:        "GZIP file is never truncated",
			compression: CompressionAuto,
			path:        gzPath,
			want:        false,
			errMsg:      "GZIP file skips truncated validation",
		},
	}

	for _, tc := range tcs {
		inp := filestream{
			compression:     tc.compression,
			encodingFactory: encoding.Plain,
			readerConfig:    readerConfig{BufferSize: 32},
		}

		f, _, truncated, err := inp.openFile(
			log, tc.path, int64(len(plainData)*2))
		require.NoError(t, err, "unexpected error")
		f.Close()

		assert.Equal(t, tc.want, truncated, tc.errMsg)
	}
}

// runFilestreamBenchmark runs the entire filestream input with the in-memory registry and the test pipeline.
// `testID` must be unique for each test run
// `cfg` must be a valid YAML string containing valid filestream configuration
// `expEventCount` is an expected amount of produced events
func runFilestreamBenchmark(b *testing.B, testID string, cfg string, expEventCount int) {
	b.Helper()
	// we don't include initialization in the benchmark time
	b.StopTimer()
	runner := createFilestreamTestRunner(context.Background(), b, testID, cfg, int64(expEventCount), false)
	// this is where the benchmark actually starts
	b.StartTimer()
	_ = runner(b)
}

// createFilestreamTestRunner can be used for both benchmarks and regular tests to run a filestream input
// with the given configuration and event limit.
// `testID` must be unique for each test run
// `cfg` must be a valid YAML string containing valid filestream configuration
// `eventLimit` is an amount of produced events after which the filestream will shut down
// `collectEvents` if `true` the runner will return a list of all events produced by the filestream input.
// Events should not be collected in benchmarks due to high extra costs of using the channel.
//
// returns a runner function that returns produced events.
func createFilestreamTestRunner(ctx context.Context, b testing.TB, testID string, cfg string, eventLimit int64, collectEvents bool) func(t testing.TB) []beat.Event {
	logger := logp.L()
	c, err := conf.NewConfigWithYAML([]byte(cfg), cfg)
	require.NoError(b, err)

	p := Plugin(logger, createTestStore(b))
	input, err := p.Manager.Create(c)
	require.NoError(b, err)

	ctx, cancel := context.WithCancel(ctx)
	v2ctx := v2.Context{
		ID:              testID,
		IDWithoutName:   testID,
		Name:            "filestream-test",
		Agent:           beat.Info{},
		Cancelation:     ctx,
		MetricsRegistry: monitoring.NewRegistry(),
		Logger:          logger,
	}

	connector, events := newTestPipeline(eventLimit, collectEvents)
	var out []beat.Event
	if collectEvents {
		out = make([]beat.Event, 0, eventLimit)
	}
	go func() {
		// even if `collectEvents` is false we need to range the channel
		// and wait until it's closed indicating that the input finished its job
		for event := range events {
			out = append(out, event)
		}
		cancel()
	}()

	return func(t testing.TB) []beat.Event {
		err := input.Run(v2ctx, connector)
		require.NoError(b, err)

		return out
	}
}

func generateFile(t testing.TB, dir string, lineCount int) string {
	t.Helper()
	file, err := os.CreateTemp(dir, "*")
	require.NoError(t, err)
	filename := file.Name()
	for i := 0; i < lineCount; i++ {
		fmt.Fprintf(file, "rather mediocre log line message in %s - %d\n", filename, i)
	}
	err = file.Close()
	require.NoError(t, err)
	return filename
}

func createTestStore(t testing.TB) statestore.States {
	return &testStore{registry: statestore.NewRegistry(storetest.NewMemoryStoreBackend())}
}

var _ statestore.States = (*testStore)(nil)

type testStore struct {
	registry *statestore.Registry
}

func (s *testStore) Close() {
	s.registry.Close()
}

func (s *testStore) StoreFor(string) (*statestore.Store, error) {
	return s.registry.Get("filestream-benchmark")
}

func (s *testStore) CleanupInterval() time.Duration {
	return time.Second
}

func newTestPipeline(eventLimit int64, collectEvents bool) (pc beat.PipelineConnector, out <-chan beat.Event) {
	ch := make(chan beat.Event, eventLimit)
	return &testPipeline{limit: eventLimit, out: ch, collect: collectEvents}, ch
}

type testPipeline struct {
	limit   int64
	out     chan beat.Event
	collect bool
}

func (p *testPipeline) ConnectWith(beat.ClientConfig) (beat.Client, error) {
	return p.Connect()
}
func (p *testPipeline) Connect() (beat.Client, error) {
	return &testClient{p}, nil
}

type testClient struct {
	testPipeline *testPipeline
}

func (c *testClient) Publish(event beat.Event) {
	newLimit := atomic.AddInt64(&c.testPipeline.limit, -1)
	if newLimit < 0 {
		return
	}
	if c.testPipeline.collect {
		c.testPipeline.out <- event
	}
	if newLimit == 0 {
		close(c.testPipeline.out)
	}
}

func (c *testClient) PublishAll(events []beat.Event) {
	for _, e := range events {
		c.Publish(e)
	}
}
func (c *testClient) Close() error {
	return nil
}
