// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix

package azureeventhub

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/logp"
)

func TestCheckpointWriterIgnoresDelayedOlderCallback(t *testing.T) {
	store := &checkpointRecorder{}
	write := newCheckpointWriter(t.Context(), store, logp.NewNopLogger())
	olderReady, resumeOlder, olderDone := make(chan struct{}), make(chan struct{}), make(chan struct{})
	var resumeOnce sync.Once
	resume := func() { resumeOnce.Do(func() { close(resumeOlder) }) }
	defer resume()
	go func() {
		defer close(olderDone)
		close(olderReady)
		// Pause before entering the guarded writer, not inside UpdateCheckpoint,
		// where this callback would already own the checkpoint mutex.
		<-resumeOlder
		write(1, writerTestMarker(1))
	}()
	waitCheckpointWriter(t, olderReady)
	write(1, writerTestMarker(2))
	assert.Equal(t, []checkpointMetadata{*writerTestMarker(2)}, store.snapshot(), "the newer callback must finish persisting before the older callback resumes")
	resume()
	waitCheckpointWriter(t, olderDone)
	write(1, writerTestMarker(2))
	assert.Equal(t, []checkpointMetadata{*writerTestMarker(2)}, store.snapshot(), "delayed older and duplicate callbacks must never overwrite a successful newer checkpoint")
}

func TestCheckpointWriterFailureDoesNotAdvanceSuccessfulPosition(t *testing.T) {
	for _, tc := range []struct {
		name      string
		callbacks []int64
		attempts  []int64
		successes []int64
	}{
		{
			name:      "next completed message advances after failure",
			callbacks: []int64{1, 2, 2, 1}, attempts: []int64{1, 2}, successes: []int64{2},
		},
		{
			name:      "failed newer marker does not suppress a delayed older completion",
			callbacks: []int64{2, 1, 3, 2, 3}, attempts: []int64{2, 1, 3}, successes: []int64{1, 3},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var attempts, successes []int64
			store := checkpointWriteFunc(func(_ context.Context, event *azeventhubs.ReceivedEventData) error {
				attempts = append(attempts, event.SequenceNumber)
				if len(attempts) == 1 {
					return errors.New("checkpoint storage unavailable")
				}
				successes = append(successes, event.SequenceNumber)
				return nil
			})
			write := newCheckpointWriter(t.Context(), store, logp.NewNopLogger())
			write(1, writerTestMarker(tc.callbacks[0]))
			assert.Equal(t, []int64{tc.callbacks[0]}, attempts, "a failed write must return without automatically retrying")
			assert.Empty(t, successes, "the first write should fail")
			for _, seq := range tc.callbacks[1:] {
				write(1, writerTestMarker(seq))
			}
			assert.Equal(t, tc.attempts, attempts, "only successful writes may suppress older or repeated callbacks")
			assert.Equal(t, tc.successes, successes, "later completions must remain able to persist progress after a write failure")
		})
	}
}

func TestCheckpointWriterSerializesOverlappingCallbacks(t *testing.T) {
	store := &checkpointRecorder{}
	entered := make(chan int64, 2)
	releaseFirst := make(chan struct{})
	var releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(releaseFirst) }) }
	defer release()
	write := newCheckpointWriter(t.Context(), checkpointWriteFunc(func(ctx context.Context, event *azeventhubs.ReceivedEventData) error {
		entered <- event.SequenceNumber
		if event.SequenceNumber == 1 {
			select {
			case <-releaseFirst:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		return store.UpdateCheckpoint(ctx, event, nil)
	}), logp.NewNopLogger())
	firstDone, secondStarted, secondDone := make(chan struct{}), make(chan struct{}), make(chan struct{})
	go func() { defer close(firstDone); write(1, writerTestMarker(1)) }()
	select {
	case seq := <-entered:
		require.Equal(t, int64(1), seq, "the first write must own the writer before the second callback starts")
	case <-time.After(5 * time.Second):
		t.Fatal("the first checkpoint write did not enter storage")
	}
	go func() {
		defer close(secondDone)
		close(secondStarted)
		write(1, writerTestMarker(2))
	}()
	waitCheckpointWriter(t, secondStarted)
	select {
	case seq := <-entered:
		t.Errorf("checkpoint %d entered storage while the first write was blocked", seq)
	case <-time.After(100 * time.Millisecond):
	}
	release()
	waitCheckpointWriter(t, firstDone)
	waitCheckpointWriter(t, secondDone)
	assert.Equal(t, []checkpointMetadata{*writerTestMarker(1), *writerTestMarker(2)}, store.snapshot(), "the second write must persist only after the first write completes")
}

func writerTestMarker(seq int64) *checkpointMetadata {
	return &checkpointMetadata{offset: strconv.FormatInt(seq*100, 10), sequenceNumber: seq}
}

func waitCheckpointWriter(t *testing.T, done <-chan struct{}) {
	t.Helper()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("checkpoint callback did not reach its synchronization point")
	}
}

type checkpointWriteFunc func(context.Context, *azeventhubs.ReceivedEventData) error

func (checkpointWriteFunc) PartitionID() string { return "0" }
func (f checkpointWriteFunc) UpdateCheckpoint(ctx context.Context, event *azeventhubs.ReceivedEventData, _ *azeventhubs.UpdateCheckpointOptions) error {
	return f(ctx, event)
}
