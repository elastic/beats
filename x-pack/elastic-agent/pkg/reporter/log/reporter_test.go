// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package log

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/reporter"
)

type testCase struct {
	event         reporter.Event
	expectedInfo  string
	expectedError string
}

func TestReport(t *testing.T) {
	infoEvent := generateEvent(reporter.EventTypeState, reporter.EventSubTypeStarting)
	errorEvent := generateEvent(reporter.EventTypeError, reporter.EventSubTypeConfig)

	testCases := []testCase{
		{infoEvent, DefaultString(infoEvent), ""},
		{errorEvent, "", DefaultString(errorEvent)},
	}

	for _, tc := range testCases {
		log := newTestLogger()
		rep := NewReporter(log)

		rep.Report(context.Background(), tc.event)

		if got := log.info(); tc.expectedInfo != got {
			t.Errorf("[%s(%v)] expected info '%s' got '%s'", tc.event.Type(), tc.event.SubType(), tc.expectedInfo, got)
		}

		if got := log.error(); tc.expectedError != got {
			t.Errorf("[%s(%v)] expected error '%s' got '%s'", tc.event.Type(), tc.event.SubType(), tc.expectedError, got)
		}
	}
}

type testLogger struct {
	errorLog string
	infoLog  string
}

func newTestLogger() *testLogger {
	t := &testLogger{}
	return t
}

func (t *testLogger) Error(args ...interface{}) {
	t.errorLog = fmt.Sprint(args...)
}

func (t *testLogger) Info(args ...interface{}) {
	t.infoLog = fmt.Sprint(args...)
}

func (t *testLogger) error() string {
	return t.errorLog
}

func (t *testLogger) info() string {
	return t.infoLog
}

func generateEvent(eventype, subType string) testEvent {
	return testEvent{
		eventtype: eventype,
		subType:   subType,
		timestamp: time.Unix(0, 1),
		message:   "message",
	}
}

type testEvent struct {
	eventtype string
	subType   string
	timestamp time.Time
	message   string
}

func (t testEvent) Type() string                  { return t.eventtype }
func (t testEvent) SubType() string               { return t.subType }
func (t testEvent) Time() time.Time               { return t.timestamp }
func (t testEvent) Message() string               { return t.message }
func (testEvent) Payload() map[string]interface{} { return map[string]interface{}{} }

func DefaultString(event testEvent) string {
	timestamp := event.timestamp.Format(timeFormat)
	return fmt.Sprintf("%s: type: '%s': sub_type: '%s' message: message", timestamp, event.Type(), event.SubType())
}
