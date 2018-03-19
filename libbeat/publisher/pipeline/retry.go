package pipeline

import (
	"github.com/elastic/beats/libbeat/logp"
)

// retryer is responsible for accepting and managing failed send attempts. It
// will also accept not yet published events from outputs being dynamically closed
// by the controller. Cancelled batches will be forwarded to the new workQueue,
// without updating the events retry counters.
// If too many batches (number of outputs/3) are stored in the retry buffer,
// will the consumer be paused, until some batches have been processed by some
// outputs.
type retryer struct {
	logger   *logp.Logger
	observer outputObserver

	done chan struct{}

	consumer *eventConsumer

	sig chan retryerSignal
	out workQueue
	in  retryQueue
}

type retryQueue chan batchEvent

type retryerSignal struct {
	tag     retryerEventTag
	channel workQueue
}

type batchEvent struct {
	tag   retryerBatchTag
	batch *Batch
}

type retryerEventTag uint8

const (
	sigRetryerOutputAdded retryerEventTag = iota
	sigRetryerOutputRemoved
	sigRetryerUpdateOutput
)

type retryerBatchTag uint8

const (
	retryBatch retryerBatchTag = iota
	cancelledBatch
)

func newRetryer(
	log *logp.Logger,
	observer outputObserver,
	out workQueue,
	c *eventConsumer,
) *retryer {
	r := &retryer{
		logger:   log,
		observer: observer,
		done:     make(chan struct{}),
		sig:      make(chan retryerSignal, 3),
		in:       retryQueue(make(chan batchEvent, 3)),
		out:      out,
		consumer: c,
	}
	go r.loop()
	return r
}

func (r *retryer) close() {
	close(r.done)
}

func (r *retryer) sigOutputAdded() {
	r.sig <- retryerSignal{tag: sigRetryerOutputAdded}
}

func (r *retryer) sigOutputRemoved() {
	r.sig <- retryerSignal{tag: sigRetryerOutputRemoved}
}

func (r *retryer) updOutput(ch workQueue) {
	r.sig <- retryerSignal{
		tag:     sigRetryerUpdateOutput,
		channel: ch,
	}
}

func (r *retryer) retry(b *Batch) {
	r.in <- batchEvent{tag: retryBatch, batch: b}
}

func (r *retryer) cancelled(b *Batch) {
	r.in <- batchEvent{tag: cancelledBatch, batch: b}
}

func (r *retryer) loop() {
	var (
		out             workQueue
		consumerBlocked bool

		active     *Batch
		activeSize int
		buffer     []*Batch
		numOutputs int

		log = r.logger
	)

	for {
		select {
		case <-r.done:
			return

		case evt := <-r.in:
			var (
				countFailed  int
				countDropped int
				batch        = evt.batch
				countRetry   = len(batch.events)
			)

			if evt.tag == retryBatch {
				countFailed = len(batch.events)
				r.observer.eventsFailed(countFailed)

				decBatch(batch)

				countRetry = len(batch.events)
				countDropped = countFailed - countRetry
				r.observer.eventsDropped(countDropped)
			}

			if len(batch.events) == 0 {
				log.Info("Drop batch")
				batch.Drop()
			} else {
				out = r.out
				buffer = append(buffer, batch)
				out = r.out
				active = buffer[0]
				activeSize = len(active.events)
				if !consumerBlocked {
					consumerBlocked = blockConsumer(numOutputs, len(buffer))
					if consumerBlocked {
						log.Info("retryer: send wait signal to consumer")
						r.consumer.sigWait()
						log.Info("  done")
					}
				}
			}

		case out <- active:
			r.observer.eventsRetry(activeSize)

			buffer = buffer[1:]
			active, activeSize = nil, 0

			if len(buffer) == 0 {
				out = nil
			} else {
				active = buffer[0]
				activeSize = len(active.events)
			}

			if consumerBlocked {
				consumerBlocked = blockConsumer(numOutputs, len(buffer))
				if !consumerBlocked {
					log.Info("retryer: send unwait-signal to consumer")
					r.consumer.sigUnWait()
					log.Info("  done")
				}
			}

		case sig := <-r.sig:
			switch sig.tag {
			case sigRetryerUpdateOutput:
				r.out = sig.channel
			case sigRetryerOutputAdded:
				numOutputs++
			case sigRetryerOutputRemoved:
				numOutputs--
			}
		}
	}
}

func blockConsumer(numOutputs, numBatches int) bool {
	return numBatches/3 >= numOutputs
}

func decBatch(batch *Batch) {
	if batch.ttl <= 0 {
		return
	}

	batch.ttl--
	if batch.ttl > 0 {
		return
	}

	// filter for evens with guaranteed send flags
	events := batch.events[:0]
	for _, event := range batch.events {
		if event.Guaranteed() {
			events = append(events, event)
		}
	}
	batch.events = events
}
