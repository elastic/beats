// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build linux && (amd64 || arm64) && cgo

package kerneltracingprovider

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/beat"
)

func TestWaitLimit(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name string
		ev   *beat.Event
		want time.Duration
	}{
		{name: "nil_event", ev: nil, want: maxWaitLimit},
		{name: "no_timestamp", ev: &beat.Event{}, want: maxWaitLimit},
		{name: "new_event", ev: &beat.Event{Timestamp: now}, want: maxWaitLimit},
		{name: "future_event", ev: &beat.Event{Timestamp: now.Add(time.Hour)}, want: maxWaitLimit},
		{name: "partly_aged", ev: &beat.Event{Timestamp: now.Add(-400 * time.Millisecond)}, want: maxWaitLimit - 400*time.Millisecond},
		{name: "exactly_at_limit", ev: &beat.Event{Timestamp: now.Add(-maxWaitLimit)}, want: 0},
		{name: "old_event", ev: &beat.Event{Timestamp: now.Add(-time.Hour)}, want: 0},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := waitLimit(test.ev, now)
			assert.Equal(t, test.want, got, "waitLimit for %s returned an unexpected limit", test.name)
		})
	}
}

func TestWaitFor(t *testing.T) {
	t.Run("found_immediately", func(t *testing.T) {
		waited, found := waitFor(func() bool { return true }, time.Second)
		assert.True(t, found, "expected the process to be found")
		assert.Less(t, waited, 100*time.Millisecond, "found immediately but waited")
	})

	t.Run("found_after_retries", func(t *testing.T) {
		var calls int
		waited, found := waitFor(func() bool {
			calls++
			return calls == 4
		}, time.Second)
		assert.True(t, found, "expected the process to be found")
		assert.Equal(t, 4, calls, "unexpected number of lookups")
		assert.Less(t, waited, time.Second, "waited for the full limit despite finding the process")
	})

	t.Run("never_found", func(t *testing.T) {
		const limit = 60 * time.Millisecond
		waited, found := waitFor(func() bool { return false }, limit)
		assert.False(t, found, "found a process that does not exist")
		assert.GreaterOrEqual(t, waited, limit, "gave up before the limit")
		assert.Less(t, waited, 10*limit, "overshot the limit by a large margin")
	})

	t.Run("small_limit_is_not_overshot", func(t *testing.T) {
		const limit = time.Millisecond
		waited, found := waitFor(func() bool { return false }, limit)
		assert.False(t, found, "found a process that does not exist")
		assert.Less(t, waited, 4*time.Millisecond, "first sleep ignored the %v limit", limit)
	})

	t.Run("zero_limit_does_one_lookup", func(t *testing.T) {
		var calls int
		_, found := waitFor(func() bool {
			calls++
			return false
		}, 0)
		assert.False(t, found, "found a process that does not exist")
		assert.Equal(t, 1, calls, "unexpected number of lookups")
	})
}
