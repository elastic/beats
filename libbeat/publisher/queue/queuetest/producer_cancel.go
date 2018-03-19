package queuetest

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/publisher"
	"github.com/elastic/beats/libbeat/publisher/queue"
)

// TestSingleProducerConsumer tests buffered events for a producer getting
// cancelled will not be consumed anymore. Concurrent producer/consumer pairs
// might still have active events not yet ACKed (not tested here).
//
// Note: queues not requiring consumers to ACK a events in order to
//       return ACKs to the producer are not supported by this test.
func TestProducerCancelRemovesEvents(t *testing.T, factory QueueFactory) {
	fn := withLogOutput(func(t *testing.T) {
		var (
			i  int
			N1 = 3
			N2 = 10
		)

		log := NewTestLogger(t)
		b := factory(t)
		defer b.Close()

		log.Debug("create first producer")
		producer := b.Producer(queue.ProducerConfig{
			ACK:          func(int) {}, // install function pointer, so 'cancel' will remove events
			DropOnCancel: true,
		})

		for ; i < N1; i++ {
			log.Debugf("send event %v to first producer", i)
			producer.Publish(makeEvent(common.MapStr{
				"value": i,
			}))
		}

		// cancel producer
		log.Debugf("cancel producer")
		producer.Cancel()

		// reconnect and send some more events
		log.Debug("connect new producer")
		producer = b.Producer(queue.ProducerConfig{})
		for ; i < N2; i++ {
			log.Debugf("send event %v to new producer", i)
			producer.Publish(makeEvent(common.MapStr{
				"value": i,
			}))
		}

		// consumer all events
		consumer := b.Consumer()
		total := N2 - N1
		events := make([]publisher.Event, 0, total)
		for len(events) < total {
			batch, err := consumer.Get(-1) // collect all events
			if err != nil {
				panic(err)
			}

			events = append(events, batch.Events()...)
			batch.ACK()
		}

		// verify
		if total != len(events) {
			assert.Equal(t, total, len(events))
			return
		}

		for i, event := range events {
			value := event.Content.Fields["value"].(int)
			assert.Equal(t, i+N1, value)
		}
	})

	fn(t)
}
