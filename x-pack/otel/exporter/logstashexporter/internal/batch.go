// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package internal

import (
	"context"
	"sync"
	"sync/atomic"

	"go.opentelemetry.io/collector/pdata/plog"

	"github.com/elastic/beats/v7/libbeat/publisher"
)

type LogBatchResult uint

const (
	LogBatchResultACK LogBatchResult = 1 << iota
	LogBatchResultDrop
	LogBatchResultRetry
	LogBatchResultCancelled
)

type LogBatch struct {
	retries       atomic.Uint64
	pendingEvents []publisher.Event
	resultCh      chan LogBatchResult
	mu            sync.RWMutex
}

func NewLogBatch(ctx context.Context, logs plog.Logs) (*LogBatch, error) {
	events, err := createEvents(ctx, &logs)
	if err != nil {
		return nil, err
	}
	return &LogBatch{
		pendingEvents: events,
		resultCh:      make(chan LogBatchResult, 1),
	}, nil
}

func createEvents(ctx context.Context, logs *plog.Logs) ([]publisher.Event, error) {
	var events []publisher.Event
	for _, rl := range logs.ResourceLogs().All() {
		for _, sl := range rl.ScopeLogs().All() {
			for _, lr := range sl.LogRecords().All() {
				record, err := parseEvent(ctx, &lr)
				if err != nil {
					return nil, err
				}
				events = append(events, publisher.Event{Content: record})
			}
		}
	}
	return events, nil
}

func (b *LogBatch) Events() []publisher.Event {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.pendingEvents
}

func (b *LogBatch) ACK() {
	b.notifyResult(LogBatchResultACK)
}

func (b *LogBatch) Drop() {
	b.notifyResult(LogBatchResultDrop)
}

func (b *LogBatch) Retry() {
	b.AddRetry(1)
	b.notifyResult(LogBatchResultRetry)
}

func (b *LogBatch) RetryEvents(events []publisher.Event) {
	b.AddRetry(1)
	b.mu.Lock()
	b.pendingEvents = events
	b.mu.Unlock()
	b.notifyResult(LogBatchResultRetry)
}

func (b *LogBatch) Cancelled() {
	b.notifyResult(LogBatchResultCancelled)
}

// SplitRetry is not used by Logstash clients currently
func (b *LogBatch) SplitRetry() bool {
	return false
}

func (b *LogBatch) NumRetries() int {
	return int(b.retries.Load())
}

func (b *LogBatch) Result() chan LogBatchResult {
	return b.resultCh
}

func (b *LogBatch) notifyResult(result LogBatchResult) {
	select {
	case b.resultCh <- result:
	default:
		// already signaled
	}
}

// AddRetry adds delta to the number of retries for this batch.
// If delta is negative, it will decrease the number of retries but not below zero.
func (b *LogBatch) AddRetry(delta int) {
	if delta == 0 {
		return
	}
	if delta > 0 {
		b.retries.Add(uint64(delta))
		return
	}
	for {
		oldValue := b.retries.Load()
		if oldValue == 0 {
			return
		}
		newValue := int(oldValue) + delta // delta is negative
		if newValue <= 0 {
			b.retries.Store(0)
			return
		}
		if b.retries.CompareAndSwap(oldValue, uint64(newValue)) {
			return
		}
	}
}
