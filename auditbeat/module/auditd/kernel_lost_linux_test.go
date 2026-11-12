// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package auditd

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/pkg/logp/logptest"
	"github.com/elastic/go-libaudit/v2"
)

// Without a monitor client there is nothing to poll, so monitorKernelLost must
// report the error and return, not wait for a tick and use the missing client.
func TestMonitorKernelLost_CreateFails(t *testing.T) {
	createErr := errors.New("too many open files")
	newClient := func() (*libaudit.AuditClient, error) { return nil, createErr }

	ms := &MetricSet{log: logptest.NewTestingLogger(t, "")}
	r := &testPushReporter{done: make(chan struct{})}
	defer close(r.done)

	waitFor(t, runMonitorKernelLost(ms, r, newClient, time.Hour), "monitorKernelLost to return after failing to create its client")
	errs := r.errors()
	if assert.Len(t, errs, 1, "reported errors") {
		assert.ErrorIs(t, errs[0], createErr, "reported error")
	}
}

// The monitor client has its own socket, which must be closed when the
// monitor stops. The data client is closed by Run and must not be involved.
func TestMonitorKernelLost_ClosesMonitorClient(t *testing.T) {
	monitorMock := NewMock()
	newClient := func() (*libaudit.AuditClient, error) {
		return &libaudit.AuditClient{Netlink: monitorMock}, nil
	}
	dataMock := NewMock()

	ms := &MetricSet{
		log:    logptest.NewTestingLogger(t, ""),
		client: &libaudit.AuditClient{Netlink: dataMock},
	}
	r := &testPushReporter{done: make(chan struct{})}
	returned := runMonitorKernelLost(ms, r, newClient, time.Hour)

	close(r.done)
	waitFor(t, returned, "monitorKernelLost to return after the reporter was done")

	assertClosed(t, monitorMock, true, "monitor client")
	assertClosed(t, dataMock, false, "data client")
}

// A failed poll replaces the monitor client. The replacement must be the one
// that is polled and closed on return, and the one that failed must be closed
// once, when it is replaced.
func TestMonitorKernelLost_ReplacesClient(t *testing.T) {
	failing := NewMock().returnReceiveError(errors.New("no reply"))
	// The mock blocks on an empty queue, so queue enough replies for every poll
	// until the test ends.
	const polls = 1000
	replacement := NewMock()
	for range polls {
		replacement.returnACK().returnStatus()
	}
	clients := []*MockNetlinkSendReceiver{failing, replacement}
	created := make(chan struct{}, len(clients))
	newClient := func() (*libaudit.AuditClient, error) {
		defer func() { created <- struct{}{} }()
		m := clients[0]
		clients = clients[1:]
		return &libaudit.AuditClient{Netlink: m}, nil
	}

	ms := &MetricSet{log: logptest.NewTestingLogger(t, "")}
	r := &testPushReporter{done: make(chan struct{})}
	returned := runMonitorKernelLost(ms, r, newClient, 10*time.Millisecond)

	waitFor(t, created, "the monitor client to be created")
	waitFor(t, created, "the monitor client to be replaced")
	assert.Eventually(t, func() bool { return replacement.unread() < 2*polls }, time.Second, 5*time.Millisecond,
		"replacement client was not polled")

	assertClosed(t, failing, true, "failed client")
	assertClosed(t, replacement, false, "replacement client before the monitor stops")

	close(r.done)
	waitFor(t, returned, "monitorKernelLost to return after the reporter was done")
	assertClosed(t, replacement, true, "replacement client after the monitor stops")
	assert.Empty(t, r.errors(), "reported errors")
}

// If a failed poll cannot be followed by a new client, monitorKernelLost must
// report the error and return, and must not close the old client again.
func TestMonitorKernelLost_ReplaceFails(t *testing.T) {
	failing := NewMock().returnReceiveError(errors.New("no reply"))
	createErr := errors.New("too many open files")
	calls := 0
	newClient := func() (*libaudit.AuditClient, error) {
		calls++
		if calls > 1 {
			return nil, createErr
		}
		return &libaudit.AuditClient{Netlink: failing}, nil
	}

	ms := &MetricSet{log: logptest.NewTestingLogger(t, "")}
	r := &testPushReporter{done: make(chan struct{})}
	defer close(r.done)

	waitFor(t, runMonitorKernelLost(ms, r, newClient, 10*time.Millisecond), "monitorKernelLost to return after failing to replace its client")
	assertClosed(t, failing, true, "failed client")
	errs := r.errors()
	if assert.Len(t, errs, 1, "reported errors") {
		assert.ErrorIs(t, errs[0], createErr, "reported error")
	}
}

// testPushReporter is a mb.PushReporterV2 that records errors.
type testPushReporter struct {
	done chan struct{}

	mu   sync.Mutex
	errs []error
}

func (r *testPushReporter) Event(mb.Event) bool { return true }

func (r *testPushReporter) Error(err error) bool {
	r.mu.Lock()
	r.errs = append(r.errs, err)
	r.mu.Unlock()
	return true
}

func (r *testPushReporter) Done() <-chan struct{} { return r.done }

func (r *testPushReporter) errors() []error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]error(nil), r.errs...)
}

// waitFor fails the test if c is not closed within a second.
func waitFor(t *testing.T, c <-chan struct{}, what string) {
	t.Helper()
	select {
	case <-c:
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for %s", what)
	}
}

// runMonitorKernelLost runs monitorKernelLost in a goroutine and returns a
// channel that is closed when it returns.
func runMonitorKernelLost(ms *MetricSet, r *testPushReporter, newClient func() (*libaudit.AuditClient, error), interval time.Duration) <-chan struct{} {
	returned := make(chan struct{})
	go func() {
		defer close(returned)
		ms.monitorKernelLost(r, newClient, interval)
	}()
	return returned
}

func assertClosed(t *testing.T, m *MockNetlinkSendReceiver, want bool, name string) {
	t.Helper()
	var got bool
	select {
	case <-m.done:
		got = true
	default:
	}
	assert.Equal(t, want, got, "%s closed", name)
}
