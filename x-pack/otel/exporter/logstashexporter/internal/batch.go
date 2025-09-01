// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package internal

import (
	"context"

	"go.opentelemetry.io/collector/pdata/plog"

	"github.com/elastic/beats/v7/libbeat/publisher"
)

type LogBatchResult struct {
	Acked     bool
	Dropped   bool
	Retry     bool
	Cancelled bool
	Retries   int
}

type LogBatch struct {
	pendingEvents []publisher.Event
	result        *LogBatchResult
}

func NewLogBatch(ctx context.Context, logs plog.Logs) (*LogBatch, error) {
	events, err := createEvents(ctx, &logs)
	if err != nil {
		return nil, err
	}
	return &LogBatch{
		pendingEvents: events,
		result:        &LogBatchResult{},
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
	return b.pendingEvents
}

func (b *LogBatch) ACK() {
	b.result.Acked = true
}

func (b *LogBatch) Drop() {
	b.result.Dropped = true
}

func (b *LogBatch) Retry() {
	b.result.Retry = true
	b.result.Retries++
}

func (b *LogBatch) RetryEvents(events []publisher.Event) {
	b.pendingEvents = events
	b.Retry()
}

// SplitRetry is not used by Logstash clients currently
func (b *LogBatch) SplitRetry() bool {
	return false
}

func (b *LogBatch) Cancelled() {
	b.result.Cancelled = true
}

func (b *LogBatch) Result() *LogBatchResult {
	return b.result
}
