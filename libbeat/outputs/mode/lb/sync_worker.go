package lb

import (
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/op"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs/mode"
)

type syncWorkerFactory struct {
	clients                 []mode.ProtocolClient
	waitRetry, maxWaitRetry time.Duration
}

// worker instances handle one load-balanced output per instance. Workers receive
// messages from context and return failed send attempts back to the context.
// Client connection state is fully handled by the worker.
type syncWorker struct {
	id      int
	client  mode.ProtocolClient
	backoff *common.Backoff
	ctx     context
}

func SyncClients(
	clients []mode.ProtocolClient,
	waitRetry, maxWaitRetry time.Duration,
) WorkerFactory {
	return &syncWorkerFactory{
		clients:      clients,
		waitRetry:    waitRetry,
		maxWaitRetry: maxWaitRetry,
	}
}

func (s *syncWorkerFactory) count() int { return len(s.clients) }

func (s *syncWorkerFactory) mk(ctx context) ([]worker, error) {
	workers := make([]worker, len(s.clients))
	for i, client := range s.clients {
		workers[i] = newSyncWorker(i, client, ctx, s.waitRetry, s.maxWaitRetry)
	}
	return workers, nil
}

func newSyncWorker(
	id int,
	client mode.ProtocolClient,
	ctx context,
	waitRetry, maxWaitRetry time.Duration,
) *syncWorker {
	return &syncWorker{
		id:      id,
		client:  client,
		backoff: common.NewBackoff(ctx.done, waitRetry, maxWaitRetry),
		ctx:     ctx,
	}
}

func (w *syncWorker) run() {
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

func (w *syncWorker) connect() bool {
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

func (w *syncWorker) sendLoop() (done bool) {
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

func (w *syncWorker) onMessage(msg eventsMessage) error {
	client := w.client

	if msg.datum.Event != nil {
		err := client.PublishEvent(msg.datum)
		if err != nil {
			if msg.attemptsLeft > 0 {
				msg.attemptsLeft--
			}
			w.onFail(msg, err)
			return err
		}
	} else {
		events := msg.data
		total := len(events)

		for len(events) > 0 {
			var err error

			events, err = client.PublishEvents(events)
			if err != nil {
				if msg.attemptsLeft > 0 {
					msg.attemptsLeft--
				}

				// reset attempt count if subset of messages has been processed
				if len(events) < total && msg.attemptsLeft >= 0 {
					debugf("reset fails")
					msg.attemptsLeft = w.ctx.maxAttempts
				}

				if err != mode.ErrTempBulkFailure {
					// retry non-published subset of events in batch
					msg.data = events
					w.onFail(msg, err)
					return err
				}

				if w.ctx.maxAttempts > 0 && msg.attemptsLeft == 0 {
					// no more attempts left => drop
					dropping(msg)
					return err
				}

				// reset total count for temporary failure loop
				total = len(events)
			}
		}
	}

	op.SigCompleted(msg.signaler)
	return nil
}

func (w *syncWorker) onFail(msg eventsMessage, err error) {
	logp.Info("Error publishing events (retrying): %s", err)
	w.ctx.pushFailed(msg)
}
