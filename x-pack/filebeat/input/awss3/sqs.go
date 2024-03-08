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

	"github.com/elastic/beats/v7/libbeat/common/atomic"
	"github.com/elastic/go-concert/timed"

	"github.com/aws/aws-sdk-go-v2/service/sqs/types"

	"github.com/elastic/beats/v7/libbeat/beat"
	awscommon "github.com/elastic/beats/v7/x-pack/libbeat/common/aws"
	"github.com/elastic/elastic-agent-libs/logp"
)

const (
	sqsRetryDelay                  = 10 * time.Second
	sqsApproximateNumberOfMessages = "ApproximateNumberOfMessages"
)

type sqsReader struct {
	maxMessagesInflight int
	workerSem           *awscommon.Sem
	sqs                 sqsAPI
	pipeline            beat.Pipeline
	msgHandler          sqsProcessor
	log                 *logp.Logger
	metrics             *inputMetrics
}

func newSQSReader(log *logp.Logger, metrics *inputMetrics, sqs sqsAPI, maxMessagesInflight int, msgHandler sqsProcessor, pipeline beat.Pipeline) *sqsReader {
	if metrics == nil {
		// Metrics are optional. Initialize a stub.
		metrics = newInputMetrics("", nil, 0)
	}
	return &sqsReader{
		maxMessagesInflight: maxMessagesInflight,
		workerSem:           awscommon.NewSem(maxMessagesInflight),
		sqs:                 sqs,
		pipeline:            pipeline,
		msgHandler:          msgHandler,
		log:                 log,
		metrics:             metrics,
	}
}

type processingData struct {
	msg   types.Message
	start time.Time
}

func (r *sqsReader) Receive(ctx context.Context) error {
	workersWg := new(sync.WaitGroup)
	workersWg.Add(r.maxMessagesInflight)
	workersChan := make(chan processingData, r.maxMessagesInflight)

	deletionWg := new(sync.WaitGroup)
	deletionWaiter := atomic.NewBool(true)

	var clientsMutex sync.Mutex
	clients := make(map[uint64]beat.Client, r.maxMessagesInflight)

	// Start a fixed amount of goroutines that will process all the SQS messages sent to the workersChan asynchronously.
	for i := 0; i < r.maxMessagesInflight; i++ {
		id := r.metrics.beginSQSWorker()

		// Create a pipeline client scoped to this goroutine.
		client, err := r.pipeline.ConnectWith(beat.ClientConfig{
			EventListener: NewEventACKHandler(),
			Processing: beat.ProcessingConfig{
				// This input only produces events with basic types so normalization
				// is not required.
				EventNormalization: boolPtr(false),
			},
		})

		clientsMutex.Lock()
		clients[id] = client
		clientsMutex.Unlock()

		if err != nil {
			r.log.Warnw("Failed setting up worker.",
				"worker_id", id,
				"error", err)

			r.metrics.endSQSWorker(id)
			workersWg.Done()
		}

		go func(id uint64, client beat.Client) {
			defer func() {
				// Mark processing wait group as done.
				r.metrics.endSQSWorker(id)
				workersWg.Done()
			}()
			for {
				incomingData, ok := <-workersChan
				if !ok {
					return
				}

				deletionWg.Add(1)
				deletionWaiter.Swap(false)

				msg := incomingData.msg
				start := incomingData.start

				r.log.Debugw("Going to process SQS message.",
					"worker_id", id,
					"message_id", *msg.MessageId,
					"elapsed_time_ns", time.Since(start))

				acker := NewEventACKTracker(ctx, deletionWg)

				err = r.msgHandler.ProcessSQS(ctx, &msg, client, acker, start)
				if err != nil {
					r.log.Warnw("Failed processing SQS message.",
						"worker_id", id,
						"error", err,
						"message_id", *msg.MessageId,
						"elapsed_time_ns", time.Since(start))

					return
				}

				r.log.Debugw("Success processing SQS message.",
					"worker_id", id,
					"message_id", msg.MessageId,
					"elapsed_time_ns", time.Since(start))
			}
		}(id, client)
	}

	// The loop tries to keep a fixed amount of goroutines that process SQS message busy as much as possible while
	// honoring the max message cap as opposed to a simpler loop that receives N messages, waits for them all to finish
	// sending events to the queue, then requests N more messages.
	for ctx.Err() == nil {
		// Receive (at most) as many SQS messages as there are workers.
		msgs, err := r.sqs.ReceiveMessage(ctx, r.maxMessagesInflight)
		if err != nil {
			if ctx.Err() == nil {
				r.log.Warnw("SQS ReceiveMessage returned an error. Will retry after a short delay.", "error", err)

				// Throttle retries.
				_ = timed.Wait(ctx, sqsRetryDelay)
			}
			continue
		}

		r.log.Debugf("Received %v SQS messages.", len(msgs))
		r.metrics.sqsMessagesReceivedTotal.Add(uint64(len(msgs)))

		for _, msg := range msgs {
			workersChan <- processingData{msg: msg, start: time.Now()}
		}
	}

	// Let's stop the workers
	close(workersChan)

	// Wait for all processing to happen.
	workersWg.Wait()

	// Wait for all deletion to happen.
	if r.metrics.sqsMessagesReceivedTotal.Get() > 0 {
		for deletionWaiter.Load() {
			_ = timed.Wait(ctx, 500*time.Millisecond)
		}
	}

	deletionWg.Wait()

	closeClients(clients)

	if errors.Is(ctx.Err(), context.Canceled) {
		// A canceled context is a normal shutdown.
		return nil
	}

	return ctx.Err()
}

func closeClients(clients map[uint64]beat.Client) {
	for _, client := range clients {
		client.Close()
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
