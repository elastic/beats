// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/config"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/ecs"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

// gatedLivePublisher records live documents in publication order. Rows from
// index blockFrom on wait until release is closed, like a full output.
type gatedLivePublisher struct {
	blockFrom int
	release   chan struct{}
	blocked   chan struct{} // receives when a row starts waiting for release
	closed    atomic.Bool

	mu   sync.Mutex
	docs []string
}

var _ liveQueryPublisher = (*gatedLivePublisher)(nil)

func newGatedLivePublisher(blockFrom int) *gatedLivePublisher {
	return &gatedLivePublisher{blockFrom: blockFrom, release: make(chan struct{}), blocked: make(chan struct{}, 16)}
}

func (p *gatedLivePublisher) PublishEach(_, idValue, _, _, _, _, _, _ string, _ map[string]any, hits []map[string]any, _ ecs.Mapping, _ any, stop func() bool) int {
	for i := range hits {
		if stop() {
			return i
		}
		if i >= p.blockFrom {
			select {
			case p.blocked <- struct{}{}:
			default:
			}
			<-p.release
		}
		p.record(fmt.Sprintf("%s row %d", idValue, i+1))
	}
	return len(hits)
}

func (p *gatedLivePublisher) PublishQueryProfile(_, _, actionID, _, _ string, _ map[string]any, _ any) {
	p.record(actionID + " profile")
}

func (p *gatedLivePublisher) PublishActionResult(req, _ map[string]any) {
	p.record(fmt.Sprint(req["id"], " response"))
}

func (p *gatedLivePublisher) Configure([]config.InputConfig) error { return nil }

func (p *gatedLivePublisher) Close() { p.closed.Store(true) }

func (p *gatedLivePublisher) record(doc string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.docs = append(p.docs, doc)
}

func (p *gatedLivePublisher) snapshot() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]string(nil), p.docs...)
}

func newLiveTestHandler(t *testing.T, d *liveResultDispatcher, pub actionQueryPublisher, rows int) *actionHandler {
	hits := make([]map[string]any, rows)
	for i := range hits {
		hits[i] = map[string]any{"row": i}
	}
	return &actionHandler{
		log:       logptest.NewTestingLogger(t, ""),
		inputType: osqueryInputType,
		queryExec: &mockExecutor{result: hits},
		publisher: pub,
		live:      d,
	}
}

func liveRequest(id string) map[string]any {
	return map[string]any{"id": id, "data": map[string]any{"query": "select * from uptime"}}
}

func TestExecuteReportsPartialCountOnPublishTimeout(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		pub := newGatedLivePublisher(2)
		d := newLiveResultDispatcher(monitoring.NewRegistry(), logptest.NewTestingLogger(t, ""), 100*time.Millisecond)
		h := newLiveTestHandler(t, d, pub, 5)

		start := time.Now()
		res, err := h.Execute(t.Context(), liveRequest("a1"))
		elapsed := time.Since(start)

		require.NoError(t, err, "Execute reports errors in the result")
		assert.Equal(t, 2, res["count"], "report the rows handed to the output before the timeout")
		assert.Contains(t, res["error"], "timed out publishing", "report the timeout as the action error")
		assert.LessOrEqual(t, elapsed, 300*time.Millisecond, "Execute must return soon after the publish timeout")
		assert.Equal(t, uint64(1), d.metrics.timeout.Get(), "count the timed-out action")
		assert.Equal(t, uint64(3), d.metrics.timeoutRows.Get(), "count the rows not handed to the output")

		close(pub.release)
		d.stop()
	})
}

func TestExecuteHealthyPublishesAll(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		pub := newGatedLivePublisher(1 << 30)
		d := newLiveResultDispatcher(monitoring.NewRegistry(), logptest.NewTestingLogger(t, ""), 100*time.Millisecond)
		h := newLiveTestHandler(t, d, pub, 5)

		res, err := h.Execute(t.Context(), liveRequest("a1"))

		require.NoError(t, err, "Execute reports errors in the result")
		assert.Equal(t, 5, res["count"], "a healthy output must publish every row")
		assert.NotContains(t, res, "error", "a healthy output must not report an error")
		assert.Equal(t, []string{"a1 row 1", "a1 row 2", "a1 row 3", "a1 row 4", "a1 row 5"}, pub.snapshot(),
			"publish every row in order")
		assert.Zero(t, d.metrics.timeoutRows.Get(), "a healthy output drops no rows")

		d.stop()
	})
}

func TestActionResponsePublishedAfterTimedOutRows(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		log := logptest.NewTestingLogger(t, "")
		pub := newGatedLivePublisher(2)
		d := newLiveResultDispatcher(monitoring.NewRegistry(), log, 100*time.Millisecond)
		rah := newResetableActionHandler(d, pub, log)
		rah.Attach(newLiveTestHandler(t, d, pub, 5))

		res, err := rah.Execute(t.Context(), liveRequest("a1"))
		require.NoError(t, err, "Execute reports errors in the result")
		assert.Equal(t, 2, res["count"], "the action response must keep the partial count")
		assert.Contains(t, res["error"], "timed out publishing", "the action response must report the timeout")

		synctest.Wait()
		assert.Equal(t, []string{"a1 row 1", "a1 row 2"}, pub.snapshot(), "the response waits behind the stuck row")

		close(pub.release)
		synctest.Wait()
		assert.Equal(t, []string{"a1 row 1", "a1 row 2", "a1 row 3", "a1 response"}, pub.snapshot(),
			"publish the response after the rows; the stuck row arrives late and later rows are skipped")
		assert.Zero(t, d.metrics.shutdown.Get(), "the timeout must not drop the response")

		d.stop()
	})
}

func TestSecondLiveActionTimesOutBehindStuckRow(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const timeout = 100 * time.Millisecond
		log := logptest.NewTestingLogger(t, "")
		pub := newGatedLivePublisher(2)
		d := newLiveResultDispatcher(monitoring.NewRegistry(), log, timeout)
		rah := newResetableActionHandler(d, pub, log)
		rah.Attach(newLiveTestHandler(t, d, pub, 5))

		_, err := rah.Execute(t.Context(), liveRequest("a1"))
		require.NoError(t, err, "Execute reports errors in the result")

		start := time.Now()
		res, err := rah.Execute(t.Context(), liveRequest("a2"))
		elapsed := time.Since(start)

		require.NoError(t, err, "Execute reports errors in the result")
		assert.Equal(t, 0, res["count"], "no row of the second action can pass the stuck row")
		assert.Contains(t, res["error"], "timed out publishing", "report the timeout as the action error")
		assert.LessOrEqual(t, elapsed, timeout+100*time.Millisecond, "the second action must wait at most its timeout")
		assert.Equal(t, uint64(2), d.metrics.timeout.Get(), "count both timed-out actions")
		assert.Equal(t, uint64(3+5), d.metrics.timeoutRows.Get(), "count the rows of both actions not handed to the output")

		close(pub.release)
		synctest.Wait()
		assert.Equal(t, []string{"a1 row 1", "a1 row 2", "a1 row 3", "a1 response", "a2 response"}, pub.snapshot(),
			"publish both responses in order and none of the second action's rows")

		d.stop()
	})
}

func TestExecuteReturnsOnShutdown(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		pub := newGatedLivePublisher(0)
		d := newLiveResultDispatcher(monitoring.NewRegistry(), logptest.NewTestingLogger(t, ""), time.Minute)
		h := newLiveTestHandler(t, d, pub, 5)

		done := make(chan map[string]any)
		go func() {
			res, _ := h.Execute(t.Context(), liveRequest("a1"))
			done <- res
		}()
		synctest.Wait()

		start := time.Now()
		d.stop()
		res := <-done

		assert.LessOrEqual(t, time.Since(start), 100*time.Millisecond, "Execute must return promptly on shutdown")
		assert.Equal(t, errLiveShutdown.Error(), res["error"], "report the shutdown as the action error")
		assert.Equal(t, uint64(1), d.metrics.shutdown.Get(), "count the abandoned action")

		close(pub.release)
	})
}

func TestLiveResponseAfterStopIsDropped(t *testing.T) {
	pub := newGatedLivePublisher(0)
	d := newLiveResultDispatcher(monitoring.NewRegistry(), logptest.NewTestingLogger(t, ""), time.Minute)
	d.stop()

	d.enqueueResponse(pub, liveRequest("a1"), map[string]any{})

	assert.Equal(t, uint64(1), d.metrics.shutdown.Get(), "count a response admitted after stop as a shutdown drop")
	assert.Empty(t, pub.snapshot(), "publish nothing after stop")
}

// TestClearWaitsAtMostLiveTimeout uses real time: synctest does not advance
// its clock while a goroutine waits for a sync.Mutex.
func TestClearWaitsAtMostLiveTimeout(t *testing.T) {
	log := logptest.NewTestingLogger(t, "")
	pub := newGatedLivePublisher(0)
	d := newLiveResultDispatcher(monitoring.NewRegistry(), log, 200*time.Millisecond)
	rah := newResetableActionHandler(d, pub, log)
	rah.Attach(newLiveTestHandler(t, d, pub, 5))

	executed := make(chan struct{})
	go func() {
		defer close(executed)
		_, _ = rah.Execute(context.Background(), liveRequest("a1"))
	}()
	<-pub.blocked

	start := time.Now()
	rah.Clear()
	assert.Less(t, time.Since(start), 400*time.Millisecond, "Clear must wait at most about the live timeout")

	<-executed
	close(pub.release)
	d.stop()
}

func TestLiveClientsCloseDoesNotWaitForStuckPublish(t *testing.T) {
	var made []*gatedLivePublisher
	c := &liveClients{newPublisher: func() liveQueryPublisher {
		p := newGatedLivePublisher(0)
		made = append(made, p)
		return p
	}}
	inputs := []config.InputConfig{{}}
	require.NoError(t, c.Configure(inputs), "configure the first clients")

	published := make(chan struct{})
	go func() {
		defer close(published)
		c.PublishEach("", "a1", "action_id", "", "", "", "", "", nil, []map[string]any{{}}, nil, nil, func() bool { return false })
	}()
	<-made[0].blocked

	require.NoError(t, c.Configure(inputs), "a policy change must not wait for a stuck publish")
	c.Close()
	assert.False(t, made[0].closed.Load(), "a client with a publish in flight must not be closed")
	assert.True(t, made[1].closed.Load(), "an idle client is closed at once")

	close(made[0].release)
	<-published
	assert.True(t, made[0].closed.Load(), "a retired client is closed when its publish returns")
}
