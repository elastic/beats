package memqueue

import (
	"flag"
	"math/rand"
	"testing"
	"time"

	"github.com/elastic/beats/libbeat/publisher/queue"
	"github.com/elastic/beats/libbeat/publisher/queue/queuetest"
)

var seed int64

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
				queuetest.TestSingleProducerConsumer(t, events, batchSize, factory)
			})
			t.Run("multi", func(t *testing.T) {
				queuetest.TestMultiProducerConsumer(t, events, batchSize, factory)
			})
		}
	}

	t.Run("direct", testWith(makeTestQueue(bufferSize, 0, 0)))
	t.Run("flush", testWith(makeTestQueue(bufferSize, batchSize/2, 100*time.Millisecond)))
}

func TestProducerCancelRemovesEvents(t *testing.T) {
	queuetest.TestProducerCancelRemovesEvents(t, makeTestQueue(1024, 0, 0))
}

func makeTestQueue(sz, minEvents int, flushTimeout time.Duration) queuetest.QueueFactory {
	return func(_ *testing.T) queue.Queue {
		return NewBroker(Settings{
			Events:         sz,
			FlushMinEvents: minEvents,
			FlushTimeout:   flushTimeout,
			WaitOnClose:    true,
		})
	}
}
