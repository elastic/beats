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

	"github.com/elastic/beats/v7/libbeat/beat"
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

func (r *sqsReader) Receive(ctx context.Context, pipeline beat.Pipeline) error {
	// The loop tries to keep the ProcessSQS workers busy as much as possible while
	// honoring the max message cap as opposed to a simpler loop that receives
	// N messages, waits for them all to finish sending events to the queue, then requests N more messages.
	var processingWg sync.WaitGroup

	deletionWg := new(sync.WaitGroup)

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

		r.log.Debugf("Received %v SQS messages.", len(msgs))
		r.metrics.sqsMessagesReceivedTotal.Add(uint64(len(msgs)))

		// Add to processing wait group to wait for all messages to be processed.
		processingWg.Add(len(msgs))
		deletionWg.Add(len(msgs))

		for _, msg := range msgs {
			// Process each SQS message asynchronously with a goroutine.
			go func(msg types.Message, start time.Time) {
				id := r.metrics.beginSQSWorker()
				defer func() {
					// Mark processing wait group as done.
					r.metrics.endSQSWorker(id)
					processingWg.Done()
					r.workerSem.Release(1)
				}()

				acker := NewEventACKTracker(ctx, deletionWg)

				// Create a pipeline client scoped to this goroutine.
				client, err := pipeline.ConnectWith(beat.ClientConfig{
					EventListener: NewEventACKHandler(),
					Processing: beat.ProcessingConfig{
						// This input only produces events with basic types so normalization
						// is not required.
						EventNormalization: boolPtr(false),
						Private:            acker,
					},
				})

				if err != nil {
					r.log.Warnw("Failed processing SQS message.",
						"error", err,
						"message_id", *msg.MessageId,
						"elapsed_time_ns", time.Since(start))

					return
				}

				defer client.Close()

				r.log.Debugw("Going to process SQS message.",
					"message_id", *msg.MessageId,
					"elapsed_time_ns", time.Since(start))

				err = r.msgHandler.ProcessSQS(ctx, &msg, client, acker, start)
				if err != nil {
					r.log.Warnw("Failed processing SQS message.",
						"error", err,
						"message_id", *msg.MessageId,
						"elapsed_time_ns", time.Since(start))

				}

				r.log.Debugw("Success processing SQS message.",
					"message_id", *msg.MessageId,
					"elapsed_time_ns", time.Since(start))
			}(msg, time.Now())
		}
	}

	// Wait for all processing goroutines to finish.
	processingWg.Wait()

	// Wait for all deletion to happen.
	deletionWg.Wait()

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
