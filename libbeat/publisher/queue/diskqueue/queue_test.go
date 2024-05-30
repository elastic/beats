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
	"io/ioutil"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/elastic/beats/v7/libbeat/publisher/queue"
	"github.com/elastic/beats/v7/libbeat/publisher/queue/queuetest"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/stretchr/testify/require"
)

var seed int64

type testQueue struct {
	*diskQueue
	teardown func()
}

func init() {
	flag.Int64Var(&seed, "seed", time.Now().UnixNano(), "test random seed")
}

func TestProduceConsumer(t *testing.T) {
	maxEvents := 1024
	minEvents := 32

	rand.Seed(seed)
	events := rand.Intn(maxEvents-minEvents) + minEvents
	batchSize := rand.Intn(events-8) + 4
	bufferSize := rand.Intn(batchSize*2) + 4

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

func TestMetrics(t *testing.T) {
	dir, err := ioutil.TempDir("", "diskqueue_metrics")
	defer func() {
		_ = os.RemoveAll(dir)
	}()
	require.NoError(t, err)
	settings := DefaultSettings()
	settings.Path = dir
	// lower max segment size so we can get multiple segments
	settings.MaxSegmentSize = 100

	testQueue, err := NewQueue(logp.L(), nil, settings, nil)
	require.NoError(t, err)
	defer testQueue.Close()

	eventsToTest := 100

	// Send events to queue
	producer := testQueue.Producer(queue.ProducerConfig{})
	sendEventsToQueue(eventsToTest, producer)

	// fetch metrics before we read any events
	time.Sleep(time.Millisecond * 500)
	testMetrics, err := testQueue.Metrics()
	require.NoError(t, err)

	require.Equal(t, testMetrics.ByteLimit.ValueOr(0), uint64((1 << 30)))
	require.NotZero(t, testMetrics.ByteCount.ValueOr(0))
	t.Logf("got %d bytes written", testMetrics.ByteCount.ValueOr(0))

}

func sendEventsToQueue(count int, prod queue.Producer) {
	for i := 0; i < count; i++ {
		prod.Publish(queuetest.MakeEvent(mapstr.M{"count": i}))
	}
}

func makeTestQueue() queuetest.QueueFactory {
	return func(t *testing.T) queue.Queue {
		dir, err := ioutil.TempDir("", "diskqueue_test")
		if err != nil {
			t.Fatal(err)
		}
		settings := DefaultSettings()
		settings.Path = dir
		queue, _ := NewQueue(logp.L(), nil, settings, nil)
		return testQueue{
			diskQueue: queue,
			teardown: func() {
				os.RemoveAll(dir)
			},
		}
	}
}

func (t testQueue) Close() error {
	err := t.diskQueue.Close()
	t.teardown()
	return err
}
