// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package v9

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestPendingTemplatesCache(t *testing.T) {

	type testEvent struct {
		key SessionKey
		buf *bytes.Buffer
	}

	tests := []struct {
		name             string
		eventsToAdd      []testEvent
		eventsToGet      []SessionKey
		eventsExpected   []*bytes.Buffer
		getDelay         time.Duration
		cleanInterval    time.Duration
		removalThreshold time.Duration
	}{
		{
			name: "Add and GetAndRemove different sessions with cache hit",
			eventsToAdd: []testEvent{
				{SessionKey{"127.0.0.1", 0}, bytes.NewBufferString("test-event-1")},
				{SessionKey{"127.0.0.2", 0}, bytes.NewBufferString("test-event-1")},
			},
			eventsToGet: []SessionKey{
				{"127.0.0.1", 0},
				{"127.0.0.2", 0},
			},
			eventsExpected: []*bytes.Buffer{
				bytes.NewBufferString("test-event-1"),
				bytes.NewBufferString("test-event-1"),
			},
			getDelay:         1 * time.Second,
			cleanInterval:    2 * time.Second,
			removalThreshold: 2 * time.Second,
		},
		{
			name: "Add and GetAndRemove same sessions with cache hit",
			eventsToAdd: []testEvent{
				{SessionKey{"127.0.0.1", 0}, bytes.NewBufferString("test-event-1")},
				{SessionKey{"127.0.0.1", 0}, bytes.NewBufferString("test-event-1")},
			},
			eventsToGet: []SessionKey{
				{"127.0.0.1", 0},
			},
			eventsExpected: []*bytes.Buffer{
				bytes.NewBufferString("test-event-1"),
				bytes.NewBufferString("test-event-1"),
			},
			getDelay:         1 * time.Second,
			cleanInterval:    2 * time.Second,
			removalThreshold: 2 * time.Second,
		},
		{
			name: "Add and GetAndRemove with cache miss",
			eventsToAdd: []testEvent{
				{SessionKey{"127.0.0.1", 0}, bytes.NewBufferString("test-event-1")},
				{SessionKey{"127.0.0.2", 0}, bytes.NewBufferString("test-event-1")},
			},
			eventsToGet: []SessionKey{
				{"127.0.0.1", 0},
				{"127.0.0.2", 0},
			},
			eventsExpected:   []*bytes.Buffer(nil),
			getDelay:         2 * time.Second,
			cleanInterval:    1 * time.Second,
			removalThreshold: 1 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancelFunc := context.WithCancel(context.Background())
			defer cancelFunc()
			cache := newPendingTemplatesCache()
			cache.start(ctx.Done(), tt.cleanInterval, tt.removalThreshold)
			for _, event := range tt.eventsToAdd {
				cache.Add(event.key, event.buf)
			}
			time.Sleep(tt.getDelay)
			var readEvents []*bytes.Buffer
			for _, key := range tt.eventsToGet {
				if events := cache.GetAndRemove(key); events != nil {
					readEvents = append(readEvents, events...)
				}
			}
			require.EqualValues(t, tt.eventsExpected, readEvents)

			time.Sleep(2 * tt.cleanInterval)

			cache.mtx.Lock()
			lruLen := len(cache.hp)
			lruCap := cap(cache.hp)
			cache.mtx.Unlock()
			require.Zero(t, lruLen)
			require.Zero(t, lruCap)

			cancelFunc()
			cache.wait()
		})
	}
}
