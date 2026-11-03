// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

func TestWatcherRunClose(t *testing.T) {
	w := NewWatcher(logptest.NewTestingLogger(t, t.Name()))

	done := make(chan struct{})
	go func() {
		defer close(done)
		w.Run()
	}()

	// Run takes the lock briefly to set cancel, so this also shows that
	// it does not hold the lock while polling.
	assert.Eventually(t, func() bool {
		w.mx.Lock()
		defer w.mx.Unlock()
		return w.cancel != nil
	}, 5*time.Second, time.Millisecond, "watcher did not start")

	closed := make(chan struct{})
	go func() {
		defer close(closed)
		w.Close()
	}()

	select {
	case <-closed:
	case <-time.After(5 * time.Second):
		t.Fatal("Close blocked while the watcher was running")
	}
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not return after Close")
	}
}

func TestWatcherRunTwice(t *testing.T) {
	w := NewWatcher(logptest.NewTestingLogger(t, t.Name()))
	running := make(chan struct{})
	go func() {
		defer close(running)
		w.Run()
	}()
	// Wait for Run to return so it does not log after the test completes.
	t.Cleanup(func() {
		w.Close()
		<-running
	})

	assert.Eventually(t, func() bool {
		w.mx.Lock()
		defer w.mx.Unlock()
		return w.cancel != nil
	}, 5*time.Second, time.Millisecond, "watcher did not start")

	done := make(chan struct{})
	go func() {
		defer close(done)
		w.Run()
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("second Run did not return while the watcher was running")
	}
}
