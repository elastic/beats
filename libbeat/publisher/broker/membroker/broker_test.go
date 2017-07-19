package membroker

import (
	"flag"
	"math/rand"
	"testing"
	"time"

	"github.com/elastic/beats/libbeat/publisher/broker"
	"github.com/elastic/beats/libbeat/publisher/broker/brokertest"
)

var seed int64

func init() {
	flag.Int64Var(&seed, "seed", time.Now().UnixNano(), "test random seed")
}

func TestProduceConsumer(t *testing.T) {
	maxEvents := 1024
	minEvents := 32

	rand.Seed(seed)
	events := rand.Intn(maxEvents-minEvents) + maxEvents
	batchSize := rand.Intn(events-8) + 4
	bufferSize := rand.Intn(batchSize*2) + 4

	// events := 4
	// batchSize := 1
	// bufferSize := 2

	t.Log("seed: ", seed)
	t.Log("events: ", events)
	t.Log("batchSize: ", batchSize)
	t.Log("bufferSize: ", bufferSize)

	factory := makeTestBroker(bufferSize)

	t.Run("single", func(t *testing.T) {
		brokertest.TestSingleProducerConsumer(t, events, batchSize, factory)
	})
	t.Run("multi", func(t *testing.T) {
		brokertest.TestMultiProducerConsumer(t, events, batchSize, factory)
	})
}

func TestProducerCancelRemovesEvents(t *testing.T) {
	brokertest.TestProducerCancelRemovesEvents(t, makeTestBroker(1024))
}

func makeTestBroker(sz int) brokertest.BrokerFactory {
	return func() broker.Broker {
		return NewBroker(Settings{Events: sz, WaitOnClose: true})
	}
}
