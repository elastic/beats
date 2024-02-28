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
	// This loop tries to keep the workers busy as much as possible while
	// honoring the max message cap as opposed to a simpler loop that receives
	// N messages, waits for them all to finish, then requests N more messages.
	var workerWg sync.WaitGroup
	endingChan := make(chan error, 1)
	processingChan := make(chan processingOutcome)

	go func(ctx context.Context) {
		// Wait for all workers to finish.
		for {
			select {
			case processOutcome, ok := <-processingChan:
				if !ok {
					if errors.Is(ctx.Err(), context.Canceled) {
						// A canceled context is a normal shutdown.
						close(endingChan)
						return
					}

					endingChan <- ctx.Err()
					return
				}

				go func(processOutcome processingOutcome) {
					// Wait for all events to be ACKed before proceeding.
					processOutcome.acker.Wait()

					// Stop keepalive routine before deleting visibility.
					processOutcome.keepaliveCancel()
					processOutcome.keepaliveWg.Wait()

					err := r.msgHandler.DeleteSQS(ctx, processOutcome.msg, processOutcome.receiveCount, processOutcome.processingErr, processOutcome.handles)
					if err != nil {
						r.log.Warnw("Failed deleting SQS message.",
							"error", err,
							"message_id", *processOutcome.msg.MessageId,
							"elapsed_time_ns", time.Since(processOutcome.start))
					}
				}(processOutcome)
			default:
			}
		}
	}(ctx)

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

				acker := awscommon.NewEventACKTracker(ctx)

				receiveCount, handles, keepaliveCancel, keepaliveWg, processingErr := r.msgHandler.ProcessSQS(ctx, &msg, client, acker)

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

	workerWg.Wait()
	close(processingChan)

	return <-endingChan
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
