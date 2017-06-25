package brokertest

import (
	"sync"
	"testing"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/publisher/broker"
)

type BrokerFactory func() broker.Broker

type workerFactory func(*sync.WaitGroup, interface{}, *TestLogger, broker.Broker) func()

func TestSingleProducerConsumer(
	t *testing.T,
	events, batchSize int,
	factory BrokerFactory,
) {
	tests := []struct {
		name                 string
		producers, consumers workerFactory
	}{
		{
			"single producer, consumer without ack, complete batches",
			makeProducer(events, false, countEvent),
			makeConsumer(events, -1),
		},
		{
			"single producer, consumer, without ack, limited batches",
			makeProducer(events, false, countEvent),
			makeConsumer(events, batchSize),
		},
		{

			"single producer, consumer, with ack, complete batches",
			makeProducer(events, true, countEvent),
			makeConsumer(events, -1),
		},
		{
			"single producer, consumer, with ack, limited batches",
			makeProducer(events, true, countEvent),
			makeConsumer(events, batchSize),
		},
	}

	for _, test := range tests {
		t.Run(test.name, withLogOutput(func(t *testing.T) {
			log := NewTestLogger(t)
			log.Debug("run test: ", test.name)

			broker := factory()
			defer func() {
				err := broker.Close()
				if err != nil {
					t.Error(err)
				}
			}()

			var wg sync.WaitGroup

			go test.producers(&wg, nil, log, broker)()
			go test.consumers(&wg, nil, log, broker)()

			wg.Wait()
		}))
	}

}

func TestMultiProducerConsumer(
	t *testing.T,
	events, batchSize int,
	factory BrokerFactory,
) {
	tests := []struct {
		name                 string
		producers, consumers workerFactory
	}{
		{
			"2 producers, 1 consumer, without ack, complete batches",
			multiple(
				makeProducer(events, false, countEvent),
				makeProducer(events, false, countEvent),
			),
			makeConsumer(events*2, -1),
		},
		{
			"2 producers, 1 consumer, all ack, complete batches",
			multiple(
				makeProducer(events, true, countEvent),
				makeProducer(events, true, countEvent),
			),
			makeConsumer(events*2, -1),
		},
		{
			"2 producers, 1 consumer, 1 ack, complete batches",
			multiple(
				makeProducer(events, true, countEvent),
				makeProducer(events, false, countEvent),
			),
			makeConsumer(events*2, -1),
		},
		{
			"2 producers, 1 consumer, without ack, limited batches",
			multiple(
				makeProducer(events, false, countEvent),
				makeProducer(events, false, countEvent),
			),
			makeConsumer(events*2, batchSize),
		},
		{
			"2 producers, 1 consumer, all ack, limited batches",
			multiple(
				makeProducer(events, true, countEvent),
				makeProducer(events, true, countEvent),
			),
			makeConsumer(events*2, batchSize),
		},
		{
			"2 producers, 1 consumer, 1 ack, limited batches",
			multiple(
				makeProducer(events, true, countEvent),
				makeProducer(events, false, countEvent),
			),
			makeConsumer(events*2, batchSize),
		},

		{
			"1 producer, 2 consumers, without ack, complete batches",
			makeProducer(events, true, countEvent),
			multiConsumer(2, events, -1),
		},
		{
			"1 producer, 2 consumers, without ack, limited batches",
			makeProducer(events, true, countEvent),
			multiConsumer(2, events, -1),
		},

		{
			"2 producers, 2 consumer, without ack, complete batches",
			multiple(
				makeProducer(events, false, countEvent),
				makeProducer(events, false, countEvent),
			),
			multiConsumer(2, events*2, -1),
		},
		{
			"2 producers, 2 consumer, all ack, complete batches",
			multiple(
				makeProducer(events, true, countEvent),
				makeProducer(events, true, countEvent),
			),
			multiConsumer(2, events*2, -1),
		},
		{
			"2 producers, 2 consumer, 1 ack, complete batches",
			multiple(
				makeProducer(events, true, countEvent),
				makeProducer(events, false, countEvent),
			),
			multiConsumer(2, events*2, -1),
		},
		{
			"2 producers, 2 consumer, without ack, limited batches",
			multiple(
				makeProducer(events, false, countEvent),
				makeProducer(events, false, countEvent),
			),
			multiConsumer(2, events*2, batchSize),
		},
		{
			"2 producers, 2 consumer, all ack, limited batches",
			multiple(
				makeProducer(events, true, countEvent),
				makeProducer(events, true, countEvent),
			),
			multiConsumer(2, events*2, batchSize),
		},
		{
			"2 producers, 2 consumer, 1 ack, limited batches",
			multiple(
				makeProducer(events, true, countEvent),
				makeProducer(events, false, countEvent),
			),
			multiConsumer(2, events*2, batchSize),
		},
	}

	for _, test := range tests {
		t.Run(test.name, withLogOutput(func(t *testing.T) {
			log := NewTestLogger(t)
			log.Debug("run test: ", test.name)

			broker := factory()
			defer func() {
				err := broker.Close()
				if err != nil {
					t.Error(err)
				}
			}()

			var wg sync.WaitGroup

			go test.producers(&wg, nil, log, broker)()
			go test.consumers(&wg, nil, log, broker)()

			wg.Wait()
		}))
	}
}

func multiple(
	fns ...workerFactory,
) workerFactory {
	return func(wg *sync.WaitGroup, info interface{}, log *TestLogger, broker broker.Broker) func() {
		runners := make([]func(), len(fns))
		for i, gen := range fns {
			runners[i] = gen(wg, info, log, broker)
		}

		return func() {
			for _, r := range runners {
				go r()
			}
		}
	}
}

func makeProducer(
	maxEvents int,
	waitACK bool,
	makeFields func(int) common.MapStr,
) func(*sync.WaitGroup, interface{}, *TestLogger, broker.Broker) func() {
	return func(wg *sync.WaitGroup, info interface{}, log *TestLogger, b broker.Broker) func() {
		wg.Add(1)
		return func() {
			defer wg.Done()

			log.Debug("start producer")
			defer log.Debug("stop producer")

			var (
				ackWG sync.WaitGroup
				ackCB func(int)
			)

			if waitACK {
				ackWG.Add(maxEvents)

				total := 0
				ackCB = func(N int) {
					total += N
					log.Debugf("producer ACK: N=%v, total=%v\n", N, total)

					for i := 0; i < N; i++ {
						ackWG.Done()
					}
				}
			}

			producer := b.Producer(broker.ProducerConfig{
				ACK: ackCB,
			})
			for i := 0; i < maxEvents; i++ {
				producer.Publish(makeEvent(makeFields(i)))
			}

			ackWG.Wait()
		}
	}
}

func makeConsumer(maxEvents, batchSize int) workerFactory {
	return multiConsumer(1, maxEvents, batchSize)
}

func multiConsumer(numConsumers, maxEvents, batchSize int) workerFactory {
	return func(wg *sync.WaitGroup, info interface{}, log *TestLogger, b broker.Broker) func() {
		wg.Add(1)
		return func() {
			defer wg.Done()

			var events sync.WaitGroup

			consumers := make([]broker.Consumer, numConsumers)
			for i := range consumers {
				consumers[i] = b.Consumer()
			}

			events.Add(maxEvents)

			for _, c := range consumers {
				c := c

				wg.Add(1)
				go func() {
					defer wg.Done()

					for {
						batch, err := c.Get(batchSize)
						if err != nil {
							return
						}

						for range batch.Events() {
							events.Done()
						}
						batch.ACK()
					}
				}()
			}

			events.Wait()

			// disconnect consumers
			for _, c := range consumers {
				c.Close()
			}
		}
	}
}

func countEvent(i int) common.MapStr {
	return common.MapStr{
		"count": i,
	}
}
