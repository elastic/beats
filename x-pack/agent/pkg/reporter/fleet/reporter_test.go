// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fleet

import (
	"context"
	"testing"
	"time"

	"github.com/elastic/beats/x-pack/agent/pkg/core/logger"
	"github.com/elastic/beats/x-pack/agent/pkg/fleetapi"
	"github.com/elastic/beats/x-pack/agent/pkg/reporter"
)

func TestReporting(t *testing.T) {
	// setup client
	threshold := 10
	r := newTestReporter(1*time.Second, threshold)

	// report events
	firstBatchSize := 5
	ee := getEvents(firstBatchSize)
	for _, e := range ee {
		r.Report(context.Background(), e)
	}

	// check after delay for output
	reportedEvents, ack := r.Events()
	if reportedCount := len(reportedEvents); reportedCount != firstBatchSize {
		t.Fatalf("expected %v events got %v", firstBatchSize, reportedCount)
	}

	// reset reported events
	ack()

	// report events > threshold
	secondBatchSize := threshold + 1
	ee = getEvents(secondBatchSize)
	for _, e := range ee {
		r.Report(context.Background(), e)
	}

	// check events are dropped
	reportedEvents, _ = r.Events()
	if reportedCount := len(reportedEvents); reportedCount != threshold {
		t.Fatalf("expected %v events got %v", secondBatchSize, reportedCount)
	}
}

func TestInfoDrop(t *testing.T) {
	// setup client
	threshold := 2
	r := newTestReporter(2*time.Second, threshold)

	// report 1 info and 1 error
	ee := []reporter.Event{testStateEvent{}, testErrorEvent{}, testErrorEvent{}}

	for _, e := range ee {
		r.Report(context.Background(), e)
	}

	// check after delay for output
	reportedEvents, _ := r.Events()
	if reportedCount := len(reportedEvents); reportedCount != 2 {
		t.Fatalf("expected %v events got %v", 2, reportedCount)
	}

	// check both are errors
	if reportedEvents[0].Type() != reportedEvents[1].Type() || reportedEvents[0].Type() != reporter.EventTypeError {
		t.Fatalf("expected ERROR events got [1]: '%v', [2]: '%v'", reportedEvents[0].Type(), reportedEvents[1].Type())
	}
}

func TestOutOfOrderAck(t *testing.T) {
	// setup client
	threshold := 100
	r := newTestReporter(1*time.Second, threshold)

	// report events
	firstBatchSize := 5
	ee := getEvents(firstBatchSize)
	for _, e := range ee {
		r.Report(context.Background(), e)
	}

	// check after delay for output
	reportedEvents1, ack1 := r.Events()
	if reportedCount := len(reportedEvents1); reportedCount != firstBatchSize {
		t.Fatalf("expected %v events got %v", firstBatchSize, reportedCount)
	}

	// report events > threshold
	secondBatchSize := threshold + 1
	ee = getEvents(secondBatchSize)
	for _, e := range ee {
		r.Report(context.Background(), e)
	}

	// check all events are returned
	reportedEvents2, ack2 := r.Events()
	if reportedCount := len(reportedEvents2); reportedCount == firstBatchSize+secondBatchSize {
		t.Fatalf("expected %v events got %v", secondBatchSize, reportedCount)
	}

	// ack second batch
	ack2()

	reportedEvents, _ := r.Events()
	if reportedCount := len(reportedEvents); reportedCount != 0 {
		t.Fatalf("expected all events are removed after second batch ack, got %v events", reportedCount)
	}

	defer func() {
		r := recover()
		if r != nil {
			t.Fatalf("expected ack is ignored but it paniced: %v", r)
		}
	}()

	ack1()
	reportedEvents, _ = r.Events()
	if reportedCount := len(reportedEvents); reportedCount != 0 {
		t.Fatalf("expected all events are still removed after first batch ack, got %v events", reportedCount)
	}
}

func TestAfterDrop(t *testing.T) {
	// setup client
	threshold := 7
	r := newTestReporter(1*time.Second, threshold)

	// report events
	firstBatchSize := 5
	ee := getEvents(firstBatchSize)
	for _, e := range ee {
		r.Report(context.Background(), e)
	}

	// check after delay for output
	reportedEvents1, ack1 := r.Events()
	if reportedCount := len(reportedEvents1); reportedCount != firstBatchSize {
		t.Fatalf("expected %v events got %v", firstBatchSize, reportedCount)
	}

	// report events > threshold
	secondBatchSize := 5
	ee = getEvents(secondBatchSize)
	for _, e := range ee {
		r.Report(context.Background(), e)
	}

	// check all events are returned
	reportedEvents2, _ := r.Events()
	if reportedCount := len(reportedEvents2); reportedCount != threshold {
		t.Fatalf("expected %v events got %v", secondBatchSize, reportedCount)
	}

	// remove first batch from queue
	ack1()

	reportedEvents, _ := r.Events()
	if reportedCount := len(reportedEvents); reportedCount != secondBatchSize {
		t.Fatalf("expected all events from first batch are removed, got %v events", reportedCount)
	}

}

func getEvents(count int) []reporter.Event {
	ee := make([]reporter.Event, 0, count)
	for i := 0; i < count; i++ {
		ee = append(ee, testStateEvent{})
	}

	return ee
}

func newTestReporter(frequency time.Duration, threshold int) *Reporter {
	log, _ := logger.New()
	r := &Reporter{
		queue:     make([]fleetapi.SerializableEvent, 0),
		logger:    log,
		threshold: threshold,
	}

	return r
}

type testStateEvent struct{}

func (testStateEvent) Type() string                    { return reporter.EventTypeState }
func (testStateEvent) SubType() string                 { return reporter.EventSubTypeInProgress }
func (testStateEvent) Time() time.Time                 { return time.Unix(0, 1) }
func (testStateEvent) Message() string                 { return "hello" }
func (testStateEvent) Payload() map[string]interface{} { return map[string]interface{}{"key": 1} }
func (testStateEvent) Data() string                    { return "" }

type testErrorEvent struct{}

func (testErrorEvent) Type() string                    { return reporter.EventTypeError }
func (testErrorEvent) SubType() string                 { return "PATH" }
func (testErrorEvent) Time() time.Time                 { return time.Unix(0, 1) }
func (testErrorEvent) Message() string                 { return "hello" }
func (testErrorEvent) Payload() map[string]interface{} { return map[string]interface{}{"key": 1} }
func (testErrorEvent) Data() string                    { return "" }
