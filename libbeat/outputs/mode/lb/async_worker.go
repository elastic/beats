package lb

import (
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/op"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/outputs/mode"
)

type asyncWorkerFactory struct {
	clients                 []mode.AsyncProtocolClient
	waitRetry, maxWaitRetry time.Duration
}

// asyncWorker instances handle one load-balanced output per instance. Workers receive
// messages from context and return failed send attempts back to the context.
// Client connection state is fully handled by the worker.
type asyncWorker struct {
	id      int
	client  mode.AsyncProtocolClient
	backoff *common.Backoff
	ctx     context
}

func AsyncClients(
	clients []mode.AsyncProtocolClient,
	waitRetry, maxWaitRetry time.Duration,
) WorkerFactory {
	return &asyncWorkerFactory{
		clients:      clients,
		waitRetry:    waitRetry,
		maxWaitRetry: maxWaitRetry,
	}
}

func (s *asyncWorkerFactory) count() int { return len(s.clients) }

func (s *asyncWorkerFactory) mk(ctx context) ([]worker, error) {
	workers := make([]worker, len(s.clients))
	for i, client := range s.clients {
		workers[i] = newAsyncWorker(i, client, ctx, s.waitRetry, s.maxWaitRetry)
	}
	return workers, nil
}

func newAsyncWorker(
	id int,
	client mode.AsyncProtocolClient,
	ctx context,
	waitRetry, maxWaitRetry time.Duration,
) *asyncWorker {
	return &asyncWorker{
		id:      id,
		client:  client,
		backoff: common.NewBackoff(ctx.done, waitRetry, maxWaitRetry),
		ctx:     ctx,
	}
}

func (w *asyncWorker) run() {
	client := w.client

	debugf("load balancer: start client loop")
	defer debugf("load balancer: stop client loop")

	done := false
	for !done {
		if done = w.connect(); !done {
			done = w.sendLoop()

			debugf("close client (done=%v)", done)
			client.Close()
		}
	}
}

func (w *asyncWorker) connect() bool {
	for {
		err := w.client.Connect(w.ctx.timeout)
		if err == nil {
			w.backoff.Reset()
			return false
		}

		logp.Err("Connect failed with: %v", err)

		cont := w.backoff.Wait()
		if !cont {
			return true
		}
	}
}

func (w *asyncWorker) sendLoop() (done bool) {
	for {
		msg, ok := w.ctx.receive()
		if !ok {
			return true
		}

		msg.worker = w.id
		err := w.onMessage(msg)
		done = !w.backoff.WaitOnError(err)
		if done || err != nil {
			return done
		}
	}
}

func (w *asyncWorker) onMessage(msg eventsMessage) error {
	var err error
	if msg.datum.Event != nil {
		err = w.client.AsyncPublishEvent(w.handleResult(msg), msg.datum)
	} else {
		err = w.client.AsyncPublishEvents(w.handleResults(msg), msg.data)
	}

	if err != nil {
		if msg.attemptsLeft > 0 {
			msg.attemptsLeft--
		}

		// asynchronously retry to insert message (if attempts left), so worker can not
		// deadlock on retries channel if client puts multiple failed outstanding
		// events into the pipeline
		w.onFail(msg, err)
	}

	return err
}

func (w *asyncWorker) handleResult(msg eventsMessage) func(error) {
	return func(err error) {
		if err != nil {
			if msg.attemptsLeft > 0 {
				msg.attemptsLeft--
			}
			w.onFail(msg, err)
			return
		}

		op.SigCompleted(msg.signaler)
	}
}

func (w *asyncWorker) handleResults(msg eventsMessage) func([]outputs.Data, error) {
	total := len(msg.data)
	return func(data []outputs.Data, err error) {
		debugf("handleResults")

		if err != nil {
			debugf("handle publish error: %v", err)

			if msg.attemptsLeft > 0 {
				msg.attemptsLeft--
			}

			// reset attempt count if subset of messages has been processed
			if len(data) < total && msg.attemptsLeft >= 0 {
				msg.attemptsLeft = w.ctx.maxAttempts
			}

			if err != mode.ErrTempBulkFailure {
				// retry non-published subset of events in batch
				msg.data = data
				w.onFail(msg, err)
				return
			}

			if w.ctx.maxAttempts > 0 && msg.attemptsLeft == 0 {
				// no more attempts left => drop
				dropping(msg)
				return
			}

			// retry non-published subset of events in batch
			msg.data = data
			w.onFail(msg, err)
			return
		}

		// re-insert non-published events into pipeline
		if len(data) != 0 {
			go func() {
				debugf("add non-published events back into pipeline: %v", len(data))
				msg.data = data
				w.ctx.pushFailed(msg)
			}()
			return
		}

		// all events published -> signal success
		debugf("async bulk publish success")
		op.SigCompleted(msg.signaler)
	}
}

func (w *asyncWorker) onFail(msg eventsMessage, err error) {
	if !w.ctx.tryPushFailed(msg) {
		// break possible deadlock by spawning go-routine returning failed messages
		// into retries queue
		go func() {
			logp.Info("Error publishing events (retrying): %s", err)
			w.ctx.pushFailed(msg)
		}()
	}
}
