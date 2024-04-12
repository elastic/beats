// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"context"
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
	}
}

// The main loop of the reader, that fetches messages from SQS
// and forwards them to workers via workChan.
func (r *sqsReader) Receive(ctx context.Context) {
	r.startWorkers(ctx)
	r.readerLoop(ctx)

	// Close the work channel to signal to the workers that we're done,
	// then wait for them to finish.
	close(r.workChan)
	r.workerWg.Wait()
}

func (r *sqsReader) readerLoop(ctx context.Context) {
	for ctx.Err() == nil {
		msgs := r.readMessages(ctx)

		for _, msg := range msgs {
			select {
			case <-ctx.Done():
			case r.workChan <- msg:
			}
		}
	}
}

func (r *sqsReader) workerLoop(ctx context.Context) {
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
	}
}

func (r *sqsReader) readMessages(ctx context.Context) []types.Message {
	// We try to read enough messages to bring activeMessages up to the
	// total worker count (plus one, to unblock us when workers are ready
	// for more messages)
	readCount := r.maxMessagesInflight + 1 - r.activeMessages.Load()
	if readCount <= 0 {
		return nil
	}
	msgs, err := r.sqs.ReceiveMessage(ctx, readCount)
	for err != nil && ctx.Err() == nil {
		r.log.Warnw("SQS ReceiveMessage returned an error. Will retry after a short delay.", "error", err)
		// Wait for the retry delay, but stop early if the context is cancelled.
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(sqsRetryDelay):
		}
		msgs, err = r.sqs.ReceiveMessage(ctx, readCount)
	}
	r.activeMessages.Add(len(msgs))
	r.log.Debugf("Received %v SQS messages.", len(msgs))
	r.metrics.sqsMessagesReceivedTotal.Add(uint64(len(msgs)))
	return msgs
}

func (r *sqsReader) startWorkers(ctx context.Context) {
	// Start the worker goroutines that will process messages from workChan
	// until the input shuts down.
	for i := 0; i < r.maxMessagesInflight; i++ {
		r.workerWg.Add(1)
		go func() {
			defer r.workerWg.Done()
			r.workerLoop(ctx)
		}()
	}
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
