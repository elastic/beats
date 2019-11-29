// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fleet

import (
	"sync"
	"testing"
	"time"

	"github.com/elastic/beats/x-pack/agent/pkg/core/logger"
	"github.com/elastic/beats/x-pack/agent/pkg/fleetapi"
	"github.com/elastic/beats/x-pack/agent/pkg/reporter"
)

func TestReporting(t *testing.T) {
	// setup client
	threshold := 10
	c := newTestClient()
	r := newTestReporter(1*time.Second, threshold, c)

	// report events
	firstBatchSize := 5
	ee := getEvents(firstBatchSize)
	for _, e := range ee {
		r.Report(e)
	}

	// check after delay for output
	<-time.After(2 * time.Second)
	reportedEvents := c.events()
	if reportedCount := len(reportedEvents); reportedCount != firstBatchSize {
		t.Fatalf("expected %v events got %v", firstBatchSize, reportedCount)
	}

	// reset reported events
	c.reset()

	// report events > threshold
	secondBatchSize := threshold + 1
	ee = getEvents(secondBatchSize)
	for _, e := range ee {
		r.Report(e)
	}

	// check events are dropped
	<-time.After(2 * time.Second)
	reportedEvents = c.events()
	if reportedCount := len(reportedEvents); reportedCount != threshold {
		t.Fatalf("expected %v events got %v", secondBatchSize, reportedCount)
	}
}

func TestInfoDrop(t *testing.T) {
	// setup client
	threshold := 2
	c := newTestClient()
	r := newTestReporter(2*time.Second, threshold, c)

	// report 1 info and 1 error
	ee := []reporter.Event{testStateEvent{}, testErrorEvent{}, testErrorEvent{}}

	for _, e := range ee {
		r.Report(e)
	}

	// check after delay for output
	<-time.After(3 * time.Second)
	reportedEvents := c.events()
	if reportedCount := len(reportedEvents); reportedCount != 2 {
		t.Fatalf("expected %v events got %v", 2, reportedCount)
	}

	// check both are errors
	if reportedEvents[0].Type() != reportedEvents[1].Type() || reportedEvents[0].Type() != reporter.EventTypeError {
		t.Fatalf("expected ERROR events got [1]: '%v', [2]: '%v'", reportedEvents[0].Type(), reportedEvents[1].Type())
	}
}

type testClient struct {
	reportedEvents []fleetapi.SerializableEvent
	lock           sync.Mutex
}

func newTestClient() *testClient {
	return &testClient{
		reportedEvents: make([]fleetapi.SerializableEvent, 0),
	}
}

func (tc *testClient) Execute(r *fleetapi.CheckinRequest) (*fleetapi.CheckinResponse, error) {
	tc.lock.Lock()
	defer tc.lock.Unlock()

	tc.reportedEvents = append(tc.reportedEvents, r.Events...)
	return nil, nil
}

func (tc *testClient) reset() {
	tc.lock.Lock()
	defer tc.lock.Unlock()
	tc.reportedEvents = make([]fleetapi.SerializableEvent, 0)
}

func (tc *testClient) events() []fleetapi.SerializableEvent {
	tc.lock.Lock()
	defer tc.lock.Unlock()

	return tc.reportedEvents
}

func getEvents(count int) []reporter.Event {
	ee := make([]reporter.Event, 0, count)
	for i := 0; i < count; i++ {
		ee = append(ee, testStateEvent{})
	}

	return ee
}

func newTestReporter(frequency time.Duration, threshold int, client checkinExecutor) *Reporter {
	log, _ := logger.New()
	r := &Reporter{
		queue:       make([]reporter.Event, 0),
		ticker:      time.NewTicker(frequency),
		logger:      log,
		checkingCmd: client,
		threshold:   threshold,
		closeChan:   make(chan struct{}),
	}

	go r.reportLoop()
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
