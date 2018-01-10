package queue

import (
	"sync"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/publisher"
)

// Wraps Queue interface to add QoS functionality on top of it.
type QueueWrapper struct {
	q   Queue
	qos Qoser
}

// TODO: Add config reading capability
func NewQueueWrapper(q Queue, _ *common.Config) Queue {
	if q != nil {
		qw := &QueueWrapper{
			q:   q,
			qos: NewWeightedScheduler(),
		}

		qw.qos.Schedule()
		return qw
	}
	return nil
}

func (q *QueueWrapper) BufferConfig() BufferConfig {
	return q.q.BufferConfig()
}

func (q *QueueWrapper) Producer(cfg ProducerConfig) Producer {
	return newProducerWrapper(q.q.Producer(cfg), cfg.Weight, q.qos)
}

func (q *QueueWrapper) Consumer() Consumer {
	return q.q.Consumer()
}

func (q *QueueWrapper) Close() error {
	return q.q.Close()
}

// Wraps Producer interface to add QoS functionality on top of it.
type ProducerWrapper struct {
	QoserClient
	p Producer

	wg     sync.WaitGroup
	weight int
}

func newProducerWrapper(p Producer, weight int, qos Qoser) *ProducerWrapper {
	if p != nil {
		return &ProducerWrapper{
			p:           p,
			QoserClient: qos.CreateClient(),
			weight:      weight,
		}
	}
	return nil
}

func (p *ProducerWrapper) Publish(event publisher.Event) bool {
	p.waitForPublish(event)
	return p.p.Publish(event)
}

func (p *ProducerWrapper) TryPublish(event publisher.Event) bool {
	p.waitForPublish(event)
	return p.p.TryPublish(event)
}

func (p *ProducerWrapper) Cancel() int {
	return p.p.Cancel()
}

func (p *ProducerWrapper) waitForPublish(event publisher.Event) {
	p.wg.Add(1)
	go func() {
		// Request for time to publish the incoming Event
		p.RequestToken(p, p.weight, event.Content.Timestamp)
		// Wait for the producer to get a cycle to publish
		p.Wait()
		p.wg.Done()
	}()

	// Wait till the go-routine is done signifying that the scheduler has allocated
	// time for the publisher to publish the event
	p.wg.Wait()
}
