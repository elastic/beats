// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build linux

package ebpf

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

const allEvents = EventMask(math.MaxUint64)

func TestWatcherStartStop(t *testing.T) {
	w, err := GetWatcher()
	if err != nil {
		t.Skipf("skipping ebpf watcher test: %v", err)
	}
	assert.Equal(t, gWatcher.status, stopped)
	assert.Equal(t, 0, gWatcher.nclients())

	_ = w.Subscribe("test-1", allEvents)
	assert.Equal(t, gWatcher.status, started)
	assert.Equal(t, 1, gWatcher.nclients())

	_ = w.Subscribe("test-2", allEvents)
	assert.Equal(t, 2, gWatcher.nclients())

	w.Unsubscribe("test-2")
	assert.Equal(t, 1, gWatcher.nclients())

	w.Unsubscribe("dummy")
	assert.Equal(t, 1, gWatcher.nclients())

	assert.Equal(t, gWatcher.status, started)
	w.Unsubscribe("test-1")
	assert.Equal(t, 0, gWatcher.nclients())
	assert.Equal(t, gWatcher.status, stopped)

	_ = w.Subscribe("new", allEvents)
	assert.Equal(t, 1, gWatcher.nclients())
	assert.Equal(t, gWatcher.status, started)
	w.Unsubscribe("new")
}
