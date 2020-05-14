// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package reporter

import (
	"context"
	"errors"
	"testing"
)

var result Event

type testReporter struct{}

func (t *testReporter) Close() error { return nil }
func (t *testReporter) Report(_ context.Context, r Event) error {
	result = r
	return nil
}

type info struct{}

func (*info) AgentID() string { return "id" }

func TestTypes(t *testing.T) {
	rep := NewReporter(context.Background(), nil, &info{}, &testReporter{})
	// test starting
	rep.OnStarting(context.Background(), "a1")
	if r := result.Type(); r != EventTypeState {
		t.Errorf("OnStarting: expected record type '%v', got '%v'", EventTypeState, r)
	}

	if r := result.SubType(); r != EventSubTypeStarting {
		t.Errorf("OnStarting: expected event type '%v', got '%v'", EventSubTypeStarting, r)
	}

	// test in progress
	rep.OnRunning(context.Background(), "a2")
	if r := result.Type(); r != EventTypeState {
		t.Errorf("OnRunning: expected record type '%v', got '%v'", EventTypeState, r)
	}

	if r := result.SubType(); r != EventSubTypeInProgress {
		t.Errorf("OnRunning: expected event type '%v', got '%v'", EventSubTypeStarting, r)
	}

	// test stopping
	rep.OnStopping(context.Background(), "a3")
	if r := result.Type(); r != EventTypeState {
		t.Errorf("OnStopping: expected record type '%v', got '%v'", EventTypeState, r)
	}

	if r := result.SubType(); r != EventSubTypeStopping {
		t.Errorf("OnStopping: expected event type '%v', got '%v'", EventSubTypeStarting, r)
	}

	// test stopped
	rep.OnStopped(context.Background(), "a4")
	if r := result.Type(); r != EventTypeState {
		t.Errorf("OnStopped: expected record type '%v', got '%v'", EventTypeState, r)
	}

	if r := result.SubType(); r != EventSubTypeStopped {
		t.Errorf("OnStopped: expected event type '%v', got '%v'", EventSubTypeStarting, r)
	}

	// test failing
	err := errors.New("e1")
	rep.OnFailing(context.Background(), "a5", err)
	if r := result.Type(); r != EventTypeError {
		t.Errorf("OnFailing: expected record type '%v', got '%v'", EventTypeState, r)
	}

	if r := result.SubType(); r != EventSubTypeConfig {
		t.Errorf("OnFailing: expected event type '%v', got '%v'", EventSubTypeStarting, r)
	}

	// test fatal
	err = errors.New("e2")
	rep.OnFatal(context.Background(), "a6", err)
	if r := result.Type(); r != EventTypeError {
		t.Errorf("OnFatal: expected record type '%v', got '%v'", EventTypeState, r)
	}

	if r := result.SubType(); r != EventSubTypeConfig {
		t.Errorf("OnFatal: expected event type '%v', got '%v'", EventSubTypeStarting, r)
	}
}
