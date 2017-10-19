package pipeline

import (
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/publisher/queue"
)

// outputController manages the pipelines output capabilities, like:
// - start
// - stop
// - reload
type outputController struct {
	logger   *logp.Logger
	observer outputObserver

	queue queue.Queue

	retryer  *retryer
	consumer *eventConsumer
	out      *outputGroup
}

// outputGroup configures a group of load balanced outputs with shared work queue.
type outputGroup struct {
	workQueue workQueue
	outputs   []outputWorker

	batchSize  int
	timeToLive int // event lifetime
}

type workQueue chan *Batch

// outputWorker instances pass events from the shared workQueue to the outputs.Client
// instances.
type outputWorker interface {
	Close() error
}

func newOutputController(
	log *logp.Logger,
	observer outputObserver,
	b queue.Queue,
) *outputController {
	c := &outputController{
		logger:   log,
		observer: observer,
		queue:    b,
	}

	ctx := &batchContext{}
	c.consumer = newEventConsumer(log, b, ctx)
	c.retryer = newRetryer(log, observer, nil, c.consumer)
	ctx.observer = observer
	ctx.retryer = c.retryer

	c.consumer.sigContinue()

	return c
}

func (c *outputController) Close() error {
	c.consumer.sigPause()

	if c.out != nil {
		for _, out := range c.out.outputs {
			out.Close()
		}
		close(c.out.workQueue)
	}

	c.consumer.close()
	c.retryer.close()

	return nil
}

func (c *outputController) Set(outGrp outputs.Group) {
	// create new outputGroup with shared work queue
	clients := outGrp.Clients
	queue := makeWorkQueue()
	worker := make([]outputWorker, len(clients))
	for i, client := range clients {
		worker[i] = makeClientWorker(c.observer, queue, client)
	}
	grp := &outputGroup{
		workQueue:  queue,
		outputs:    worker,
		timeToLive: outGrp.Retry + 1,
		batchSize:  outGrp.BatchSize,
	}

	// update consumer and retryer
	c.consumer.sigPause()
	if c.out != nil {
		for range c.out.outputs {
			c.retryer.sigOutputRemoved()
		}
	}
	c.retryer.updOutput(queue)
	for range clients {
		c.retryer.sigOutputAdded()
	}
	c.consumer.updOutput(grp)

	// close old group, so events are send to new workQueue via retryer
	if c.out != nil {
		for _, w := range c.out.outputs {
			w.Close()
		}
	}

	// restart consumer (potentially blocked by retryer)
	c.consumer.sigContinue()

	c.observer.updateOutputGroup()
}

func makeWorkQueue() workQueue {
	return workQueue(make(chan *Batch, 0))
}
