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

package pipeline

import (
	"fmt"
	"math"
	"strings"
	"sync"
	"testing"
	"testing/quick"
	"time"

	"go.elastic.co/apm/v2/apmtest"

	"github.com/stretchr/testify/require"

	"github.com/menderesk/beats/v7/libbeat/common/atomic"
	"github.com/menderesk/beats/v7/libbeat/internal/testutil"
	"github.com/menderesk/beats/v7/libbeat/outputs"
	"github.com/menderesk/beats/v7/libbeat/publisher"
)

func TestMakeClientWorker(t *testing.T) {
	tests := map[string]func(mockPublishFn) outputs.Client{
		"client":         newMockClient,
		"network_client": newMockNetworkClient,
	}

	for name, ctor := range tests {
		t.Run(name, func(t *testing.T) {
			testutil.SeedPRNG(t)

			err := quick.Check(func(i uint) bool {
				numBatches := 300 + (i % 100) // between 300 and 399
				var numEvents uint

				logger := makeBufLogger(t)

				workQueue := make(chan publisher.Batch)
				retryer := newStandaloneRetryer(workQueue)
				defer retryer.close()

				var published atomic.Uint
				publishFn := func(batch publisher.Batch) error {
					published.Add(uint(len(batch.Events())))
					return nil
				}

				client := ctor(publishFn)

				worker := makeClientWorker(nilObserver, workQueue, client, logger, nil)
				defer worker.Close()

				for i := uint(0); i < numBatches; i++ {
					batch := randomBatch(50, 150).withRetryer(retryer)
					numEvents += uint(len(batch.Events()))
					workQueue <- batch
				}

				// Give some time for events to be published
				timeout := 20 * time.Second

				// Make sure that all events have eventually been published
				success := waitUntilTrue(timeout, func() bool {
					return numEvents == published.Load()
				})
				if !success {
					logger.Flush()
					t.Logf("numBatches = %v, numEvents = %v, published = %v", numBatches, numEvents, published)
				}
				return success
			}, nil)

			if err != nil {
				t.Error(err)
			}
		})
	}
}

func TestReplaceClientWorker(t *testing.T) {
	t.Skip("Flaky test: https://github.com/menderesk/beats/issues/17965")

	tests := map[string]func(mockPublishFn) outputs.Client{
		"client":         newMockClient,
		"network_client": newMockNetworkClient,
	}

	const minEventsInBatch = 50
	const maxEventsInBatch = 150

	for name, ctor := range tests {
		t.Run(name, func(t *testing.T) {
			testutil.SeedPRNG(t)

			err := quick.Check(func(i uint) bool {
				numBatches := 1000 + (i % 100) // between 1000 and 1099

				logger := makeBufLogger(t)

				workQueue := make(chan publisher.Batch)
				retryer := newStandaloneRetryer(workQueue)
				defer retryer.close()

				var batches []publisher.Batch
				var numEvents int
				for i := uint(0); i < numBatches; i++ {
					batch := randomBatch(
						minEventsInBatch, maxEventsInBatch,
					).withRetryer(retryer)
					batch.events[0].Content.Private = i
					numEvents += batch.Len()
					batches = append(batches, batch)
				}

				var wg sync.WaitGroup
				wg.Add(1)
				go func() {
					defer wg.Done()
					for _, batch := range batches {
						t.Logf("publish batch: %v", batch.(*mockBatch).events[0].Content.Private)
						workQueue <- batch
					}
				}()

				// Publish at least 1 batch worth of events but no more than 20% events
				publishLimit := uint(math.Max(minEventsInBatch, float64(numEvents)*0.2))

				var publishedFirst atomic.Uint
				blockCtrl := make(chan struct{})
				blockingPublishFn := func(batch publisher.Batch) error {
					// Emulate blocking. Upon unblocking the in-flight batch that was
					// blocked is published.
					if publishedFirst.Load() >= publishLimit {
						<-blockCtrl
					}

					count := len(batch.Events())
					publishedFirst.Add(uint(count))
					t.Logf("#1 processed batch: %v (%v)", batch.(*mockBatch).events[0].Content.Private, count)
					return nil
				}

				client := ctor(blockingPublishFn)
				worker := makeClientWorker(nilObserver, workQueue, client, logger, nil)

				// Allow the worker to make *some* progress before we close it
				timeout := 10 * time.Second
				progress := waitUntilTrue(timeout, func() bool {
					return publishedFirst.Load() >= publishLimit
				})
				if !progress {
					return false
				}

				// Close worker before all batches have had time to be published
				err := worker.Close()
				require.NoError(t, err)

				close(blockCtrl)

				// Start new worker to drain work queue
				var publishedLater atomic.Uint
				countingPublishFn := func(batch publisher.Batch) error {
					count := len(batch.Events())
					publishedLater.Add(uint(count))
					t.Logf("#2 processed batch: %v (%v)", batch.(*mockBatch).events[0].Content.Private, count)
					return nil
				}

				client = ctor(countingPublishFn)
				makeClientWorker(nilObserver, workQueue, client, logger, nil)
				wg.Wait()

				// Make sure that all events have eventually been published
				timeout = 20 * time.Second
				success := waitUntilTrue(timeout, func() bool {
					return numEvents == int(publishedFirst.Load()+publishedLater.Load())
				})
				if !success {
					logger.Flush()
					t.Logf("numBatches = %v, numEvents = %v, publishedFirst = %v, publishedLater = %v",
						numBatches, numEvents, publishedFirst.Load(), publishedLater.Load())
				}
				return success
			}, &quick.Config{MaxCount: 25})

			if err != nil {
				t.Error(err)
			}
		})
	}
}

func TestMakeClientTracer(t *testing.T) {
	testutil.SeedPRNG(t)

	numBatches := 10
	var numEvents uint

	logger := makeBufLogger(t)

	workQueue := make(chan publisher.Batch)
	retryer := newStandaloneRetryer(workQueue)
	defer retryer.close()

	var published atomic.Uint
	publishFn := func(batch publisher.Batch) error {
		published.Add(uint(len(batch.Events())))
		return nil
	}

	client := newMockNetworkClient(publishFn)

	recorder := apmtest.NewRecordingTracer()
	defer recorder.Close()

	worker := makeClientWorker(nilObserver, workQueue, client, logger, recorder.Tracer)
	defer worker.Close()

	for i := 0; i < numBatches; i++ {
		batch := randomBatch(10, 15).withRetryer(retryer)
		numEvents += uint(len(batch.Events()))
		workQueue <- batch
	}

	// Give some time for events to be published
	timeout := 10 * time.Second

	// Make sure that all events have eventually been published
	matches := waitUntilTrue(timeout, func() bool {
		return numEvents == published.Load()
	})
	if !matches {
		t.Errorf("expected %d events, got %d", numEvents, published)
	}
	recorder.Flush(nil)

	apmEvents := recorder.Payloads()
	transactions := apmEvents.Transactions
	if len(transactions) != numBatches {
		logger.Flush()
		t.Errorf("expected %d traces, got %d", numBatches, len(transactions))
	}
}

// bufLogger is a buffered logger. It does not immediately print out log lines; instead it
// buffers them. To print them out, one must explicitly call it's Flush() method. This is
// useful when you want to see the logs only when tests fail but not when they pass.
type bufLogger struct {
	t     *testing.T
	lines []string
	mu    sync.RWMutex
}

func (l *bufLogger) Debug(vs ...interface{})              { l.report("DEBUG", vs) }
func (l *bufLogger) Debugf(fmt string, vs ...interface{}) { l.reportf("DEBUG ", fmt, vs) }

func (l *bufLogger) Info(vs ...interface{})              { l.report("INFO", vs) }
func (l *bufLogger) Infof(fmt string, vs ...interface{}) { l.reportf("INFO", fmt, vs) }

func (l *bufLogger) Error(vs ...interface{})              { l.report("ERROR", vs) }
func (l *bufLogger) Errorf(fmt string, vs ...interface{}) { l.reportf("ERROR", fmt, vs) }

func (l *bufLogger) report(level string, vs []interface{}) {
	str := strings.TrimRight(strings.Repeat("%v ", len(vs)), " ")
	l.reportf(level, str, vs)
}
func (l *bufLogger) reportf(level, str string, vs []interface{}) {
	str = level + ": " + str
	line := fmt.Sprintf(str, vs...)

	l.mu.Lock()
	defer l.mu.Unlock()
	l.lines = append(l.lines, line)
}

func (l *bufLogger) Flush() {
	l.mu.Lock()
	defer l.mu.Unlock()

	for _, line := range l.lines {
		l.t.Log(line)
	}

	l.lines = make([]string, 0)
}

func makeBufLogger(t *testing.T) *bufLogger {
	return &bufLogger{
		t:     t,
		lines: make([]string, 0),
	}
}
