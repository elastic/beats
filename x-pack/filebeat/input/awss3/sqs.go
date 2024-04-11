// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/sqs/types"

	"github.com/elastic/beats/v7/libbeat/common/atomic"
	"github.com/elastic/elastic-agent-libs/logp"
)

const (
	sqsRetryDelay                  = 10 * time.Second
	sqsApproximateNumberOfMessages = "ApproximateNumberOfMessages"
)

type sqsReader struct {
	maxMessagesInflight int
	activeMessages      atomic.Int
	sqs                 sqsAPI
	msgHandler          sqsProcessor
	log                 *logp.Logger
	metrics             *inputMetrics

	// The main loop sends incoming messages to workChan, and the worker
	// goroutines read from it.
	workChan chan types.Message

	// workerWg is used to wait on worker goroutines during shutdown
	workerWg sync.WaitGroup

	// This channel is used by wakeUpMainLoop() to signal to the main
	// loop that a worker is ready for more data
	wakeUpChan chan struct{}

	// If retryTimer is set, there was an error receiving SQS messages,
	// and the run loop will not try again until the timer expires.
	retryTimer *time.Timer
}

func newSQSReader(log *logp.Logger, metrics *inputMetrics, sqs sqsAPI, maxMessagesInflight int, msgHandler sqsProcessor) *sqsReader {
	if metrics == nil {
		// Metrics are optional. Initialize a stub.
		metrics = newInputMetrics("", nil, 0)
	}
	return &sqsReader{
		maxMessagesInflight: maxMessagesInflight,
		sqs:                 sqs,
		msgHandler:          msgHandler,
		log:                 log,
		metrics:             metrics,
		workChan:            make(chan types.Message),

		// wakeUpChan is buffered so we can always trigger it without blocking,
		// even if the main loop is in the middle of other work
		wakeUpChan: make(chan struct{}, 1),
	}
}

func (r *sqsReader) wakeUpMainLoop() {
	select {
	case r.wakeUpChan <- struct{}{}:
	default:
	}
}

func (r *sqsReader) sqsWorkerLoop(ctx context.Context) {
	for msg := range r.workChan {
		start := time.Now()

		id := r.metrics.beginSQSWorker()
		if err := r.msgHandler.ProcessSQS(ctx, &msg); err != nil {
			r.log.Warnw("Failed processing SQS message.",
				"error", err,
				"message_id", *msg.MessageId,
				"elapsed_time_ns", time.Since(start))
		}
		r.metrics.endSQSWorker(id)
		r.activeMessages.Dec()
		// Notify the main loop that we're ready for more data, in case it's asleep
		r.wakeUpMainLoop()
	}
}

func (r *sqsReader) getMessageBatch(ctx context.Context) []types.Message {
	// We read enough messages to bring activeMessages up to the total
	// worker count
	receiveCount := r.maxMessagesInflight - r.activeMessages.Load()
	if receiveCount > 0 {
		msgs, err := r.sqs.ReceiveMessage(ctx, receiveCount)
		if err != nil && ctx.Err() == nil {
			r.log.Warnw("SQS ReceiveMessage returned an error. Will retry after a short delay.", "error", err)
			r.retryTimer = time.NewTimer(sqsRetryDelay)
		}
		r.activeMessages.Add(len(msgs))
		r.log.Debugf("Received %v SQS messages.", len(msgs))
		r.metrics.sqsMessagesReceivedTotal.Add(uint64(len(msgs)))
		return msgs
	}
	return nil
}

func (r *sqsReader) startWorkers(ctx context.Context) {
	// Start the worker goroutines that will process messages from workChan
	// until the input shuts down.
	for i := 0; i < r.maxMessagesInflight; i++ {
		r.workerWg.Add(1)
		go func() {
			defer r.workerWg.Done()
			r.sqsWorkerLoop(ctx)
		}()
	}
}

func (r *sqsReader) Receive(ctx context.Context) error {
	var msgs []types.Message
	for ctx.Err() == nil {
		// If we don't have any messages, and we aren't in a retry delay,
		// try to read some
		if len(msgs) == 0 && r.retryTimer == nil {
			msgs = r.getMessageBatch(ctx)
		}

		// Unblock the local work channel only if there are messages to send
		var workChan chan types.Message
		var nextMessage types.Message
		if len(msgs) > 0 {
			workChan = r.workChan
			nextMessage = msgs[0]
		}

		// Unblock the retry channel only if there's an active retry timer
		var retryChan <-chan time.Time
		if r.retryTimer != nil {
			retryChan = r.retryTimer.C
		}

		select {
		case <-ctx.Done():
		case workChan <- nextMessage:
			msgs = msgs[1:]
		case <-retryChan:
			// The retry interval has elapsed, clear the timer so we can request
			// new messages again
			r.retryTimer = nil
		case <-r.wakeUpChan:
			// No need to do anything, this is just to unblock us when a worker is
			// ready for more data
		}
	}

	// Close the work channel to signal to the workers that we're done
	close(r.workChan)

	// Wait for all workers to finish.
	r.workerWg.Wait()

	if errors.Is(ctx.Err(), context.Canceled) {
		// A canceled context is a normal shutdown.
		return nil
	}
	return ctx.Err()
}

func (r *sqsReader) GetApproximateMessageCount(ctx context.Context) (int, error) {
	attributes, err := r.sqs.GetQueueAttributes(ctx, []types.QueueAttributeName{sqsApproximateNumberOfMessages})
	if err == nil {
		if c, found := attributes[sqsApproximateNumberOfMessages]; found {
			if messagesCount, err := strconv.Atoi(c); err == nil {
				return messagesCount, nil
			}
		}
	}
	return -1, err
}
