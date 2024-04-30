// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/aws/smithy-go"

	"github.com/elastic/elastic-agent-libs/logp"
)

const (
	sqsAccessDeniedErrorCode       = "AccessDeniedException"
	sqsRetryDelay                  = 10 * time.Second
	sqsApproximateNumberOfMessages = "ApproximateNumberOfMessages"
)

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

var errBadQueueURL = errors.New("QueueURL is not in format: https://sqs.{REGION_ENDPOINT}.{ENDPOINT}/{ACCOUNT_NUMBER}/{QUEUE_NAME} or https://{VPC_ENDPOINT}.sqs.{REGION_ENDPOINT}.vpce.{ENDPOINT}/{ACCOUNT_NUMBER}/{QUEUE_NAME}")

func getRegionFromQueueURL(queueURL, endpoint string) (string, error) {
	// get region from queueURL
	// Example for sqs queue: https://sqs.us-east-1.amazonaws.com/12345678912/test-s3-logs
	// Example for vpce: https://vpce-test.sqs.us-east-1.vpce.amazonaws.com/12345678912/sqs-queue
	u, err := url.Parse(queueURL)
	if err != nil {
		return "", fmt.Errorf(queueURL + " is not a valid URL")
	}
	if (u.Scheme == "https" || u.Scheme == "http") && u.Host != "" {
		queueHostSplit := strings.SplitN(u.Host, ".", 3)
		// check for sqs queue url
		if len(queueHostSplit) == 3 && queueHostSplit[0] == "sqs" {
			if queueHostSplit[2] == endpoint || (endpoint == "" && strings.HasPrefix(queueHostSplit[2], "amazonaws.")) {
				return queueHostSplit[1], nil
			}
		}

		// check for vpce url
		queueHostSplitVPC := strings.SplitN(u.Host, ".", 5)
		if len(queueHostSplitVPC) == 5 && queueHostSplitVPC[1] == "sqs" {
			if queueHostSplitVPC[4] == endpoint || (endpoint == "" && strings.HasPrefix(queueHostSplitVPC[4], "amazonaws.")) {
				return queueHostSplitVPC[2], nil
			}
		}
	}
	return "", errBadQueueURL
}

func pollSqsWaitingMetric(ctx context.Context, receiver *sqsReader) {
	t := time.NewTicker(time.Minute)
	defer t.Stop()
	for {
		if err := updateMessageCount(receiver, ctx); isSQSAuthError(err) {
			// stop polling if auth error is encountered
			// Set it back to -1 because there is a permission error
			receiver.metrics.sqsMessagesWaiting.Set(int64(-1))
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

// updateMessageCount runs GetApproximateMessageCount for the given context and updates the receiver metric with the count returning false on no error
// If there is an error, the metric is reinitialized to -1 and true is returned
func updateMessageCount(receiver *sqsReader, ctx context.Context) error {
	count, err := receiver.GetApproximateMessageCount(ctx)
	if err == nil {
		receiver.metrics.sqsMessagesWaiting.Set(int64(count))
	}
	return err
}

func isSQSAuthError(err error) bool {
	var apiError smithy.APIError
	if errors.As(err, &apiError) {
		return apiError.ErrorCode() == sqsAccessDeniedErrorCode
	}
	return false
}
