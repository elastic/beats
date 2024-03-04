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

type processingOutcome struct {
	start           time.Time
	keepaliveWg     *sync.WaitGroup
	keepaliveCancel context.CancelFunc
	acker           *awscommon.EventACKTracker
	msg             *types.Message
	receiveCount    int
	handles         []s3ObjectHandler
	processingErr   error
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

	// At the end of the loop, after a shutdown or anything else, we still need to wait for the DeleteSQS goroutines to
	// complete, otherwise the messages will be sent back to the queue even if they should be deleted.
	var deletionWg sync.WaitGroup

	// We send to processingChan the outcome of each ProcessSQS call.
	// We don't want to buffer the processingChan, since it will prevent workers ProcessSQS goroutines to return.
	processingChan := make(chan processingOutcome)

	// We use deletionChan to throttle the number of DeleteSQS goroutines.
	// deletionChan := make(chan struct{}, 3200)

	go func() {
		for {
			outcome, ok := <-processingChan
			// processingChang is closed, no more outcomes to process, we can exit.
			if !ok {
				return
			}

			// A ProcessSQS goroutine has sent an outcome, let's process it asynchronously in order to handle SQS message deletion.
			go func(outcome processingOutcome) {
				// Mark deletion wait group as done when the goroutine is done.
				defer deletionWg.Done()

				r.log.Debugw("Waiting worker when deleting SQS message.",
					"message_id", *outcome.msg.MessageId,
					"elapsed_time_ns", time.Since(outcome.start))

				// We don't want to cap processingChan, since it will prevent workers ProcessSQS goroutines to return
				// and in flight message would be capped as well.
				// We want to cap number of goroutines for DeleteSQS
				// deletionChan <- struct{}{}

				r.log.Debugw("Waited worker when deleting SQS message.",
					"message_id", *outcome.msg.MessageId,
					"elapsed_time_ns", time.Since(outcome.start))

				r.log.Debugw("Waiting acker when deleting SQS message.",
					"message_id", *outcome.msg.MessageId,
					"elapsed_time_ns", time.Since(outcome.start))

				// Wait for all events to be ACKed before proceeding.
				outcome.acker.Wait()

				r.log.Debugw("Waited acker when deleting SQS message.",
					"message_id", *outcome.msg.MessageId,
					"elapsed_time_ns", time.Since(outcome.start))

				// Stop keepalive visibility routine before deleting.
				outcome.keepaliveCancel()
				outcome.keepaliveWg.Wait()

				err := r.msgHandler.DeleteSQS(outcome.msg, outcome.receiveCount, outcome.processingErr, outcome.handles)
				if err != nil {
					r.log.Warnw("Failed deleting SQS message.",
						"error", err,
						"message_id", *outcome.msg.MessageId,
						"elapsed_time_ns", time.Since(outcome.start))
				} else {
					r.log.Debugw("Success deleting SQS message.",
						"message_id", *outcome.msg.MessageId,
						"elapsed_time_ns", time.Since(outcome.start))
				}

				// <-deletionChan
			}(outcome)
		}
	}()

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

				// Create a pipeline client scoped to this goroutine.
				client, err := pipeline.ConnectWith(beat.ClientConfig{
					EventListener: awscommon.NewEventACKHandler(),
					Processing: beat.ProcessingConfig{
						// This input only produces events with basic types so normalization
						// is not required.
						EventNormalization: boolPtr(false),
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

				acker := awscommon.NewEventACKTracker()

				receiveCount, handles, keepaliveCancel, keepaliveWg, processingErr := r.msgHandler.ProcessSQS(ctx, &msg, client, acker)

				r.log.Debugw("Success processing SQS message.",
					"message_id", *msg.MessageId,
					"elapsed_time_ns", time.Since(start))

				// Add to deletion waiting group before sending to processingChan.
				deletionWg.Add(1)

				// Send the outcome to the processingChan so the deletion goroutine can delete the message.
				processingChan <- processingOutcome{
					start:           start,
					keepaliveWg:     keepaliveWg,
					keepaliveCancel: keepaliveCancel,
					acker:           acker,
					msg:             &msg,
					receiveCount:    receiveCount,
					handles:         handles,
					processingErr:   processingErr,
				}
			}(msg, time.Now())
		}
	}

	// Wait for all processing goroutines to finish.
	processingWg.Wait()

	// We need to close the processingChan to signal to the deletion goroutines that they should stop.
	close(processingChan)

	// Wait for all deletion goroutines to finish.
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
