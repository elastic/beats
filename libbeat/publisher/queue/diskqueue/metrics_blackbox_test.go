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

package diskqueue_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/elastic/beats/v7/libbeat/publisher/queue"
	"github.com/elastic/beats/v7/libbeat/publisher/queue/diskqueue"
	"github.com/elastic/beats/v7/libbeat/publisher/queue/queuetest"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/monitoring"
	"github.com/elastic/elastic-agent-libs/paths"
)

func TestDiskQueueMetricsAfterBlockedPublish(t *testing.T) {
	settings := diskqueue.DefaultSettings()
	settings.Path = t.TempDir()
	settings.MaxBufferSize = 64 * 1024
	settings.MaxSegmentSize = 16 * 1024
	settings.ReadAheadLimit = 64
	settings.WriteAheadLimit = 1024

	reg := monitoring.NewRegistry()
	dq, err := diskqueue.NewQueue(
		logptest.NewTestingLogger(t, ""),
		queue.NewQueueObserver(reg),
		settings,
		nil,
		&paths.Path{},
	)
	require.NoError(t, err, "Disk queue should be created")

	var queueInstance queue.Queue[publisher.Event] = dq
	defer func() {
		assert.NoError(t, queueInstance.Close(true), "Disk queue cleanup should succeed")
		select {
		case <-queueInstance.Done():
		case <-time.After(5 * time.Second):
			assert.Fail(t, "Disk queue cleanup timed out")
		}
	}()

	producer := queueInstance.Producer(queue.ProducerConfig{})
	defer producer.Close()
	event := queuetest.MakeEvent(mapstr.M{
		"message": strings.Repeat("x", 4*1024),
	})

	const blockingCycles = 8
	type publishResult struct {
		accepted int
		ok       bool
	}

	accepted := 0
	consumed := 0
	for range blockingCycles {
		for {
			_, ok := producer.TryPublish(event)
			if !ok {
				break
			}
			accepted++
			require.Less(t, accepted, 200, "Disk queue should reach its configured capacity")
		}

		blockedResult := make(chan publishResult, 1)
		go func() {
			_, blockedPublishOK := producer.Publish(event)
			if !blockedPublishOK {
				blockedResult <- publishResult{}
				return
			}

			// Model a live input that keeps publishing after capacity is freed.
			// The follow-up request also dispatches the recovered event.
			_, nextPublishOK := producer.Publish(event)
			blockedResult <- publishResult{
				accepted: 2,
				ok:       nextPublishOK,
			}
		}()

		// Ensure Publish is blocking
		select {
		case result := <-blockedResult:
			require.FailNowf(t, "Publish should block on a full disk queue",
				"Publish returned early with results %+v", result)
		case <-time.After(50 * time.Millisecond):
		}

		publishUnblocked := false
		for !publishUnblocked {
			select {
			case result := <-blockedResult:
				require.True(t, result.ok,
					"Blocked and follow-up publishes should succeed after capacity is freed")
				require.Equal(t, 2, result.accepted,
					"Continuous producer should publish every expected event")
				accepted += result.accepted
				publishUnblocked = true
			default:
				batch := getDiskQueueBatch(t, queueInstance)
				consumed += batch.Count()
				batch.Done()
			}
		}
	}

	// Ensure the queue is drained
	for consumed < accepted {
		batch := getDiskQueueBatch(t, queueInstance)
		consumed += batch.Count()
		batch.Done()
	}

	producer.Close()
	require.NoError(t, queueInstance.Close(false), "Drained disk queue should close")
	select {
	case <-queueInstance.Done():
	case <-time.After(5 * time.Second):
		require.FailNow(t, "Drained disk queue should finish closing")
	}

	snapshot := monitoring.CollectFlatSnapshot(reg, monitoring.Full, false)
	assert.Equal(
		t,
		int64(accepted),
		snapshot.Ints["queue.added.events"],
		"Added events should include the publish that blocked",
	)

	assert.LessOrEqual(
		t,
		snapshot.Ints["queue.filled.events"],
		int64(accepted),
		"Filled event count should not exceed all accepted events",
	)

	assert.LessOrEqual(
		t,
		snapshot.Ints["queue.filled.bytes"],
		int64(settings.MaxBufferSize),
		"Filled byte count should not exceed queue capacity",
	)

	assert.GreaterOrEqual(
		t,
		snapshot.Floats["queue.filled.pct"],
		float64(0),
		"Queue fill percentage should not be negative",
	)

	assert.LessOrEqual(
		t,
		snapshot.Floats["queue.filled.pct"],
		float64(1),
		"Queue fill percentage should not exceed one",
	)
}

func getDiskQueueBatch(
	t *testing.T,
	queueInstance queue.Queue[publisher.Event],
) queue.Batch[publisher.Event] {
	t.Helper()

	type result struct {
		batch queue.Batch[publisher.Event]
		err   error
	}
	resultChan := make(chan result, 1)
	go func() {
		batch, err := queueInstance.Get(1)
		resultChan <- result{batch: batch, err: err}
	}()

	select {
	case result := <-resultChan:
		require.NoError(t, result.err, "Reading from disk queue should succeed")
		require.NotNil(t, result.batch, "Disk queue should return a batch")
		return result.batch
	case <-time.After(5 * time.Second):
		require.FailNow(t, "Timed out reading from disk queue")
		return nil
	}
}
