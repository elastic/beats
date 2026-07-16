// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/elastic/beats/v7/libbeat/management/status"
	"github.com/elastic/elastic-agent-libs/logp"
)

// defaultRecvFailThreshold is the number of consecutive ReceiveMessage
// failures required before the input reports Degraded for receive errors.
const defaultRecvFailThreshold = 3

// defaultProcFailThreshold is the number of consecutive S3 object processing
// failures required before the input reports Degraded for processing errors.
const defaultProcFailThreshold = 3

// sqsHealth aggregates health signals from the SQS reader, S3 processors,
// and message disposition callbacks into a single coherent status for Fleet.
//
// It replaces the pattern of scattered UpdateStatus calls that race on one
// reporter. Processing code reports events (Set*/Clear*); the aggregator
// decides what state to publish based on active conditions.
//
// sqsHealth implements status.StatusReporter so it can be passed to
// readSQSMessages (which expects that interface) without signature changes.
type sqsHealth struct {
	mu       sync.Mutex
	reporter status.StatusReporter
	log      *logp.Logger

	conditions map[condition]healthCondition

	consecutiveRecvFails int
	recvFailThreshold    int

	consecutiveProcFails int
	procFailThreshold    int

	currentStatus status.Status
	currentMsg    string

	closed bool
}

type condition string

const (
	condReceive  condition = "receive"
	condWorker   condition = "worker"
	condProcess  condition = "process"
	condDelete   condition = "delete"
	condFinalize condition = "finalize"
	condPoison   condition = "poison"
)

type healthCondition struct {
	msg string
	at  time.Time
}

func newSQSHealth(reporter status.StatusReporter, log *logp.Logger) *sqsHealth {
	return &sqsHealth{
		reporter:          reporter,
		log:               log,
		conditions:        make(map[condition]healthCondition),
		recvFailThreshold: defaultRecvFailThreshold,
		procFailThreshold: defaultProcFailThreshold,
	}
}

// UpdateStatus satisfies status.StatusReporter. It is called by
// readSQSMessages (which reports receive-level Degraded/Running) and by
// lifecycle code (Starting, Configuring, Failed, Stopped).
//
// Lifecycle states pass through directly and reset runtime conditions.
// Running from readSQSMessages clears the receive condition.
// Degraded from readSQSMessages tracks consecutive receive failures.
func (h *sqsHealth) UpdateStatus(s status.Status, msg string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.closed {
		return
	}

	switch s {
	case status.Starting, status.Configuring, status.Stopping, status.Stopped, status.Failed:
		clear(h.conditions)
		h.consecutiveRecvFails = 0
		h.consecutiveProcFails = 0
		h.publish(s, msg)
		if s == status.Stopped || s == status.Failed {
			h.closed = true
		}
	case status.Running:
		h.consecutiveRecvFails = 0
		delete(h.conditions, condReceive)
		h.update()
	case status.Degraded:
		h.consecutiveRecvFails++
		if h.consecutiveRecvFails >= h.recvFailThreshold {
			h.conditions[condReceive] = healthCondition{msg: msg, at: time.Now()}
		}
		h.update()
	}
}

// SetWorkerError records a worker setup failure (for example pipeline
// connection error). This is a persistent condition cleared only by a
// successful lifecycle transition.
func (h *sqsHealth) SetWorkerError(err error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return
	}
	h.conditions[condWorker] = healthCondition{
		msg: fmt.Sprintf("The input could not start an SQS message processor: %s", err),
		at:  time.Now(),
	}
	h.update()
}

// SetProcessingError records an S3 processing failure. Errors caused by
// context cancellation (shutdown) are suppressed. Individual failures do
// not degrade, but sustained consecutive failures (above threshold)
// indicate a persistent problem like missing S3 permissions.
func (h *sqsHealth) SetProcessingError(err error) {
	if isShutdownErr(err) {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return
	}
	h.consecutiveProcFails++
	if h.consecutiveProcFails >= h.procFailThreshold {
		h.conditions[condProcess] = healthCondition{
			msg: fmt.Sprintf("The input cannot process S3 objects (%d consecutive failures): %s", h.consecutiveProcFails, err),
			at:  time.Now(),
		}
		h.update()
	}
}

// SetDeleteFailed records a failure to delete an SQS message after
// successful processing. This means the message will be reprocessed,
// causing duplicates.
func (h *sqsHealth) SetDeleteFailed(err error) {
	if isShutdownErr(err) {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return
	}
	h.conditions[condDelete] = healthCondition{
		msg: fmt.Sprintf("The input processed an SQS message but could not delete it from the queue. AWS may deliver the message again, which can result in duplicate events: %s", err),
		at:  time.Now(),
	}
	h.update()
}

// SetFinalizeFailed records a failure to finalize S3 objects after
// successful processing and deletion.
func (h *sqsHealth) SetFinalizeFailed(err error) {
	if isShutdownErr(err) {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return
	}
	h.conditions[condFinalize] = healthCondition{
		msg: fmt.Sprintf("The input could not copy an S3 object to the backup bucket, or could not delete the source object after backup. Check the backup bucket configuration and permissions: %s", err),
		at:  time.Now(),
	}
	h.update()
}

// RecordPoisonPill records that a message was deleted as a poison pill
// (non-retryable error after max receives). This is a data-loss signal.
func (h *sqsHealth) RecordPoisonPill(err error) {
	if isShutdownErr(err) {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return
	}
	h.conditions[condPoison] = healthCondition{
		msg: fmt.Sprintf("The input stopped retrying an SQS message after repeated processing failures and deleted it from the queue. Data from that notification may be missing: %s", err),
		at:  time.Now(),
	}
	h.update()
}

// ClearDisposition clears delete, finalize, poison-pill, and processing
// conditions after a successful end-to-end message completion (delete +
// finalize). Resets the consecutive processing failure counter.
func (h *sqsHealth) ClearDisposition() {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return
	}
	h.consecutiveProcFails = 0
	delete(h.conditions, condProcess)
	delete(h.conditions, condDelete)
	delete(h.conditions, condFinalize)
	delete(h.conditions, condPoison)
	h.update()
}

// update derives the aggregate status from active conditions and
// publishes it if changed. Must be called with h.mu held.
func (h *sqsHealth) update() {
	if len(h.conditions) == 0 {
		h.publish(status.Running, "Input is running")
		return
	}
	msg := h.pickMessage()
	h.publish(status.Degraded, msg)
}

// pickMessage returns the message from the highest-priority active condition.
// Priority order: worker, receive, process, delete, finalize, poison.
func (h *sqsHealth) pickMessage() string {
	for _, key := range []condition{condWorker, condReceive, condProcess, condDelete, condFinalize, condPoison} {
		if c, ok := h.conditions[key]; ok {
			return c.msg
		}
	}
	return "Input is degraded"
}

// publish sends the status to the underlying reporter if it differs from
// the last published state. This replaces StatusReporterHelper's dedup.
func (h *sqsHealth) publish(s status.Status, msg string) {
	if s == h.currentStatus && msg == h.currentMsg {
		return
	}
	h.currentStatus = s
	h.currentMsg = msg
	h.log.Debugw("health status", "status", s, "msg", msg)
	h.reporter.UpdateStatus(s, msg)
}

func isShutdownErr(err error) bool {
	return errors.Is(err, context.Canceled)
}
