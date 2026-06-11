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
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	loginp "github.com/elastic/beats/v7/filebeat/input/filestream/internal/input-logfile"
	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/statestore"
	"github.com/elastic/beats/v7/libbeat/statestore/storetest"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

func BenchmarkFilestream(b *testing.B) {
	// Info level keeps per-line Debugf calls out of the hot path.
	logger := logptest.NewTestingLogger(b, "", zap.IncreaseLevel(zap.InfoLevel))

	cases := []struct {
		name        string
		lineCount   int
		fileCount   int
		fingerprint bool
	}{
		{"1_file/inode", 10_000, 1, false},
		{"1_file/fingerprint", 10_000, 1, true},
		{"100_files/inode", 1000, 100, false},
		{"100_files/fingerprint", 1000, 100, true},
		{"1000_files/fingerprint", 20, 1000, true},
		{"10000_files/fingerprint", 20, 10_000, true},
	}

	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			dir := b.TempDir()
			var ingestPath string
			for i := 0; i < tc.fileCount; i++ {
				ingestPath = generateFile(b, dir, tc.lineCount)
			}

			if tc.fileCount > 1 {
				ingestPath = filepath.Join(dir, "*")
			}

			expEvents := tc.lineCount * tc.fileCount
			cfg := filestreamBenchCfg(ingestPath, tc.fingerprint)
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				runFilestreamBenchmark(b, logger, fmt.Sprintf("%s-%d", tc.name, i), cfg, expEvents)
			}
		})
	}

	b.Run("line_filter", func(b *testing.B) {
		lineCount := 10_000
		filename := generateFile(b, b.TempDir(), lineCount)
		b.ResetTimer()

		filterCases := []struct {
			name         string
			includeLines string
			excludeLines string
			expEvents    int
		}{
			{"none", "", "", lineCount},
			{"include", "include_lines: ['^rather']", "", lineCount},
			{"exclude", "", "exclude_lines: ['^NOMATCH']", lineCount},
			{"include_and_exclude", "include_lines: ['^rather']", "exclude_lines: ['^NOMATCH']", lineCount},
			{"drop_all", "include_lines: [' - 9999$']", "", 1},
		}
		for _, fc := range filterCases {
			b.Run(fc.name, func(b *testing.B) {
				cfg := fmt.Sprintf(`
type: filestream
prospector.scanner.check_interval: 100ms
prospector.scanner.fingerprint.enabled: false
close.reader.on_eof: true
file_identity.native: ~
%s
%s
paths:
    - %s
`, fc.includeLines, fc.excludeLines, filename)
				for i := 0; i < b.N; i++ {
					runFilestreamBenchmark(b, logger, fmt.Sprintf("filter-%s-%d", fc.name, i), cfg, fc.expEvents)
				}
			})
		}
	})
}

func filestreamBenchCfg(path string, fingerprint bool) string {
	identity := `
prospector.scanner.fingerprint.enabled: false
file_identity.native: ~`
	if fingerprint {
		identity = `
prospector.scanner.fingerprint.enabled: true
file_identity.fingerprint: ~`
	}
	return fmt.Sprintf(`
type: filestream
prospector.scanner.check_interval: 100ms
close.reader.on_eof: true%s
paths:
  - %s
`, identity, path)
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
	logger := logptest.NewTestingLogger(t, "")
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			filename := generateFile(t, t.TempDir(), 5)
			cfg := fmt.Sprintf(`
type: filestream
prospector.scanner.check_interval: 1s
prospector.scanner.fingerprint.enabled: false
take_over: %t
paths:
    - %s`, testCase.takeOver, filename)
			runner := createFilestreamTestRunner(t, logger, testCase.name, cfg, 5, true)
			events := runner(t)
			for _, event := range events {
				testCase.testFunc(t, event)
			}
		})
	}
}

// runFilestreamBenchmark runs the entire filestream input with the in-memory registry and the test pipeline.
// `testID` must be unique for each test run
// `cfg` must be a valid YAML string containing valid filestream configuration
// `expEventCount` is an expected amount of produced events
func runFilestreamBenchmark(b *testing.B, logger *logp.Logger, testID string, cfg string, expEventCount int) {
	b.Helper()
	// we don't include initialization in the benchmark time
	b.StopTimer()
	runner := createFilestreamTestRunner(b, logger, testID, cfg, int64(expEventCount), false)
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
func createFilestreamTestRunner(b testing.TB, logger *logp.Logger, testID string, cfg string, eventLimit int64, collectEvents bool) func(t testing.TB) []beat.Event {
	c, err := conf.NewConfigWithYAML([]byte(cfg), cfg)
	require.NoError(b, err)

	p := Plugin(logger, createTestStore(b))
	input, err := p.Manager.Create(c)
	require.NoError(b, err)

	ctx, cancel := context.WithCancel(b.Context())
	v2ctx := v2.Context{
		ID:              testID,
		IDWithoutName:   testID,
		Name:            "filestream-test",
		Agent:           beat.Info{},
		Cancelation:     ctx,
		MetricsRegistry: monitoring.NewRegistry(),
		Logger:          logger,
	}

	var out []beat.Event
	if collectEvents {
		out = make([]beat.Event, 0, eventLimit)
	}
	connector, events := newTestPipeline(eventLimit, collectEvents)
	go func() {
		defer cancel()
		for event := range events {
			out = append(out, event)
		}
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
	var chBuf int64
	if collectEvents {
		chBuf = eventLimit
	}
	ch := make(chan beat.Event, chBuf)
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

// TestFilestream_handleReadError_ErrClosed verifies the contract
// handleReadError must uphold for ErrClosed: the readUntilEOF
// drain must happen *only* when the input is being cancelled. A plain
// ErrClosed from any other source (close.reader.after_interval,
// close.on_state_change.removed, close.on_state_change.renamed, explicit
// Close) must close the reader.
func TestFilestream_handleReadError_ErrClosed(t *testing.T) {
	newCtx := func(t *testing.T, cancelled bool) v2.Context {
		t.Helper()
		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(cancel)
		if cancelled {
			cancel()
		}
		return v2.Context{
			Cancelation: ctx,
			Logger:      logptest.NewTestingLogger(t, ""),
		}
	}

	metrics := loginp.NewMetrics(monitoring.NewRegistry(), logp.NewNopLogger())

	t.Run("read_until_eof=false: always close", func(t *testing.T) {
		inp := &filestream{
			readUntilEOF: loginp.ReadUntilEOFConfig{Enabled: false},
		}
		for _, cancelled := range []bool{false, true} {
			ctx := newCtx(t, cancelled)
			gotErr, shouldContinue := inp.handleReadError(
				ctx, ErrClosed, ctx.Logger, "/path", metrics, false)
			assert.NoError(t, gotErr,
				"ErrClosed with read_until_eof=false must not propagate")
			assert.False(t, shouldContinue,
				"ErrClosed with read_until_eof=false must end the loop (cancelled=%v)",
				cancelled)
		}
	})

	t.Run("read_until_eof=true + input not cancelled: exit immediately", func(t *testing.T) {
		inp := &filestream{
			readUntilEOF: loginp.ReadUntilEOFConfig{Enabled: true},
		}
		ctx := newCtx(t, false)
		gotErr, shouldContinue := inp.handleReadError(
			ctx, ErrClosed, ctx.Logger, "/path", metrics, false)
		assert.NoError(t, gotErr,
			"ErrClosed must not propagate when input isn't closed")
		assert.False(t, shouldContinue,
			"ErrClosed with input not cancelled must close the reader")
	})

	t.Run("read_until_eof=true + input cancelled: triggers readUntilEOF", func(t *testing.T) {
		inp := &filestream{
			readUntilEOF: loginp.ReadUntilEOFConfig{Enabled: true},
		}
		ctx := newCtx(t, true)
		gotErr, shouldContinue := inp.handleReadError(
			ctx, ErrClosed, ctx.Logger, "/path", metrics, false)
		assert.NoError(t, gotErr,
			"handleReadError must return returning nil")
		assert.True(t, shouldContinue,
			"handleReadError must return true so readFromSource falls through to "+
				"the readUntilEOF block")
	})
}

// TestFilestream_handleReadError_OtherErrors ensures EOF / ErrInactive /
// ErrFileTruncate / unknown errors behave the same regardless of
// whether ctx is cancelled or read_until_eof is enabled.
func TestFilestream_handleReadError_OtherErrors(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	logger := logptest.NewTestingLogger(t, "")
	metrics := loginp.NewMetrics(monitoring.NewRegistry(), logp.NewNopLogger())

	for _, readUntilEOF := range []bool{false, true} {
		name := fmt.Sprintf("read_until_eof=%v", readUntilEOF)
		t.Run(name, func(t *testing.T) {
			inp := &filestream{
				readUntilEOF: loginp.ReadUntilEOFConfig{Enabled: readUntilEOF},
			}
			inpCtx := v2.Context{Cancelation: ctx, Logger: logger}

			t.Run("EOF", func(t *testing.T) {
				gotErr, shouldContinue := inp.handleReadError(
					inpCtx, io.EOF, logger, "/p", metrics, false)
				if readUntilEOF {
					assert.ErrorIs(t, gotErr, io.EOF,
						"read_until_eof=true: EOF must propagate so readFromSource "+
							"ends without entering readUntilEOF")
				} else {
					assert.NoError(t, gotErr)
				}
				assert.False(t, shouldContinue, "want shouldContinue == false")
			})

			t.Run("ErrInactive", func(t *testing.T) {
				gotErr, shouldContinue := inp.handleReadError(
					inpCtx, ErrInactive, logger, "/p", metrics, false)
				assert.ErrorIs(t, gotErr, ErrInactive)
				assert.False(t, shouldContinue, "want shouldContinue == false")
			})

			t.Run("ErrFileTruncate", func(t *testing.T) {
				gotErr, shouldContinue := inp.handleReadError(
					inpCtx, ErrFileTruncate, logger, "/p", metrics, false)
				assert.NoError(t, gotErr, "ErrFileTruncate shouldn't propagate")
				assert.False(t, shouldContinue, "want shouldContinue == false")
			})
		})
	}
}
