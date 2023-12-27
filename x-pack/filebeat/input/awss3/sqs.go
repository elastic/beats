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

	awscommon "github.com/elastic/beats/v7/x-pack/libbeat/common/aws"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/go-concert/timed"
)

const (
	sqsRetryDelay                  = 10 * time.Second
	sqsApproximateNumberOfMessages = "ApproximateNumberOfMessages"
)

type sqsReader struct {
	maxMessagesInflight int
	workerSem           *awscommon.Sem
	sqs                 sqsAPI
	msgHandler          sqsProcessor
	log                 *logp.Logger
	metrics             *inputMetrics
}

func newSQSReader(log *logp.Logger, metrics *inputMetrics, sqs sqsAPI, maxMessagesInflight int, msgHandler sqsProcessor) *sqsReader {
	if metrics == nil {
		// Metrics are optional. Initialize a stub.
		metrics = newInputMetrics("", nil, 0)
	}
	return &sqsReader{
		maxMessagesInflight: maxMessagesInflight,
		workerSem:           awscommon.NewSem(maxMessagesInflight),
		sqs:                 sqs,
		msgHandler:          msgHandler,
		log:                 log,
		metrics:             metrics,
	}
}

func (r *sqsReader) Receive(ctx context.Context) error {
	// This loop tries to keep the workers busy as much as possible while
	// honoring the max message cap as opposed to a simpler loop that receives
	// N messages, waits for them all to finish, then requests N more messages.
	var workerWg sync.WaitGroup
	for ctx.Err() == nil {
		// Determine how many SQS workers are available.
		workers, err := r.workerSem.AcquireContext(r.maxMessagesInflight, ctx)
		if err != nil {
			break
		}

		// Receive (at most) as many SQS messages as there are workers.
		msgs, err := r.sqs.ReceiveMessage(ctx, workers)
		if err != nil {
			r.workerSem.Release(workers)

			if ctx.Err() == nil {
				r.log.Warnw("SQS ReceiveMessage returned an error. Will retry after a short delay.", "error", err)

				// Throttle retries.
				_ = timed.Wait(ctx, sqsRetryDelay)
			}
			continue
		}

		// Release unused workers.
		r.workerSem.Release(workers - len(msgs))

		// Process each SQS message asynchronously with a goroutine.
		r.log.Debugf("Received %v SQS messages.", len(msgs))
		r.metrics.sqsMessagesReceivedTotal.Add(uint64(len(msgs)))
		workerWg.Add(len(msgs))

		for _, msg := range msgs {
			go func(msg types.Message, start time.Time) {
				id := r.metrics.beginSQSWorker()
				defer func() {
					r.metrics.endSQSWorker(id)
					workerWg.Done()
					r.workerSem.Release(1)
				}()

				if err := r.msgHandler.ProcessSQS(ctx, &msg); err != nil {
					r.log.Warnw("Failed processing SQS message.",
						"error", err,
						"message_id", *msg.MessageId,
						"elapsed_time_ns", time.Since(start))
				}
			}(msg, time.Now())
		}
	}

	// Wait for all workers to finish.
	workerWg.Wait()

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
