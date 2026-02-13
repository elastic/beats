// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package otelconsumer

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/pdata/plog"
)

// ---------------------------------------------------------------------------
// Simulated OTel queue for testing
// ---------------------------------------------------------------------------

type fakeOTelQueue struct {
	mu       sync.Mutex
	capacity int
	used     int
	delay    time.Duration // simulated downstream latency
}

func newFakeOTelQueue(capacity int, delay time.Duration) *fakeOTelQueue {
	return &fakeOTelQueue{capacity: capacity, delay: delay}
}

func (q *fakeOTelQueue) Send(ctx context.Context, data plog.Logs) error {
	q.mu.Lock()
	if q.used >= q.capacity {
		q.mu.Unlock()
		return exporterhelper.ErrQueueIsFull
	}
	q.used++
	q.mu.Unlock()

	// Simulate blocking until "sent".
	select {
	case <-time.After(q.delay):
	case <-ctx.Done():
		q.mu.Lock()
		q.used--
		q.mu.Unlock()
		return ctx.Err()
	}

	q.mu.Lock()
	q.used--
	q.mu.Unlock()
	return nil
}

func (q *fakeOTelQueue) SetDelay(d time.Duration) {
	q.mu.Lock()
	q.delay = d
	q.mu.Unlock()
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestBasicPublish(t *testing.T) {
	otel := newFakeOTelQueue(100, 1*time.Millisecond)
	q := New(otel.Send)
	defer q.Close()

	ctx := context.Background()
	r := q.Publish(ctx, plog.NewLogs())
	if err := r.Err(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPublishSync(t *testing.T) {
	otel := newFakeOTelQueue(100, 1*time.Millisecond)
	q := New(otel.Send)
	defer q.Close()

	if err := q.PublishSync(context.Background(), plog.NewLogs()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSlowStartGrowsWindow(t *testing.T) {
	otel := newFakeOTelQueue(1000, 1*time.Millisecond)
	q := New(otel.Send, WithInitialWindow(2))
	defer q.Close()

	ctx := context.Background()
	// Send enough events to trigger slow-start growth.
	for i := 0; i < 20; i++ {
		if err := q.PublishSync(ctx, plog.NewLogs()); err != nil {
			t.Fatalf("publish %d: %v", i, err)
		}
	}

	s := q.Stats()
	if s.Window <= 2 {
		t.Fatalf("expected window to grow from 2 during slow start, got %v", s.Window)
	}
	if !s.SlowStart {
		t.Log("exited slow start (hit ssthresh) — also valid")
	}
	t.Logf("window after 20 events: %.1f", s.Window)
}

func TestQueueFullTriggersBackoff(t *testing.T) {
	// Tiny OTel queue that fills up fast.
	otel := newFakeOTelQueue(2, 50*time.Millisecond)
	q := New(otel.Send, WithInitialWindow(8), WithMaxWindow(64))
	defer q.Close()

	ctx := context.Background()
	var wg sync.WaitGroup
	var rejects atomic.Int64

	// Fire a burst — some will succeed, some will get queue-full.
	for i := 0; i < 30; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			err := q.PublishSync(ctx, plog.NewLogs())
			if errors.Is(err, exporterhelper.ErrQueueIsFull) {
				rejects.Add(1)
			}
		}(i)
	}
	wg.Wait()

	s := q.Stats()
	if s.TotalReject == 0 {
		t.Fatal("expected at least one queue-full rejection")
	}
	// Window should have shrunk from 8.
	if s.Window >= 8 {
		t.Fatalf("expected window to decrease after rejections, got %.1f", s.Window)
	}
	t.Logf("final window: %.1f, rejects: %d, successes: %d",
		s.Window, s.TotalReject, s.TotalSuccess)
}

func TestContextCancellation(t *testing.T) {
	otel := newFakeOTelQueue(1, 10*time.Second) // very slow
	q := New(otel.Send, WithInitialWindow(1))
	defer q.Close()

	ctx := context.Background()

	// Fill the single slot.
	_ = q.Publish(ctx, plog.NewLogs())

	// This publish should block on acquire; cancel quickly.
	cancelCtx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
	defer cancel()

	r := q.Publish(cancelCtx, plog.NewLogs())
	err := r.Err()
	if err == nil {
		t.Fatal("expected context error")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected DeadlineExceeded, got %v", err)
	}
}

func TestCloseWakesWaiters(t *testing.T) {
	otel := newFakeOTelQueue(1, 10*time.Second)
	q := New(otel.Send, WithInitialWindow(1))

	ctx := context.Background()
	_ = q.Publish(ctx, plog.NewLogs())

	done := make(chan error, 1)
	go func() {
		done <- q.PublishSync(ctx, plog.NewLogs())
	}()

	time.Sleep(20 * time.Millisecond)
	q.Close()

	select {
	case err := <-done:
		if !errors.Is(err, errClosed) {
			t.Fatalf("expected errClosed, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("waiter was not woken by Close")
	}
}

func TestInflightNeverExceedsWindow(t *testing.T) {
	otel := newFakeOTelQueue(1000, 5*time.Millisecond)
	q := New(otel.Send, WithInitialWindow(8), WithMaxWindow(32))
	defer q.Close()

	ctx := context.Background()
	var maxInflight atomic.Int64
	var wg sync.WaitGroup

	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			r := q.Publish(ctx, plog.NewLogs())
			// Sample inflight at the moment of publish.
			s := q.Stats()
			for {
				cur := maxInflight.Load()
				v := int64(s.Inflight)
				if v <= cur {
					break
				}
				if maxInflight.CompareAndSwap(cur, v) {
					break
				}
			}
			_ = r.Err()
		}(i)
	}
	wg.Wait()

	s := q.Stats()
	if maxInflight.Load() > int64(s.Window)+1 { // +1 for race tolerance
		t.Fatalf("inflight (%d) exceeded window (%.0f)", maxInflight.Load(), s.Window)
	}
	t.Logf("max observed inflight: %d, final window: %.1f",
		maxInflight.Load(), s.Window)
}

func TestAIMDRecovery(t *testing.T) {
	// Start with a tiny OTel queue that causes rejections, then expand it.
	otel := newFakeOTelQueue(2, 5*time.Millisecond)
	q := New(otel.Send, WithInitialWindow(4), WithMaxWindow(64))
	defer q.Close()

	ctx := context.Background()

	// Phase 1: trigger rejections and shrink the window.
	for i := 0; i < 40; i++ {
		_ = q.PublishSync(ctx, plog.NewLogs())
	}
	s1 := q.Stats()
	t.Logf("after phase 1 — window: %.1f, rejects: %d", s1.Window, s1.TotalReject)

	// Phase 2: expand OTel queue — window should recover.
	otel.mu.Lock()
	otel.capacity = 100
	otel.mu.Unlock()

	for i := 0; i < 200; i++ {
		_ = q.PublishSync(ctx, plog.NewLogs())
	}
	s2 := q.Stats()
	t.Logf("after phase 2 — window: %.1f, rejects: %d", s2.Window, s2.TotalReject)

	if s2.Window <= s1.Window {
		t.Fatalf("window did not recover: was %.1f, now %.1f", s1.Window, s2.Window)
	}
}

// ---------------------------------------------------------------------------
// Benchmark
// ---------------------------------------------------------------------------

func BenchmarkPublishSync(b *testing.B) {
	otel := newFakeOTelQueue(10000, 0) // instant send
	q := New(otel.Send, WithInitialWindow(64), WithMaxWindow(4096))
	defer q.Close()

	ctx := context.Background()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = q.PublishSync(ctx, plog.NewLogs())
		}
	})
}

// ---------------------------------------------------------------------------
// Example
// ---------------------------------------------------------------------------

func ExampleQueue() {
	// Simulated OTel queue with capacity 10 and 1ms send latency.
	otel := newFakeOTelQueue(10, 1*time.Millisecond)

	q := New(otel.Send,
		WithInitialWindow(4),
		WithMaxWindow(64),
		WithBackoff(0.5),
	)
	defer q.Close()

	ctx := context.Background()

	// Fire-and-forget style (beat model).
	results := make([]*Result, 20)
	for i := range results {
		results[i] = q.Publish(ctx, plog.NewLogs())
	}

	// Collect outcomes.
	for i, r := range results {
		if err := r.Err(); err != nil {
			fmt.Printf("event-%d: %v\n", i, err)
		}
	}

	s := q.Stats()
	fmt.Printf("window=%.0f successes=%d rejects=%d\n",
		math.Round(s.Window), s.TotalSuccess, s.TotalReject)
}
