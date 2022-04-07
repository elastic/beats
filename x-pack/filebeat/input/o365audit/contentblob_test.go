// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package o365audit

import (
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v8/libbeat/beat"
	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/libbeat/logp"
	"github.com/elastic/beats/v8/x-pack/filebeat/input/o365audit/poll"
)

type contentStore struct {
	events  []beat.Event
	stopped bool
}

var errStopped = errors.New("stopped")

func (s *contentStore) onEvent(b beat.Event, checkpointUpdate interface{}) error {
	b.Private = checkpointUpdate
	s.events = append(s.events, b)
	if s.stopped {
		return errStopped
	}
	return nil
}

func (f *fakePoll) BlobContent(t testing.TB, b poll.Transaction, data []common.MapStr, nextUrl string) poll.Transaction {
	urls, next := f.deliverResult(t, b, data, nextUrl)
	if !assert.Empty(t, urls) {
		t.Fatal("blob returned urls to fetch")
	}
	return next
}

func makeEvent(ts time.Time, id string) common.MapStr {
	return common.MapStr{
		"CreationTime": ts.Format(apiDateFormat),
		"Id":           id,
	}
}

func validateBlobs(t testing.TB, store contentStore, expected []string, c checkpoint) checkpoint {
	assert.Len(t, store.events, len(expected))
	for idx := range expected {
		id, err := getString(store.events[idx].Fields, fieldsPrefix+".Id")
		if !assert.NoError(t, err) {
			t.Fatal(err)
		}
		assert.Equal(t, expected[idx], id)
	}
	prev := c
	baseLine := c.Line
	for idx, id := range expected {
		ev := store.events[idx]
		cursor, ok := ev.Private.(checkpoint)
		if !assert.True(t, ok) {
			t.Fatal("no cursor for event id", id)
		}
		assert.Equal(t, idx+1+baseLine, cursor.Line)
		assert.True(t, prev.Before(cursor))
		prev = cursor
	}
	return prev
}

func TestContentBlob(t *testing.T) {
	var f fakePoll
	var store contentStore
	ctx := apiEnvironment{
		Logger:   logp.L(),
		Callback: store.onEvent,
	}
	baseCursor := checkpoint{Timestamp: time.Now()}
	query := ContentBlob("http://test.localhost/", baseCursor, ctx)
	data := []common.MapStr{
		makeEvent(now.Add(-time.Hour), "e1"),
		makeEvent(now.Add(-2*time.Hour), "e2"),
		makeEvent(now.Add(-30*time.Minute), "e3"),
		makeEvent(now.Add(-10*time.Second), "e4"),
		makeEvent(now.Add(-20*time.Minute), "e5"),
	}
	expected := []string{"e1", "e2", "e3", "e4", "e5"}
	next := f.BlobContent(t, query, data, "")
	assert.Nil(t, next)
	c := validateBlobs(t, store, expected, baseCursor)
	assert.Equal(t, len(expected), c.Line)
}

func TestContentBlobResumeToLine(t *testing.T) {
	var f fakePoll
	var store contentStore
	ctx := testConfig()
	ctx.Callback = store.onEvent
	baseCursor := checkpoint{Timestamp: time.Now()}
	const skip = 3
	baseCursor.Line = skip
	query := ContentBlob("http://test.localhost/", baseCursor, ctx).WithSkipLines(skip)
	data := []common.MapStr{
		makeEvent(now.Add(-time.Hour), "e1"),
		makeEvent(now.Add(-2*time.Hour), "e2"),
		makeEvent(now.Add(-30*time.Minute), "e3"),
		makeEvent(now.Add(-10*time.Second), "e4"),
		makeEvent(now.Add(-20*time.Minute), "e5"),
	}
	expected := []string{"e4", "e5"}
	next := f.BlobContent(t, query, data, "")
	assert.Nil(t, next)
	c := validateBlobs(t, store, expected, baseCursor)
	assert.Equal(t, len(expected), c.Line-skip)
}

func TestContentBlobPaged(t *testing.T) {
	var f fakePoll
	var store contentStore
	ctx := apiEnvironment{
		Logger:   logp.L(),
		Callback: store.onEvent,
	}
	baseCursor := checkpoint{Timestamp: time.Now()}
	query := ContentBlob("http://test.localhost/", baseCursor, ctx)
	data := []common.MapStr{
		makeEvent(now.Add(-time.Hour), "e1"),
		makeEvent(now.Add(-2*time.Hour), "e2"),
		makeEvent(now.Add(-30*time.Minute), "e3"),
		makeEvent(now.Add(-10*time.Second), "e4"),
		makeEvent(now.Add(-20*time.Minute), "e5"),
		makeEvent(now.Add(-20*time.Minute), "e6"),
	}
	expected := []string{"e1", "e2", "e3"}
	next := f.BlobContent(t, query, data[:3], "http://test.localhost/page/2")
	assert.NotNil(t, next)
	assert.IsType(t, paginator{}, next)
	c := validateBlobs(t, store, expected, baseCursor)
	assert.Equal(t, 3, c.Line)
	store.events = nil
	next = f.BlobContent(t, next, data[3:5], "http://test.localhost/page/3")
	assert.IsType(t, paginator{}, next)
	expected = []string{"e4", "e5"}
	c = validateBlobs(t, store, expected, c)
	assert.Equal(t, 5, c.Line)
	store.events = nil
	next = f.BlobContent(t, next, data[5:], "")
	assert.Nil(t, next)
	expected = []string{"e6"}
	c = validateBlobs(t, store, expected, c)
	assert.Equal(t, 6, c.Line)
}
