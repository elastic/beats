package lb

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/stretchr/testify/assert"
)

func TestInfRetryNoDeadlock(t *testing.T) {
	N := 100       // Number of events to be send
	Fails := 1000  // Number of fails per event
	NumWorker := 2 // Number of concurrent workers pushing events into retry queue

	ctx := makeContext(1, -1, 0)

	// atomic success counter incremented whenever one event didn't fail
	// Test finishes if i==N
	i := int32(0)

	var closer sync.Once
	worker := func(wg *sync.WaitGroup) {
		defer wg.Done()

		// close queue once done, so other workers waiting for new messages will be
		// released
		defer closer.Do(func() {
			ctx.Close()
		})

		for int(atomic.LoadInt32(&i)) < N {
			msg, open := ctx.receive()
			if !open {
				break
			}

			fails := msg.datum.Event["fails"].(int)
			if fails < Fails {
				msg.datum.Event["fails"] = fails + 1
				ctx.pushFailed(msg)
				continue
			}

			atomic.AddInt32(&i, 1)
		}
	}

	var wg sync.WaitGroup
	wg.Add(NumWorker)
	for w := 0; w < NumWorker; w++ {
		go worker(&wg)
	}

	// push up to N events to workers. If workers deadlock, pushEvents will block.
	for i := 0; i < N; i++ {
		msg := eventsMessage{
			worker: -1,
			datum:  outputs.Data{Event: common.MapStr{"fails": int(0)}},
		}
		ok := ctx.pushEvents(msg, true)
		assert.True(t, ok)
	}

	// wait for all workers to terminate before test timeout
	wg.Wait()
}
