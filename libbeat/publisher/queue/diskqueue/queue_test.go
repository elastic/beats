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

package diskqueue

import (
	"flag"
	"math/rand/v2"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/elastic/beats/v7/libbeat/publisher/queue"
	"github.com/elastic/beats/v7/libbeat/publisher/queue/queuetest"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/paths"
	"github.com/elastic/elastic-agent-libs/testing/fs"
)

var seed int64

type testQueue struct {
	*diskQueue
}

func init() {
	flag.Int64Var(&seed, "seed", time.Now().UnixNano(), "test random seed")
}

func TestProduceConsumer(t *testing.T) {
	maxEvents := 1024
	minEvents := 32

	r := rand.New(rand.NewPCG(uint64(seed), 0)) //nolint:gosec //Safe to ignore in tests
	events := r.IntN(maxEvents-minEvents) + minEvents
	batchSize := r.IntN(events-8) + 4
	bufferSize := r.IntN(batchSize*2) + 4

	// events := 4
	// batchSize := 1
	// bufferSize := 2

	t.Log("seed: ", seed)
	t.Log("events: ", events)
	t.Log("batchSize: ", batchSize)
	t.Log("bufferSize: ", bufferSize)

	testWith := func(factory queuetest.QueueFactory) func(t *testing.T) {
		return func(t *testing.T) {
			t.Run("single", func(t *testing.T) {
				t.Parallel()
				queuetest.TestSingleProducerConsumer(t, events, batchSize, factory)
			})
			t.Run("multi", func(t *testing.T) {
				t.Parallel()
				queuetest.TestMultiProducerConsumer(t, events, batchSize, factory)
			})
		}
	}

	t.Run("direct", testWith(makeTestQueue()))
}

func makeTestQueue() queuetest.QueueFactory {
	return func(t *testing.T) queue.Queue[publisher.Event] {
		dir := t.TempDir()
		settings := DefaultSettings()
		settings.Path = dir
		logger := logptest.NewTestingLogger(t, "")
		queue, _ := NewQueue(logger, nil, settings, nil, &paths.Path{})
		return testQueue{
			diskQueue: queue,
		}
	}
}

func (t testQueue) Close(force bool) error {
	err := t.diskQueue.Close(force)
	return err
}

func TestQueueDoesNotReplayLastEventAfterRestart(t *testing.T) {
	workDir := fs.TempDir(t, "..", "..", "..", "build", "integration-tests")
	diskQueuePath := filepath.Join(workDir, "queue")
	settings := DefaultSettings()
	settings.Path = diskQueuePath
	// Keep segment size small enough to produce multiple segments quickly.
	settings.MaxSegmentSize = 4 * 1024

	fileLogger := logptest.NewFileLogger(t, workDir)

	// Run 1: publish and ACK two events.
	run1Queue, err := NewQueue(fileLogger.Logger, nil, settings, nil, &paths.Path{})
	require.NoError(t, err, "run1 queue should be created successfully")

	run1Producer := run1Queue.Producer(queue.ProducerConfig{})
	publishAndACKSingleEvent(t, run1Queue, run1Producer, "event-1")
	publishAndACKSingleEvent(t, run1Queue, run1Producer, "event-2")
	run1Producer.Close()
	closeQueueAndWait(t, run1Queue)

	// Run 2: reopen queue, publish one event and ACK it.
	run2Queue, err := NewQueue(fileLogger.Logger, nil, settings, nil, &paths.Path{})
	require.NoError(t, err, "run2 queue should be created successfully")

	run2Producer := run2Queue.Producer(queue.ProducerConfig{})
	publishAndACKSingleEvent(t, run2Queue, run2Producer, "event-3")
	run2Producer.Close()
	closeQueueAndWait(t, run2Queue)

	// Run 3: reopen queue without publishing a new event. Correct behavior is
	// that no event is replayed. This used to fail with the last event being
	// replayed.
	run3Queue, err := NewQueue(fileLogger.Logger, nil, settings, nil, &paths.Path{})
	require.NoError(t, err, "run3 queue should be created successfully")

	replayedBatch := readBatch(t, run3Queue, 3*time.Second)
	if replayedBatch != nil {
		count := replayedBatch.Count()
		var msg any
		if count > 0 {
			msg, _ = replayedBatch.Entry(0).Content.Fields.GetValue("message")
		}
		replayedBatch.Done()
		t.Fatalf("unexpected replayed event after restart"+
			"found replayed batch with count=%d and first message=%v",
			count,
			msg,
		)
	}
	closeQueueAndWait(t, run3Queue)
}

func publishAndACKSingleEvent(
	t *testing.T,
	queueInstance *diskQueue,
	producer queue.Producer[publisher.Event],
	msg string,
) {
	_, ok := producer.Publish(makeDiskQueueTestEvent(msg))
	require.True(t, ok, "publishing test event %q should succeed", msg)

	batch := readBatch(t, queueInstance, 3*time.Second)
	require.NotNil(t, batch, "queue should return a batch for message %q", msg)
	require.Equal(t, 1, batch.Count(), "queue should return a single event batch for message %q", msg)
	assertEventMessage(t, batch.Entry(0), msg)
	batch.Done()
}

func readBatch(t *testing.T, queueInstance *diskQueue, timeout time.Duration) queue.Batch[publisher.Event] {
	type getResult struct {
		batch queue.Batch[publisher.Event]
		err   error
	}

	results := make(chan getResult, 1)
	go func() {
		batch, err := queueInstance.Get(1)
		results <- getResult{batch: batch, err: err}
	}()

	select {
	case result := <-results:
		require.NoError(t, result.err, "reading from queue should not return an error")
		return result.batch
	case <-time.After(timeout):
		return nil
	}
}

func closeQueueAndWait(t *testing.T, queueInstance *diskQueue) {
	err := queueInstance.Close(false)
	require.NoError(t, err, "closing queue should not return an error")

	select {
	case <-queueInstance.Done():
	case <-time.After(5 * time.Second):
		require.Fail(t, "queue did not close in time", "queue.Done() should close within timeout")
	}
}

func assertEventMessage(t *testing.T, event publisher.Event, expectedMsg string) {
	msg, _ := event.Content.Fields.GetValue("message")
	assert.Equal(t, expectedMsg, msg, "unexpected message in consumed event")
}

func makeDiskQueueTestEvent(msg string) publisher.Event {
	return queuetest.MakeEvent(mapstr.M{
		"message": msg,
		"payload": strings.Repeat("x", 2048),
	})
}
