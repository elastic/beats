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

package filestream

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	loginp "github.com/elastic/beats/v7/filebeat/input/filestream/internal/input-logfile"
	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/statestore"
	"github.com/elastic/beats/v7/libbeat/statestore/storetest"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
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
// `eventLimit` is an amount of produced events after which the filestream will shutdown
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
	context := v2.Context{
		Logger:      logger,
		ID:          testID,
		Cancelation: ctx,
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
		err := input.Run(context, connector)
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

func createTestStore(t testing.TB) loginp.StateStore {
	return &testStore{registry: statestore.NewRegistry(storetest.NewMemoryStoreBackend())}
}

type testStore struct {
	registry *statestore.Registry
}

func (s *testStore) Close() {
	s.registry.Close()
}

func (s *testStore) Access() (*statestore.Store, error) {
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
