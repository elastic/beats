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

	"github.com/elastic/beats/v9/libbeat/management/status"
	"github.com/elastic/elastic-agent-libs/logp"
)

type messageCountMonitor struct {
	sqs     sqsAPI
	metrics *inputMetrics
}

const (
	sqsAccessDeniedErrorCode       = "AccessDeniedException"
	sqsRetryDelay                  = 10 * time.Second
	sqsApproximateNumberOfMessages = "ApproximateNumberOfMessages"
)

var errBadQueueURL = errors.New("QueueURL is not in format: https://sqs.{REGION_ENDPOINT}.{ENDPOINT}/{ACCOUNT_NUMBER}/{QUEUE_NAME} or https://{VPC_ENDPOINT}.sqs.{REGION_ENDPOINT}.vpce.{ENDPOINT}/{ACCOUNT_NUMBER}/{QUEUE_NAME}")

func getRegionFromQueueURL(queueURL string) string {
	// get region from queueURL
	// Example for custom domain queue: https://sqs.us-east-1.abc.xyz/12345678912/test-s3-logs
	// Example for sqs queue: https://sqs.us-east-1.amazonaws.com/12345678912/test-s3-logs
	// Example for vpce: https://vpce-test.sqs.us-east-1.vpce.amazonaws.com/12345678912/sqs-queue
	// We use a simple heuristic that works for all essential cases:
	// - If queue hostname is sqs.X.*, return region X
	// - If queue hostname is X.sqs.Y.*, return region Y
	// Hosts that don't follow this convention need the input config to
	// specify a custom endpoint and an explicit region.
	u, err := url.Parse(queueURL)
	if err != nil {
		return ""
	}
	hostSplit := strings.SplitN(u.Hostname(), ".", 5)

	// check for sqs-style queue url
	if len(hostSplit) >= 4 && hostSplit[0] == "sqs" {
		return hostSplit[1]
	}

	// check for vpce-style url
	if len(hostSplit) == 5 && hostSplit[1] == "sqs" {
		return hostSplit[2]
	}

	return ""
}

// readSQSMessages reads up to the requested number of SQS messages via
// ReceiveMessage. It always returns at least one result unless the
// context expires
func readSQSMessages(
	ctx context.Context,
	log *logp.Logger,
	statusReporter status.StatusReporter,
	sqs sqsAPI,
	metrics *inputMetrics,
	count int,
	queueURL string,
) []types.Message {
	if count <= 0 {
		return nil
	}
	msgs, err := sqs.ReceiveMessage(ctx, count)
	for (err != nil || len(msgs) == 0) && ctx.Err() == nil {
		if err != nil {
			statusReporter.UpdateStatus(status.Degraded, fmt.Sprintf("Retryable SQS fetching error for queue '%s': %s", queueURL, err.Error()))
			log.Warnw("SQS ReceiveMessage returned an error. Will retry after a short delay.", "error", err)
		} else {
			// no auth error - input is running
			statusReporter.UpdateStatus(status.Running, "Input is running")
		}
		// Wait for the retry delay, but stop early if the context is cancelled.
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(sqsRetryDelay):
		}
		msgs, err = sqs.ReceiveMessage(ctx, count)
	}
	statusReporter.UpdateStatus(status.Running, "Input is running")
	log.Debugf("Received %v SQS messages.", len(msgs))
	metrics.sqsMessagesReceivedTotal.Add(uint64(len(msgs)))
	return msgs
}

func (mcm messageCountMonitor) run(ctx context.Context) {
	t := time.NewTicker(time.Minute)
	defer t.Stop()
	for {
		if err := mcm.updateMessageCount(ctx); isSQSAuthError(err) {
			// stop polling if auth error is encountered
			// Set it back to -1 because there is a permission error
			mcm.metrics.sqsMessagesWaiting.Set(int64(-1))
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

// updateMessageCount runs GetApproximateMessageCount and updates the
// sqsMessagesWaiting metric with the result.
// If there is an error, the metric is reinitialized to -1 and true is returned
func (mcm messageCountMonitor) updateMessageCount(ctx context.Context) error {
	count, err := mcm.getApproximateMessageCount(ctx)
	if err == nil {
		mcm.metrics.sqsMessagesWaiting.Set(int64(count))
	}
	return err
}

// Query the approximate message count for the queue via the SQS API.
func (mcm messageCountMonitor) getApproximateMessageCount(ctx context.Context) (int, error) {
	attributes, err := mcm.sqs.GetQueueAttributes(ctx, []types.QueueAttributeName{sqsApproximateNumberOfMessages})
	if err == nil {
		if c, found := attributes[sqsApproximateNumberOfMessages]; found {
			if messagesCount, err := strconv.Atoi(c); err == nil {
				return messagesCount, nil
			}
		}
	}
	return -1, err
}

func isSQSAuthError(err error) bool {
	var apiError smithy.APIError
	if errors.As(err, &apiError) {
		return apiError.ErrorCode() == sqsAccessDeniedErrorCode
	}
	return false
}
