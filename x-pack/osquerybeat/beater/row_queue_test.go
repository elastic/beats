// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testQueueItem struct {
	name    string
	n       int
	profile bool
	checks  *int // Counts evictFirst calls when set.
}

func (i testQueueItem) rows() int           { return i.n }
func (i testQueueItem) enqueued() time.Time { return time.Time{} }
func (i testQueueItem) evictFirst() bool {
	if i.checks != nil {
		*i.checks++
	}
	return i.profile
}

func newTestRowQueue(limit int) (*rowQueue[testQueueItem], *[]string) {
	var evicted []string
	q := newRowQueue(limit, func(i testQueueItem) { evicted = append(evicted, i.name) })
	return q, &evicted
}

func assertStats(t *testing.T, q *rowQueue[testQueueItem], wantItems, wantEvents int) {
	t.Helper()
	items, events := q.stats()
	assert.Equal(t, wantItems, items, "queued items")
	assert.Equal(t, wantEvents, events, "queued rows")
}

func TestRowQueueEvictsOldestWhenFull(t *testing.T) {
	q, evicted := newTestRowQueue(10)
	for _, name := range []string{"a", "b", "c"} {
		q.push(testQueueItem{name: name, n: 4})
	}
	assertStats(t, q, 2, 8)
	assert.Equal(t, []string{"a"}, *evicted, "only the oldest item should make room for the third")
}

func TestRowQueueEvictsProfilesFirst(t *testing.T) {
	q, evicted := newTestRowQueue(10)
	q.push(testQueueItem{name: "r1", n: 4})
	q.push(testQueueItem{name: "p", n: 1, profile: true})
	q.push(testQueueItem{name: "r2", n: 4})
	q.push(testQueueItem{name: "r3", n: 4})
	assert.Equal(t, []string{"p", "r1"}, *evicted, "the profile item should go before the oldest result")
	assertStats(t, q, 2, 8)
}

func TestRowQueueDropsProfileRatherThanResult(t *testing.T) {
	q, evicted := newTestRowQueue(10)
	q.push(testQueueItem{name: "r1", n: 6})
	q.push(testQueueItem{name: "r2", n: 4})
	q.push(testQueueItem{name: "p", n: 1, profile: true})
	assert.Equal(t, []string{"p"}, *evicted, "an incoming profile should be dropped instead of evicting a result")
	assertStats(t, q, 2, 10)
}

func TestRowQueueTracksEvictFirstItems(t *testing.T) {
	// Popping a result must not hide a queued profile from eviction.
	q, evicted := newTestRowQueue(10)
	q.push(testQueueItem{name: "r0", n: 4})
	q.push(testQueueItem{name: "r1", n: 4})
	q.push(testQueueItem{name: "p", n: 1, profile: true})
	_, ok := q.pop(t.Context())
	require.True(t, ok, "pop the oldest result")
	q.push(testQueueItem{name: "r2", n: 4})
	q.push(testQueueItem{name: "r3", n: 4})
	assert.Equal(t, []string{"p", "r1"}, *evicted, "the profile should still go before the oldest result")

	// Without queued profiles, eviction must not scan the queue for one.
	var checks int
	q, _ = newTestRowQueue(10)
	for range 20 {
		q.push(testQueueItem{n: 1, checks: &checks})
	}
	assert.LessOrEqual(t, checks, 2*20, "each item should be checked at most when pushed and when removed")
}

func TestRowQueueAcceptsOversizedItemWhole(t *testing.T) {
	tests := []struct {
		name        string
		queued      []testQueueItem
		wantEvicted []string
	}{
		{
			name:        "evicts result",
			queued:      []testQueueItem{{name: "small", n: 3}},
			wantEvicted: []string{"small"},
		},
		{
			name:        "evicts profile then result",
			queued:      []testQueueItem{{name: "result", n: 3}, {name: "profile", n: 1, profile: true}},
			wantEvicted: []string{"profile", "result"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			q, evicted := newTestRowQueue(10)
			for _, item := range test.queued {
				q.push(item)
			}
			q.push(testQueueItem{name: "huge", n: 5000})
			assertStats(t, q, 1, 5000)
			assert.Equal(t, test.wantEvicted, *evicted, "everything older should be evicted for the oversized item")
		})
	}
}

func TestRowQueueShrinkEvicts(t *testing.T) {
	tests := []struct {
		name        string
		rows        []int
		wantEvicted []string
		wantItems   int
		wantEvents  int
	}{
		{name: "until within limit", rows: []int{2, 2, 2, 2}, wantEvicted: []string{"0", "1", "2"}, wantItems: 1, wantEvents: 2},
		{name: "keeps last item", rows: []int{8}, wantEvicted: nil, wantItems: 1, wantEvents: 8},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			q, evicted := newTestRowQueue(10)
			for i, n := range test.rows {
				q.push(testQueueItem{name: string(rune('0' + i)), n: n})
			}
			q.setLimit(3)
			assert.Equal(t, test.wantEvicted, *evicted, "items evicted by shrinking the limit")
			assertStats(t, q, test.wantItems, test.wantEvents)
		})
	}
}

func TestRowQueuePopReturnsOldestAndReleasesRows(t *testing.T) {
	q, _ := newTestRowQueue(10)
	q.push(testQueueItem{name: "a", n: 4})
	q.push(testQueueItem{name: "b", n: 3})
	got, ok := q.pop(t.Context())
	assert.True(t, ok, "pop should return a queued item")
	assert.Equal(t, "a", got.name, "pop should return the oldest item")
	assertStats(t, q, 1, 3)

	drained := q.drain()
	require.Len(t, drained, 1, "drain should return the remaining item")
	assert.Equal(t, "b", drained[0].name, "drain should return the remaining item")
	assertStats(t, q, 0, 0)
}

func TestRowQueuePopHonoursContext(t *testing.T) {
	tests := []struct {
		name  string
		block time.Duration // How long pop waits on the empty queue before cancellation.
	}{
		{name: "already cancelled"},
		{name: "cancelled while blocked", block: 10 * time.Millisecond},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			q, _ := newTestRowQueue(10)
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			if test.block == 0 {
				cancel()
			}
			done := make(chan bool, 1)
			go func() {
				_, ok := q.pop(ctx)
				done <- ok
			}()
			if test.block > 0 {
				time.Sleep(test.block)
				cancel()
			}
			select {
			case ok := <-done:
				assert.False(t, ok, "pop should report the cancelled context")
			case <-time.After(50 * time.Millisecond):
				t.Error("pop did not return within 50ms of cancellation")
			}
		})
	}
}

func TestRowQueuePopWaitsForPush(t *testing.T) {
	q, _ := newTestRowQueue(10)
	done := make(chan string, 1)
	go func() {
		got, _ := q.pop(t.Context())
		done <- got.name
	}()
	q.push(testQueueItem{name: "late", n: 1})
	select {
	case got := <-done:
		assert.Equal(t, "late", got, "pop should return the pushed item")
	case <-time.After(time.Second):
		t.Error("pop did not wake for a pushed item")
	}
}

func TestScheduledResultRowsCountsDiff(t *testing.T) {
	var r scheduledResult
	r.result.DiffResults.Added = make([]map[string]string, 2)
	r.result.DiffResults.Removed = make([]map[string]string, 3)
	assert.Equal(t, 6, r.rows(), "added and removed rows plus the response document")
}

func TestResolvedScheduledResultRowsCountsProfile(t *testing.T) {
	r := resolvedScheduledResult{batches: []resolvedRows{{hits: make([]map[string]any, 2)}, {hits: make([]map[string]any, 3)}}}
	assert.Equal(t, 6, r.rows(), "rows of every batch plus the response document")
	r.profile = map[string]any{}
	assert.Equal(t, 7, r.rows(), "a profile document counts as one more row")
}

func TestRowQueueObservesEveryChange(t *testing.T) {
	q, _ := newTestRowQueue(10)
	var got [2]int
	q.observe = func(items, events int) { got = [2]int{items, events} }

	q.push(testQueueItem{name: "a", n: 4})
	q.push(testQueueItem{name: "b", n: 4})
	assert.Equal(t, [2]int{2, 8}, got, "observe the queue after a push")

	q.setLimit(5)
	assert.Equal(t, [2]int{1, 4}, got, "observe the queue after a shrinking limit evicts")

	_, ok := q.pop(t.Context())
	require.True(t, ok, "pop the queued item")
	assert.Equal(t, [2]int{0, 0}, got, "observe the queue after a pop")

	q.push(testQueueItem{name: "c", n: 2})
	q.drain()
	assert.Equal(t, [2]int{0, 0}, got, "observe the queue after a drain")
}
