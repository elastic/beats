package queue

import (
	"container/heap"
	"time"

	"github.com/elastic/beats/libbeat/common"
)

type Qoser interface {
	// Release the process lock and give the processing time to the next available producer
	ReleaseToken()

	// Schedule picks the next producer who gets to produce to the queue
	Schedule()

	// Clone an instance of QoSer
	CreateClient() QoserClient
}

type QoserClient interface {
	// Add the producer in the wait queue based
	RequestToken(*ProducerWrapper, int, time.Time)

	// Wait until the next available slot
	Wait()

	// Send a signal to the producer to wake up
	Awaken()
}

type weightedScheduler struct {
	heap heap.Interface

	notify chan struct{}
}

func NewWeightedScheduler() Qoser {
	return &weightedScheduler{
		heap:   NewHeap(),
		notify: make(chan struct{}),
	}
}

func (w *weightedScheduler) ReleaseToken() {
	// Pop the most high priority producer
	iface := w.heap.Pop()

	if iface == nil {
		return
	}

	if it, ok := iface.(*item); ok {
		producer := it.value.(*ProducerWrapper)
		// Notify the producer to wake up.
		producer.Awaken()
	}
}

func (w *weightedScheduler) Schedule() {
	go func() {
		for {
			select {
			case <-w.notify:
				w.ReleaseToken()
			}
		}
	}()
}

func (w *weightedScheduler) CreateClient() QoserClient {
	return NewWeightedClient(w.heap, w.notify)
}

type weightedClient struct {
	heap heap.Interface

	cond   chan struct{}
	notify chan struct{}
}

func NewWeightedClient(h heap.Interface, notify chan struct{}) QoserClient {
	return &weightedClient{
		heap:   h,
		cond:   make(chan struct{}, 1),
		notify: notify,
	}
}

func (c *weightedClient) RequestToken(p *ProducerWrapper, weight int, time time.Time) {
	// Calculate priority and add the producer into the queue
	priority := c.computePriority(weight, time)
	newItem := &item{
		value:    p,
		priority: priority,
	}
	c.heap.Push(newItem)
	c.notify <- struct{}{}
}

func (c *weightedClient) Wait() {
	<-c.cond
}

// TODO: Add the packet delivery logic after initial review
func (c *weightedClient) computePriority(weight int, t time.Time) common.Float {
	return common.Float(int64(weight))
}

func (c *weightedClient) Awaken() {
	c.cond <- struct{}{}
}
