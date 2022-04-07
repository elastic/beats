// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package test

import (
	"sync"
	"testing"
	"time"

	"github.com/elastic/beats/v8/libbeat/common/bus"
)

// TestEventAccumulator defined a list of events for testing
type TestEventAccumulator struct {
	events []bus.Event
	lock   sync.Mutex
}

// Add expends events
func (tea *TestEventAccumulator) Add(e bus.Event) {
	tea.lock.Lock()
	defer tea.lock.Unlock()

	tea.events = append(tea.events, e)
}

// Len returns length of events
func (tea *TestEventAccumulator) Len() int {
	tea.lock.Lock()
	defer tea.lock.Unlock()

	return len(tea.events)
}

// Get copies the event and return it
func (tea *TestEventAccumulator) Get() []bus.Event {
	tea.lock.Lock()
	defer tea.lock.Unlock()

	res := make([]bus.Event, len(tea.events))
	copy(res, tea.events)
	return res
}

// WaitForNumEvents waits to get target length of events
func (tea *TestEventAccumulator) WaitForNumEvents(t *testing.T, targetLen int, timeout time.Duration) {
	start := time.Now()

	for time.Now().Sub(start) < timeout {
		if tea.Len() >= targetLen {
			return
		}
		time.Sleep(time.Millisecond)
	}

	t.Fatalf("Timed out waiting for num events to be %d", targetLen)
}
