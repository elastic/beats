// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"context"
	"errors"
	"testing"

	"github.com/elastic/beats/v7/libbeat/management/status"
	"github.com/elastic/elastic-agent-libs/logp"
)

func TestSQSHealth_LifecyclePassthrough(t *testing.T) {
	r := &testReporter{}
	h := newSQSHealth(r, logp.NewNopLogger())

	h.UpdateStatus(status.Starting, "Input starting")
	h.UpdateStatus(status.Configuring, "Configuring")
	h.UpdateStatus(status.Running, "Input is running")
	h.UpdateStatus(status.Stopped, "Done")

	if len(r.updates) != 4 {
		t.Fatalf("want 4 updates, got %d: %+v", len(r.updates), r.updates)
	}
	if r.updates[0].status != status.Starting {
		t.Errorf("updates[0]: got %v, want Starting", r.updates[0].status)
	}
	if r.updates[3].status != status.Stopped {
		t.Errorf("updates[3]: got %v, want Stopped", r.updates[3].status)
	}
}

func TestSQSHealth_ReceiveThreshold(t *testing.T) {
	r := &testReporter{}
	h := newSQSHealth(r, logp.NewNopLogger())

	// Start as Running.
	h.UpdateStatus(status.Running, "Input is running")
	if len(r.updates) != 1 || r.updates[0].status != status.Running {
		t.Fatalf("want Running, got %+v", r.updates)
	}

	// First two receive errors should NOT degrade (below threshold of 3).
	h.UpdateStatus(status.Degraded, "SQS error 1")
	h.UpdateStatus(status.Degraded, "SQS error 2")
	if len(r.updates) != 1 {
		t.Fatalf("want no new updates after 2 receive errors, got %d total: %+v", len(r.updates), r.updates)
	}

	// Third receive error hits threshold → Degraded.
	h.UpdateStatus(status.Degraded, "SQS error 3")
	if len(r.updates) != 2 {
		t.Fatalf("want 2 updates after threshold hit, got %d: %+v", len(r.updates), r.updates)
	}
	if r.updates[1].status != status.Degraded {
		t.Errorf("want Degraded, got %v", r.updates[1].status)
	}

	// Successful receive clears the condition → Running.
	h.UpdateStatus(status.Running, "Input is running")
	if len(r.updates) != 3 {
		t.Fatalf("want 3 updates after receive OK, got %d: %+v", len(r.updates), r.updates)
	}
	if r.updates[2].status != status.Running {
		t.Errorf("want Running after receive OK, got %v", r.updates[2].status)
	}
}

func TestSQSHealth_TransientReceiveErrorStaysRunning(t *testing.T) {
	r := &testReporter{}
	h := newSQSHealth(r, logp.NewNopLogger())
	h.UpdateStatus(status.Running, "Input is running")

	// Two errors then success → should stay Running the whole time.
	h.UpdateStatus(status.Degraded, "error 1")
	h.UpdateStatus(status.Degraded, "error 2")
	h.UpdateStatus(status.Running, "Input is running")

	// Only the initial Running should have been published (no state change).
	if len(r.updates) != 1 {
		t.Fatalf("want 1 update (initial Running), got %d: %+v", len(r.updates), r.updates)
	}
}

func TestSQSHealth_DeleteFailureDegrades(t *testing.T) {
	r := &testReporter{}
	h := newSQSHealth(r, logp.NewNopLogger())
	h.UpdateStatus(status.Running, "Input is running")

	h.SetDeleteFailed(errors.New("access denied"))
	if len(r.updates) != 2 {
		t.Fatalf("want 2 updates, got %d: %+v", len(r.updates), r.updates)
	}
	if r.updates[1].status != status.Degraded {
		t.Errorf("want Degraded after delete fail, got %v", r.updates[1].status)
	}

	// ClearDisposition restores Running.
	h.ClearDisposition()
	if len(r.updates) != 3 {
		t.Fatalf("want 3 updates, got %d: %+v", len(r.updates), r.updates)
	}
	if r.updates[2].status != status.Running {
		t.Errorf("want Running after clear, got %v", r.updates[2].status)
	}
}

func TestSQSHealth_FinalizeFailureDegrades(t *testing.T) {
	r := &testReporter{}
	h := newSQSHealth(r, logp.NewNopLogger())
	h.UpdateStatus(status.Running, "Input is running")

	h.SetFinalizeFailed(errors.New("bucket gone"))
	if len(r.updates) != 2 || r.updates[1].status != status.Degraded {
		t.Fatalf("want Degraded, got %+v", r.updates)
	}

	h.ClearDisposition()
	if r.updates[len(r.updates)-1].status != status.Running {
		t.Errorf("want Running after clear, got %v", r.updates[len(r.updates)-1].status)
	}
}

func TestSQSHealth_PoisonPillDegrades(t *testing.T) {
	r := &testReporter{}
	h := newSQSHealth(r, logp.NewNopLogger())
	h.UpdateStatus(status.Running, "Input is running")

	h.RecordPoisonPill(errors.New("bad message"))
	if len(r.updates) < 2 || r.updates[1].status != status.Degraded {
		t.Fatalf("want Degraded after poison pill, got %+v", r.updates)
	}

	// Next successful delete clears the condition.
	h.ClearDisposition()
	last := r.updates[len(r.updates)-1]
	if last.status != status.Running {
		t.Errorf("want Running after clear, got %v", last.status)
	}
}

func TestSQSHealth_ContextCanceledSuppressed(t *testing.T) {
	r := &testReporter{}
	h := newSQSHealth(r, logp.NewNopLogger())
	h.UpdateStatus(status.Running, "Input is running")

	// Shutdown errors (context.Canceled only) should not set any conditions.
	h.SetDeleteFailed(context.Canceled)
	h.SetFinalizeFailed(context.Canceled)
	h.RecordPoisonPill(context.Canceled)
	h.SetProcessingError(context.Canceled)

	if len(r.updates) != 1 {
		t.Fatalf("want no status change from shutdown errors, got %d: %+v", len(r.updates), r.updates)
	}
}

func TestSQSHealth_WorkerErrorDegrades(t *testing.T) {
	r := &testReporter{}
	h := newSQSHealth(r, logp.NewNopLogger())
	h.UpdateStatus(status.Running, "Input is running")

	h.SetWorkerError(errors.New("pipeline connect failed"))
	if len(r.updates) != 2 || r.updates[1].status != status.Degraded {
		t.Fatalf("want Degraded after worker error, got %+v", r.updates)
	}
}

func TestSQSHealth_ReceiveOKDoesNotClearDeleteCondition(t *testing.T) {
	r := &testReporter{}
	h := newSQSHealth(r, logp.NewNopLogger())
	h.UpdateStatus(status.Running, "Input is running")

	// Set a delete failure condition.
	h.SetDeleteFailed(errors.New("fail"))
	if r.updates[len(r.updates)-1].status != status.Degraded {
		t.Fatalf("want Degraded, got %+v", r.updates)
	}

	// Successful receive should NOT clear the delete condition.
	h.UpdateStatus(status.Running, "Input is running")
	last := r.updates[len(r.updates)-1]
	if last.status != status.Degraded {
		t.Errorf("want still Degraded after receive OK (delete condition persists), got %v", last.status)
	}
}

func TestSQSHealth_Dedup(t *testing.T) {
	r := &testReporter{}
	h := newSQSHealth(r, logp.NewNopLogger())

	// Publishing the same status+msg twice should only forward once.
	h.UpdateStatus(status.Running, "Input is running")
	h.UpdateStatus(status.Running, "Input is running")
	if len(r.updates) != 1 {
		t.Fatalf("want 1 update (dedup), got %d: %+v", len(r.updates), r.updates)
	}
}

func TestSQSHealth_ProcessingErrorDoesNotDegrade(t *testing.T) {
	r := &testReporter{}
	h := newSQSHealth(r, logp.NewNopLogger())
	h.UpdateStatus(status.Running, "Input is running")

	// Below threshold: no status change.
	h.SetProcessingError(errors.New("s3 download failed"))
	h.SetProcessingError(errors.New("s3 download failed"))
	if len(r.updates) != 1 {
		t.Fatalf("want no status change from transient processing errors, got %d: %+v", len(r.updates), r.updates)
	}
}

func TestSQSHealth_SustainedProcessingErrorDegrades(t *testing.T) {
	r := &testReporter{}
	h := newSQSHealth(r, logp.NewNopLogger())
	h.UpdateStatus(status.Running, "Input is running")

	// Hit the threshold (3 consecutive).
	h.SetProcessingError(errors.New("access denied"))
	h.SetProcessingError(errors.New("access denied"))
	h.SetProcessingError(errors.New("access denied"))
	if len(r.updates) != 2 || r.updates[1].status != status.Degraded {
		t.Fatalf("want Degraded after sustained processing failures, got %+v", r.updates)
	}

	// Successful disposition clears it.
	h.ClearDisposition()
	last := r.updates[len(r.updates)-1]
	if last.status != status.Running {
		t.Errorf("want Running after clear, got %v", last.status)
	}
}

func TestSQSHealth_LatchAfterStopped(t *testing.T) {
	r := &testReporter{}
	h := newSQSHealth(r, logp.NewNopLogger())
	h.UpdateStatus(status.Running, "Input is running")
	h.UpdateStatus(status.Stopped, "Done")

	n := len(r.updates)

	// Further updates should be ignored.
	h.ClearDisposition()
	h.SetDeleteFailed(errors.New("fail"))
	h.SetWorkerError(errors.New("fail"))
	h.UpdateStatus(status.Running, "Input is running")

	if len(r.updates) != n {
		t.Fatalf("want no updates after Stopped, got %d (was %d): %+v", len(r.updates), n, r.updates[n:])
	}
}

func TestSQSHealth_LatchAfterFailed(t *testing.T) {
	r := &testReporter{}
	h := newSQSHealth(r, logp.NewNopLogger())
	h.UpdateStatus(status.Running, "Input is running")
	h.UpdateStatus(status.Failed, "Broken")

	n := len(r.updates)

	h.ClearDisposition()
	h.SetDeleteFailed(errors.New("fail"))
	h.UpdateStatus(status.Running, "Input is running")

	if len(r.updates) != n {
		t.Fatalf("want no updates after Failed, got %d (was %d): %+v", len(r.updates), n, r.updates[n:])
	}
}

// testReporter records all status updates without dedup.
type testReporter struct {
	updates []mgmtStatusUpdate
}

func (r *testReporter) UpdateStatus(s status.Status, msg string) {
	r.updates = append(r.updates, mgmtStatusUpdate{status: s, msg: msg})
}
