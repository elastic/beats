package api

import (
	"sync"
	"time"

	"github.com/joeshaw/multierror"

	"github.com/elastic/beats/libbeat/logp"
)

var debugK = "event_reporter"

// EventReporter is an object that will periodically send asyncronously events to the
// CM events endpoints.
type EventReporter struct {
	logger       *logp.Logger
	client       AuthClienter
	wg           sync.WaitGroup
	events       chan Event
	period       time.Duration
	maxBatchSize int
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
		events:       make(chan Event),
		maxBatchSize: maxBatchSize,
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
	close(e.events)
	e.wg.Wait()
}

func (e *EventReporter) worker() {
	defer e.wg.Done()

	ticker := time.NewTicker(e.period)
	defer ticker.Stop()

	var buffer []Event
	for {
		select {
		case event, ok := <-e.events:
			if !ok {
				e.reportEvents(buffer)
				return
			}
			buffer = append(buffer, event)
		case <-ticker.C:
			e.reportEvents(buffer)
			buffer = nil
		}
	}
}

func (e *EventReporter) reportEvents(events []Event) {
	if len(events) == 0 {
		return
	}
	e.logger.Debug("Reporting %d events to Kibana", len(events))

	// NOTE: Should we retry here? or do X attempts.
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

// AddEvents add an event to be send on the next tick.
func (e *EventReporter) AddEvents(events ...Event) {
	for _, event := range events {
		e.events <- event
	}
}

// AddEvent adds an event to be send on the next tick.
func (e *EventReporter) AddEvent(event Event) {
	e.events <- event
}
