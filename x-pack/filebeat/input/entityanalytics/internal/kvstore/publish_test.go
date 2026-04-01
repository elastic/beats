// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package kvstore

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/entcollect"
)

type fakeClient struct {
	mu     sync.Mutex
	events []beat.Event
}

func (c *fakeClient) Publish(event beat.Event) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.events = append(c.events, event)
}

func (c *fakeClient) PublishAll(events []beat.Event) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.events = append(c.events, events...)
}

func (c *fakeClient) Close() error { return nil }

func (c *fakeClient) Events() []beat.Event {
	c.mu.Lock()
	defer c.mu.Unlock()
	dst := make([]beat.Event, len(c.events))
	copy(dst, c.events)
	return dst
}

func TestNewPublisher_FieldMapping(t *testing.T) {
	t.Parallel()

	client := &fakeClient{}
	tracker := NewTxTracker(context.Background())
	pub := NewPublisher(client, "input-123", tracker)

	ts := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	doc := entcollect.Document{
		ID:        "user-abc",
		Kind:      entcollect.KindUser,
		Action:    entcollect.ActionDiscovered,
		Timestamp: ts,
		Fields: map[string]any{
			"user.name": "alice",
			"user.id":   "abc",
		},
	}

	if err := pub(context.Background(), doc); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	events := client.Events()
	if got := len(events); got != 1 {
		t.Fatalf("len(events) = %d; want 1", got)
	}

	e := events[0]
	if e.Timestamp != ts {
		t.Errorf("Timestamp = %v; want %v", e.Timestamp, ts)
	}

	checks := []struct {
		field string
		want  any
	}{
		{"event.action", "user-discovered"},
		{"event.kind", "asset"},
		{"labels.identity_source", "input-123"},
		{"user.name", "alice"},
		{"user.id", "abc"},
	}
	for _, c := range checks {
		got, _ := e.Fields.GetValue(c.field)
		if got != c.want {
			t.Errorf("Fields[%q] = %v; want %v", c.field, got, c.want)
		}
	}
}

func TestNewPublisher_TrackerAddBeforePublish(t *testing.T) {
	t.Parallel()

	tracker := NewTxTracker(context.Background())
	client := &fakeClient{}
	pub := NewPublisher(client, "input-1", tracker)

	const n = 5
	for i := range n {
		doc := entcollect.Document{
			ID:        "id",
			Kind:      entcollect.KindDevice,
			Action:    entcollect.ActionModified,
			Timestamp: time.Now(),
			Fields:    map[string]any{"i": i},
		}
		if err := pub(context.Background(), doc); err != nil {
			t.Fatalf("Publish[%d]: %v", i, err)
		}
	}

	if got := tracker.pending.Load(); got != n {
		t.Fatalf("tracker.pending = %d; want %d (Add must be called once per Publish)", got, n)
	}

	for range n {
		tracker.Ack()
	}
	tracker.Wait()
	if got := tracker.pending.Load(); got != 0 {
		t.Errorf("tracker.pending after Wait = %d; want 0", got)
	}
}

func TestNewPublisher_ContextCancelled(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	client := &fakeClient{}
	tracker := NewTxTracker(ctx)
	pub := NewPublisher(client, "input-1", tracker)

	err := pub(ctx, entcollect.Document{
		ID:        "id",
		Kind:      entcollect.KindUser,
		Action:    entcollect.ActionDeleted,
		Timestamp: time.Now(),
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Publish(cancelled) = %v; want context.Canceled", err)
	}
}

func TestNewPublisher_TrackerWaitBlocks(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tracker := NewTxTracker(ctx)
	client := &fakeClient{}
	pub := NewPublisher(client, "input-1", tracker)

	doc := entcollect.Document{
		ID:        "id",
		Kind:      entcollect.KindUser,
		Action:    entcollect.ActionDiscovered,
		Timestamp: time.Now(),
	}
	if err := pub(ctx, doc); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	done := make(chan struct{})
	go func() {
		tracker.Wait()
		close(done)
	}()

	select {
	case <-done:
		t.Fatal("Wait returned before ACK")
	case <-time.After(50 * time.Millisecond):
	}

	tracker.Ack()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Wait did not return after ACK")
	}
}
