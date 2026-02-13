// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package otelconsumer

// This file implements a congestion-controlled queue that bridges
// Beats' blocking-publish model with the OpenTelemetry Collector's
// immediate-reject queue model.
//
// The algorithm is inspired by TCP congestion control:
//
//   - Slow start: the window doubles per round-trip until the first
//     queue-full rejection (or until ssthresh is reached).
//   - Congestion avoidance (AIMD): after the first rejection the window
//     grows by additive/window on each success and is halved on each
//     rejection.
//
// Callers interact through [Queue.Publish], which returns a [Result]
// (promise). Publish blocks only when the congestion window is fully
// utilized, propagating back-pressure to the beat without buffering
// data beyond the window size.

import (
	"context"
	"errors"
	"math"
	"sync"
	"sync/atomic"

	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/pdata/plog"
)

// errClosed is returned from Publish after Close is called.
var errClosed = errors.New("adaptive queue closed")

// ---------------------------------------------------------------------------
// Result – the promise returned to beat callers
// ---------------------------------------------------------------------------

// Result is the promise-like handle returned by [Queue.Publish].
type Result struct {
	done chan struct{}
	err  error
}

// Done returns a channel that is closed when the send outcome is known.
func (r *Result) Done() <-chan struct{} { return r.done }

// Err blocks until the result is available and returns the outcome.
// A nil error means the data was accepted by the OTel queue.
func (r *Result) Err() error {
	<-r.done
	return r.err
}

// ---------------------------------------------------------------------------
// SendFunc
// ---------------------------------------------------------------------------

// SendFunc is the OTel collector queue's send operation.  It must block
// until the data is dispatched downstream *or* return
// [exporterhelper.ErrQueueIsFull] immediately when the queue has no room.
type SendFunc func(ctx context.Context, data plog.Logs) error

// ---------------------------------------------------------------------------
// Queue
// ---------------------------------------------------------------------------

// Queue implements adaptive congestion control between a beat producer
// and the OTel collector queue.
type Queue struct {
	send SendFunc

	mu     sync.Mutex
	cond   *sync.Cond
	closed bool

	// Congestion window (protected by mu).
	window    float64
	inflight  int
	ssthresh  float64 // slow-start threshold
	slowStart bool

	// AIMD parameters (immutable after construction).
	minWindow float64
	maxWindow float64
	additive  float64 // additive increase per window's worth of ACKs
	backoff   float64 // multiplicative decrease factor (e.g. 0.5)

	// Observability counters (atomic).
	totalSuccess atomic.Int64
	totalReject  atomic.Int64
}

// Option configures a [Queue].
type Option func(*Queue)

func WithInitialWindow(n float64) Option { return func(q *Queue) { q.window = n } }
func WithMinWindow(n float64) Option     { return func(q *Queue) { q.minWindow = n } }
func WithMaxWindow(n float64) Option     { return func(q *Queue) { q.maxWindow = n } }
func WithAdditive(n float64) Option      { return func(q *Queue) { q.additive = n } }
func WithBackoff(n float64) Option       { return func(q *Queue) { q.backoff = n } }

// New creates an adaptive queue wrapping the given OTel send function.
func New(send SendFunc, opts ...Option) *Queue {
	q := &Queue{
		send:      send,
		window:    4,
		minWindow: 1,
		maxWindow: 4096,
		ssthresh:  math.MaxFloat64, // no limit until first rejection
		slowStart: true,
		additive:  1,
		backoff:   0.5,
	}
	q.cond = sync.NewCond(&q.mu)
	for _, o := range opts {
		o(q)
	}
	return q
}

// Publish submits data to the OTel queue.  It blocks when the congestion
// window is fully utilized, providing back-pressure to the beat.
//
// The returned [Result] resolves once the OTel send completes (success)
// or is rejected (error, including queue-full).
func (q *Queue) Publish(ctx context.Context, data plog.Logs) *Result {
	r := &Result{done: make(chan struct{})}

	if err := q.acquire(ctx); err != nil {
		r.err = err
		close(r.done)
		return r
	}

	go q.doSend(ctx, data, r)
	return r
}

// PublishSync is a convenience wrapper that blocks until the send outcome
// is known and returns the error directly.
func (q *Queue) PublishSync(ctx context.Context, data plog.Logs) error {
	return q.Publish(ctx, data).Err()
}

// Close prevents new publishes and wakes all blocked callers.
// In-flight sends are not cancelled (use context for that).
func (q *Queue) Close() {
	q.mu.Lock()
	q.closed = true
	q.cond.Broadcast()
	q.mu.Unlock()
}

// ---------------------------------------------------------------------------
// Observability
// ---------------------------------------------------------------------------

// Snapshot holds a point-in-time view of the queue state.
type Snapshot struct {
	Window       float64
	Inflight     int
	SlowStart    bool
	SSThresh     float64
	TotalSuccess int64
	TotalReject  int64
}

// Stats returns an instantaneous snapshot for metrics / logging.
func (q *Queue) Stats() Snapshot {
	q.mu.Lock()
	defer q.mu.Unlock()
	return Snapshot{
		Window:       q.window,
		Inflight:     q.inflight,
		SlowStart:    q.slowStart,
		SSThresh:     q.ssthresh,
		TotalSuccess: q.totalSuccess.Load(),
		TotalReject:  q.totalReject.Load(),
	}
}

// ---------------------------------------------------------------------------
// Internal
// ---------------------------------------------------------------------------

// acquire blocks until a slot in the congestion window is available or
// ctx is cancelled.
func (q *Queue) acquire(ctx context.Context) error {
	// Wake the cond loop if the context is cancelled while we wait.
	stop := context.AfterFunc(ctx, func() { q.cond.Broadcast() })
	defer stop()

	q.mu.Lock()
	defer q.mu.Unlock()

	for {
		if q.closed {
			return errClosed
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if q.inflight < int(q.window) {
			q.inflight++
			return nil
		}
		q.cond.Wait()
	}
}

// release returns a window slot after a send completes.
func (q *Queue) release() {
	q.mu.Lock()
	q.inflight--
	q.cond.Broadcast()
	q.mu.Unlock()
}

// doSend performs the actual OTel send and adjusts the window based on
// the outcome.
func (q *Queue) doSend(ctx context.Context, data plog.Logs, r *Result) {
	defer func() {
		q.release()
		close(r.done)
	}()

	err := q.send(ctx, data)
	if err != nil {
		r.err = err
		if errors.Is(err, exporterhelper.ErrQueueIsFull) {
			q.onReject()
		}
		return
	}
	q.onSuccess()
}

// onSuccess applies the congestion-avoidance increase or slow-start
// doubling.
func (q *Queue) onSuccess() {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.totalSuccess.Add(1)

	if q.slowStart && q.window < q.ssthresh {
		// Exponential growth: +1 per ACK ≈ doubles per window.
		q.window++
	} else {
		// Congestion avoidance: +additive/window per ACK.
		q.slowStart = false
		q.window += q.additive / q.window
	}

	if q.window > q.maxWindow {
		q.window = q.maxWindow
	}
	q.cond.Broadcast() // window grew — wake blocked publishers
}

// onReject applies a multiplicative decrease and records ssthresh.
func (q *Queue) onReject() {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.totalReject.Add(1)

	// Record the congestion point.
	q.ssthresh = q.window * q.backoff
	if q.ssthresh < q.minWindow {
		q.ssthresh = q.minWindow
	}

	// Multiplicative decrease.
	q.window *= q.backoff
	if q.window < q.minWindow {
		q.window = q.minWindow
	}

	q.slowStart = false
}
