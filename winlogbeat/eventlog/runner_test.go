// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package eventlog

import (
	"context"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/management/status"
	"github.com/elastic/beats/v7/winlogbeat/checkpoint"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

var errRepeatedRecordIDGap = errors.New("repeated record ID gap")

func TestRunReopensFromLatestCheckpointAfterRecoverableGap(t *testing.T) {
	const (
		originalCheckpoint = 3599
		lastBeforeGap      = 3600
		firstAfterGap      = 3636
		gapRetryLimit      = 3
	)

	restore := stubRecoverableForTest(errRepeatedRecordIDGap)
	defer restore()

	api := &replayingGapEventLog{
		t:                  t,
		lastBeforeGap:      lastBeforeGap,
		firstAfterGap:      firstAfterGap,
		gapRetryLimit:      gapRetryLimit,
		maxOpenInvocations: gapRetryLimit + 2,
	}

	err := Run(
		noOpStatusReporter{},
		context.Background(),
		monitoring.NewRegistry(),
		api,
		checkpoint.EventLogState{Name: "System", RecordNumber: originalCheckpoint},
		noOpPublisher{},
		logp.NewLogger("eventlog_runner_test"),
	)
	require.NoError(t, err)

	require.Equal(t,
		[]uint64{originalCheckpoint, lastBeforeGap, lastBeforeGap, lastBeforeGap},
		api.openedFromRecordNumbers,
		"recoverable gap retries must reopen from the latest in-memory checkpoint, not the original checkpoint")
	require.True(t, api.gapAccepted, "gap should be accepted after repeated retries")
	require.Equal(t, uint64(firstAfterGap), api.Checkpoint().RecordNumber)
}

func stubRecoverableForTest(recoverableErr error) func() {
	originalIsRecoverable := isRecoverable
	originalOpenDelay := openRetryInitialDelay
	originalReadDelay := readRetryInitialDelay
	originalMaxDelay := retryMaxDelay

	isRecoverable = func(err error, _ bool) bool {
		return errors.Is(err, recoverableErr)
	}
	openRetryInitialDelay = time.Nanosecond
	readRetryInitialDelay = time.Nanosecond
	retryMaxDelay = time.Nanosecond

	return func() {
		isRecoverable = originalIsRecoverable
		openRetryInitialDelay = originalOpenDelay
		readRetryInitialDelay = originalReadDelay
		retryMaxDelay = originalMaxDelay
	}
}

type replayingGapEventLog struct {
	t *testing.T

	checkpoint              checkpoint.EventLogState
	openedFromRecordNumbers []uint64

	lastBeforeGap uint64
	firstAfterGap uint64

	gapRetryCount      int
	gapRetryLimit      int
	gapAccepted        bool
	maxOpenInvocations int
}

func (l *replayingGapEventLog) Open(state checkpoint.EventLogState, _ *monitoring.Registry) error {
	l.openedFromRecordNumbers = append(l.openedFromRecordNumbers, state.RecordNumber)
	require.LessOrEqual(l.t, len(l.openedFromRecordNumbers), l.maxOpenInvocations,
		"gap retry loop reopened too many times without accepting the gap")

	l.checkpoint = state
	return nil
}

func (l *replayingGapEventLog) Checkpoint() checkpoint.EventLogState {
	return l.checkpoint
}

func (l *replayingGapEventLog) Read() ([]Record, error) {
	if l.gapAccepted {
		return nil, io.EOF
	}

	if l.checkpoint.RecordNumber < l.lastBeforeGap {
		// Simulate replaying a successfully processed or filtered event before
		// hitting the same gap again. winEventLog resets the gap retry counter
		// after each successful event.
		l.gapRetryCount = 0
		l.checkpoint.RecordNumber = l.lastBeforeGap
	}

	l.gapRetryCount++
	if l.gapRetryCount <= l.gapRetryLimit {
		return nil, errRepeatedRecordIDGap
	}

	l.checkpoint.RecordNumber = l.firstAfterGap
	l.gapAccepted = true
	return nil, io.EOF
}

func (l *replayingGapEventLog) Reset() error {
	return nil
}

func (l *replayingGapEventLog) Close() error {
	return nil
}

func (l *replayingGapEventLog) Name() string {
	return "System"
}

func (l *replayingGapEventLog) Channel() string {
	return "System"
}

func (l *replayingGapEventLog) IsFile() bool {
	return false
}

func (l *replayingGapEventLog) IgnoreMissingChannel() bool {
	return false
}

// TestRunClosesEventLogWithoutRacingRead verifies that when the runner's
// context is cancelled while a Read is in progress, the event log is not
// closed until that Read has returned. Closing the event log frees native
// Windows Event Log handles; doing so concurrently with an in-flight
// Read/render previously caused an access violation crash during shutdown.
func TestRunClosesEventLogWithoutRacingRead(t *testing.T) {
	api := &concurrencyProbeEventLog{readStarted: make(chan struct{})}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- Run(
			noOpStatusReporter{},
			ctx,
			monitoring.NewRegistry(),
			api,
			checkpoint.EventLogState{Name: "System"},
			noOpPublisher{},
			logp.NewLogger("eventlog_runner_test"),
		)
	}()

	// Cancel only once a Read is actively in progress so that a close racing
	// the read would be observable.
	<-api.readStarted
	cancel()

	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(10 * time.Second):
		t.Fatal("Run did not return after context cancellation")
	}

	require.False(t, api.observedOverlap(), "Close must not run while a Read is in progress")
	require.True(t, api.wasClosed(), "event log must be closed when the runner returns")
}

// concurrencyProbeEventLog records whether Read and Close ever overlap.
type concurrencyProbeEventLog struct {
	mu          sync.Mutex
	reading     bool
	overlap     bool
	closed      bool
	readStarted chan struct{}
	signalOnce  sync.Once
}

func (l *concurrencyProbeEventLog) Open(_ checkpoint.EventLogState, _ *monitoring.Registry) error {
	return nil
}

func (l *concurrencyProbeEventLog) Checkpoint() checkpoint.EventLogState {
	return checkpoint.EventLogState{Name: "System"}
}

func (l *concurrencyProbeEventLog) Read() ([]Record, error) {
	l.mu.Lock()
	if l.closed {
		l.overlap = true
	}
	l.reading = true
	l.mu.Unlock()

	l.signalOnce.Do(func() { close(l.readStarted) })

	// Keep the read in progress long enough to expose a racing close.
	time.Sleep(50 * time.Millisecond)

	l.mu.Lock()
	l.reading = false
	l.mu.Unlock()
	return nil, nil
}

func (l *concurrencyProbeEventLog) Reset() error { return nil }

func (l *concurrencyProbeEventLog) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.reading {
		l.overlap = true
	}
	l.closed = true
	return nil
}

func (l *concurrencyProbeEventLog) observedOverlap() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.overlap
}

func (l *concurrencyProbeEventLog) wasClosed() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.closed
}

func (l *concurrencyProbeEventLog) Name() string               { return "System" }
func (l *concurrencyProbeEventLog) Channel() string            { return "System" }
func (l *concurrencyProbeEventLog) IsFile() bool               { return false }
func (l *concurrencyProbeEventLog) IgnoreMissingChannel() bool { return false }

type noOpPublisher struct{}

func (noOpPublisher) Publish([]Record) error {
	return nil
}

type noOpStatusReporter struct{}

func (noOpStatusReporter) UpdateStatus(status.Status, string) {}
