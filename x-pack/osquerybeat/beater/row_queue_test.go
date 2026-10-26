// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"context"
	"slices"
	"testing"
	"time"
)

type testQueueItem struct {
	name    string
	n       int
	profile bool
}

func (i testQueueItem) rows() int           { return i.n }
func (i testQueueItem) enqueued() time.Time { return time.Time{} }
func (i testQueueItem) evictFirst() bool    { return i.profile }

func newTestRowQueue(limit int) (*rowQueue[testQueueItem], *[]string) {
	var evicted []string
	q := newRowQueue(limit, func(i testQueueItem) { evicted = append(evicted, i.name) })
	return q, &evicted
}

func TestRowQueueEvictsOldestWhenFull(t *testing.T) {
	q, evicted := newTestRowQueue(10)
	for _, name := range []string{"a", "b", "c"} {
		q.push(testQueueItem{name: name, n: 4})
	}
	items, events := q.stats()
	if items != 2 || events != 8 {
		t.Errorf("stats() = (%d, %d); want (2, 8)", items, events)
	}
	if want := []string{"a"}; !slices.Equal(*evicted, want) {
		t.Errorf("evicted = %v; want %v", *evicted, want)
	}
}

func TestRowQueueEvictsProfilesFirst(t *testing.T) {
	q, evicted := newTestRowQueue(10)
	q.push(testQueueItem{name: "r1", n: 4})
	q.push(testQueueItem{name: "p", n: 1, profile: true})
	q.push(testQueueItem{name: "r2", n: 4})
	q.push(testQueueItem{name: "r3", n: 4})
	if want := []string{"p", "r1"}; !slices.Equal(*evicted, want) {
		t.Errorf("evicted = %v; want %v", *evicted, want)
	}
	items, events := q.stats()
	if items != 2 || events != 8 {
		t.Errorf("stats() = (%d, %d); want (2, 8)", items, events)
	}
}

func TestRowQueueAcceptsOversizedItemWhole(t *testing.T) {
	q, evicted := newTestRowQueue(10)
	q.push(testQueueItem{name: "small", n: 3})
	q.push(testQueueItem{name: "huge", n: 5000})
	items, events := q.stats()
	if items != 1 || events != 5000 {
		t.Errorf("stats() = (%d, %d); want (1, 5000)", items, events)
	}
	if want := []string{"small"}; !slices.Equal(*evicted, want) {
		t.Errorf("evicted = %v; want %v", *evicted, want)
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
			if !slices.Equal(*evicted, test.wantEvicted) {
				t.Errorf("evicted = %v; want %v", *evicted, test.wantEvicted)
			}
			items, events := q.stats()
			if items != test.wantItems || events != test.wantEvents {
				t.Errorf("stats() = (%d, %d); want (%d, %d)", items, events, test.wantItems, test.wantEvents)
			}
		})
	}
}

func TestRowQueuePopReturnsOldestAndReleasesRows(t *testing.T) {
	q, _ := newTestRowQueue(10)
	q.push(testQueueItem{name: "a", n: 4})
	q.push(testQueueItem{name: "b", n: 3})
	got, ok := q.pop(t.Context())
	if !ok || got.name != "a" {
		t.Errorf("pop() = (%v, %t); want (a, true)", got.name, ok)
	}
	items, events := q.stats()
	if items != 1 || events != 3 {
		t.Errorf("stats() = (%d, %d); want (1, 3)", items, events)
	}
	drained := q.drain()
	if len(drained) != 1 || drained[0].name != "b" {
		t.Errorf("drain() = %v; want [b]", drained)
	}
	items, events = q.stats()
	if items != 0 || events != 0 {
		t.Errorf("stats() after drain = (%d, %d); want (0, 0)", items, events)
	}
}

func TestRowQueuePopHonoursContext(t *testing.T) {
	q, _ := newTestRowQueue(10)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	done := make(chan bool, 1)
	go func() {
		_, ok := q.pop(ctx)
		done <- ok
	}()
	select {
	case ok := <-done:
		if ok {
			t.Error("pop(cancelled) = true; want false")
		}
	case <-time.After(50 * time.Millisecond):
		t.Error("pop(cancelled) did not return within 50ms")
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
		if got != "late" {
			t.Errorf("pop() = %s; want late", got)
		}
	case <-time.After(time.Second):
		t.Error("pop() did not wake for a pushed item")
	}
}

func TestScheduledResultRowsCountsDiff(t *testing.T) {
	var r scheduledResult
	r.result.DiffResults.Added = make([]map[string]string, 2)
	r.result.DiffResults.Removed = make([]map[string]string, 3)
	if got := r.rows(); got != 6 {
		t.Errorf("rows() = %d; want 6", got)
	}
}
