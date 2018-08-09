// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package core

import (
	"context"
	"fmt"

	"github.com/elastic/beats/libbeat/beat"
)

// Client implements the interface used by all the beatless function, we only implement a synchronous
// client. This interface superseed the core beat.Client interface inside beatless because our publish
// and publishAll methods can return an error.
type Client interface {
	// Publish accepts a unique events and will publish it to the pipeline.
	Publish(context.Context, beat.Event) error

	// PublishAll accepts a list of multiple events and will publish them to the pipeline.
	PublishAll(context.Context, []beat.Event) error

	// Close closes the current client, no events will be accepted.
	Close() error
}

// SyncClient wraps an existing beat.Client and provide a sync interface, when a new client is created.
// it need to be added to the AckMultiplexer.
type SyncClient struct {
	client      beat.Client
	eventsAcked chan int
}

// NewSyncClient returns a new sync client.
func NewSyncClient(client beat.Client) *SyncClient {
	return &SyncClient{client: client, eventsAcked: make(chan int)}
}

// Publish publishes one event to the pipeline and will wait the ACK before returning.
// The call is also unblocked via context cancellation or timeout.
func (s *SyncClient) Publish(ctx context.Context, event beat.Event) error {
	event.Private = SourceMetadata{Acker: s}
	s.client.Publish(event)

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-s.eventsAcked:
		return nil
	}
}

// PublishAll publish a slice of events to the pipeline and will wait the ACK before returning.
func (s *SyncClient) PublishAll(ctx context.Context, events []beat.Event) error {
	for _, e := range events {
		e.Private = SourceMetadata{Acker: s}
	}
	s.client.PublishAll(events)
	ackedCount := 0
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case count := <-s.eventsAcked:
			ackedCount += count
		}

		if ackedCount == len(events) {
			return nil
		}

		if ackedCount > len(events) {
			// Something is definitively wrong and should not happen.
			panic(fmt.Sprintf("Too many ACKS received, expected: %d, received: %d", len(events), ackedCount))
		}
	}
}

// Close closes the wrapped beat.Client.
func (s *SyncClient) Close() error {
	defer close(s.eventsAcked)
	return s.client.Close()
}

// AckEvents receives an array with all the event acked for this client.
func (s *SyncClient) AckEvents(events []interface{}) {
	select {
	case s.eventsAcked <- len(events):
		return
	}
}
