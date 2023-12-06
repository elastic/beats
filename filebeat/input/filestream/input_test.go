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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	loginp "github.com/elastic/beats/v7/filebeat/input/filestream/internal/input-logfile"
	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/statestore"
	"github.com/elastic/beats/v7/libbeat/statestore/storetest"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

// runFilestreamBenchmark runs the entire filestream input with the in-memory registry and the test pipeline.
// `testID` must be unique for each test run
// `cfg` must be a valid YAML string containing valid filestream configuration
// `expEventCount` is an expected amount of produced events
func runFilestreamBenchmark(b *testing.B, testID string, cfg string, expEventCount int) {
	logger := logp.L()
	c, err := conf.NewConfigWithYAML([]byte(cfg), cfg)
	require.NoError(b, err)

	p := Plugin(logger, createTestStore(b))
	input, err := p.Manager.Create(c)
	require.NoError(b, err)

	ctx, cancel := context.WithCancel(context.Background())
	context := v2.Context{
		Logger:      logger,
		ID:          testID,
		Cancelation: ctx,
	}

	connector, eventsDone := newTestPipeline(expEventCount)
	done := make(chan struct{})
	go func() {
		err := input.Run(context, connector)
		assert.NoError(b, err)
		done <- struct{}{}
	}()

	<-eventsDone
	cancel()
	<-done // for more stable results we should wait until the full shutdown
}

func generateFile(b *testing.B, lineCount int) string {
	b.Helper()
	dir := b.TempDir()
	file, err := os.CreateTemp(dir, "lines.log")
	require.NoError(b, err)

	for i := 0; i < lineCount; i++ {
		fmt.Fprintf(file, "rather mediocre log line message - %d\n", i)
	}
	filename := file.Name()
	err = file.Close()
	require.NoError(b, err)
	return filename
}

func BenchmarkFilestream(b *testing.B) {
	logp.TestingSetup(logp.ToDiscardOutput())
	lineCount := 10000
	filename := generateFile(b, lineCount)

	b.Run("filestream default throughput", func(b *testing.B) {
		cfg := `
type: filestream
prospector.scanner.check_interval: 1s
paths:
    - ` + filename + `
`
		for i := 0; i < b.N; i++ {
			runFilestreamBenchmark(b, fmt.Sprintf("default-benchmark-%d", i), cfg, lineCount)
		}
	})

	b.Run("filestream fingerprint throughput", func(b *testing.B) {
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
			runFilestreamBenchmark(b, fmt.Sprintf("fp-benchmark-%d", i), cfg, lineCount)
		}
	})
}

func createTestStore(t *testing.B) loginp.StateStore {
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

func newTestPipeline(eventLimit int) (pc beat.PipelineConnector, done <-chan struct{}) {
	ch := make(chan struct{})
	return &testPipeline{limit: eventLimit, done: ch}, ch
}

type testPipeline struct {
	done  chan struct{}
	limit int
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
	c.testPipeline.limit--
	if c.testPipeline.limit == 0 {
		c.testPipeline.done <- struct{}{}
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
