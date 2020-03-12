// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package api

import (
	"sync"
	"time"

	"github.com/joeshaw/multierror"

	"github.com/elastic/beats/v7/libbeat/logp"
)

const debugK = "event_reporter"

// EventReporter is an object that will periodically send asyncronously events to the
// CM events endpoints.
type EventReporter struct {
	logger       *logp.Logger
	client       AuthClienter
	period       time.Duration
	maxBatchSize int
	done         chan struct{}
	buffer       []Event
	mu           sync.Mutex
	wg           sync.WaitGroup
}

// NewEventReporter returns a new event reporter
func NewEventReporter(
	logger *logp.Logger,
	client AuthClienter,
	period time.Duration,
	maxBatchSize int,
) *EventReporter {
	log := logger.Named(debugK)
	return &EventReporter{
		logger:       log,
		client:       client,
		period:       period,
		maxBatchSize: maxBatchSize,
		done:         make(chan struct{}),
	}
}

// Start starts the event reported and wait for new events.
func (e *EventReporter) Start() {
	e.wg.Add(1)
	go e.worker()
	e.logger.Info("Starting event reporter service")
}

// Stop stops the reporting events to the endpoint.
func (e *EventReporter) Stop() {
	e.logger.Info("Stopping event reporter service")
	close(e.done)
	e.wg.Wait()
}

func (e *EventReporter) worker() {
	defer e.wg.Done()
	ticker := time.NewTicker(e.period)
	defer ticker.Stop()

	var done bool
	for !done {
		select {
		case <-e.done:
			done = true
		case <-ticker.C:
		}

		var buf []Event
		e.mu.Lock()
		buf, e.buffer = e.buffer, nil
		e.mu.Unlock()

		e.reportEvents(buf)
	}
}

func (e *EventReporter) reportEvents(events []Event) {
	if len(events) == 0 {
		return
	}
	e.logger.Debugf("Reporting %d events to Kibana", len(events))
	if err := e.sendBatchEvents(events); err != nil {
		e.logger.Errorf("could not send events, error: %+v", err)
	}
}

func (e *EventReporter) sendBatchEvents(events []Event) error {
	var errors multierror.Errors
	for pos := 0; pos < len(events); pos += e.maxBatchSize {
		j := pos + e.maxBatchSize
		if j > len(events) {
			j = len(events)
		}
		if err := e.sendEvents(events[pos:j]); err != nil {
			errors = append(errors, err)
		}
	}
	return errors.Err()
}

func (e *EventReporter) sendEvents(events []Event) error {
	requests := make([]EventRequest, len(events))
	for i, event := range events {
		requests[i] = EventRequest{
			Timestamp: time.Now(),
			EventType: event.EventType(),
			Event:     event,
		}
	}
	return e.client.SendEvents(requests)
}

// AddEvent adds an event to be send on the next tick.
func (e *EventReporter) AddEvent(event Event) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.buffer = append(e.buffer, event)
}
