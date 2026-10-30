// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package logstashexporter

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/x-pack/otel/exporter/logstashexporter/internal"
)

// fillWorkQueue fills the exporter's work queue so the next enqueueWork blocks,
// and returns the works that were placed in it.
func fillWorkQueue(t *testing.T, exp *logstashExporter) []*internal.Work {
	t.Helper()
	filler := make([]*internal.Work, 0, cap(exp.workQueue))
	for range cap(exp.workQueue) {
		w := internal.NewWork(nil)
		select {
		case exp.workQueue <- w:
			filler = append(filler, w)
		default:
			require.Fail(t, "work queue filled earlier than its capacity")
		}
	}
	return filler
}

func TestEnqueueWorkAdmitsBlockedSendersInArrivalOrder(t *testing.T) {
	exp := newExporterWithDefaults(t)
	t.Cleanup(func() { _ = exp.Shutdown(context.Background()) })
	filler := fillWorkQueue(t, exp)

	const senders = 10
	sent := make([]*internal.Work, senders)
	errCh := make(chan error, senders)
	for i := range senders {
		sent[i] = internal.NewWork(nil)
		go func() { errCh <- exp.enqueueWork(t.Context(), sent[i]) }()
		// Let this sender block on the full queue before the next one starts,
		// so that arrival order is the order of the loop.
		time.Sleep(10 * time.Millisecond)
	}

	received := make([]*internal.Work, 0, len(filler)+senders)
	timeout := time.After(exporterTestDefaultTimeout)
	for range len(filler) + senders {
		select {
		case w := <-exp.workQueue:
			received = append(received, w)
		case <-timeout:
			require.Fail(t, "timed out draining the work queue")
		}
	}

	assert.Equal(t, filler, received[:len(filler)], "works already queued should be received first")
	assert.Equal(t, sent, received[len(filler):], "blocked senders should be admitted in arrival order")
	for range senders {
		assert.NoError(t, <-errCh, "enqueueWork should succeed once there is room")
	}
}

func TestEnqueueWorkReturnsWhenContextIsCancelled(t *testing.T) {
	exp := newExporterWithDefaults(t)
	t.Cleanup(func() { _ = exp.Shutdown(context.Background()) })
	fillWorkQueue(t, exp)

	ctx, cancel := context.WithCancel(t.Context())
	errCh := make(chan error, 1)
	go func() { errCh <- exp.enqueueWork(ctx, internal.NewWork(nil)) }()
	cancel()

	select {
	case err := <-errCh:
		assert.ErrorIs(t, err, context.Canceled, "a blocked enqueueWork should return the context error")
	case <-time.After(exporterTestDefaultTimeout):
		require.Fail(t, "enqueueWork did not return after the context was cancelled")
	}
}

func TestEnqueueWorkReturnsOnShutdown(t *testing.T) {
	exp := newExporterWithDefaults(t)
	fillWorkQueue(t, exp)

	errCh := make(chan error, 1)
	go func() { errCh <- exp.enqueueWork(t.Context(), internal.NewWork(nil)) }()
	time.Sleep(10 * time.Millisecond)

	require.NoError(t, exp.Shutdown(t.Context()), "shutdown should succeed")
	select {
	case err := <-errCh:
		assert.ErrorIs(t, err, errExporterShutdown, "a blocked enqueueWork should return when the exporter shuts down")
	case <-time.After(exporterTestDefaultTimeout):
		require.Fail(t, "enqueueWork did not return after the exporter was shut down")
	}

	assert.NoError(t, exp.Shutdown(t.Context()), "shutting down twice should not panic or fail")
}
