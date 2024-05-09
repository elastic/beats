// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"context"
	"errors"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/aws/smithy-go"

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

func getRegionFromQueueURL(queueURL, endpoint string) string {
	// get region from queueURL
	// Example for sqs queue: https://sqs.us-east-1.amazonaws.com/12345678912/test-s3-logs
	// Example for vpce: https://vpce-test.sqs.us-east-1.vpce.amazonaws.com/12345678912/sqs-queue
	u, err := url.Parse(queueURL)
	if err != nil {
		return ""
	}

	// check for sqs queue url
	host := strings.SplitN(u.Host, ".", 3)
	if len(host) == 3 && host[0] == "sqs" {
		if host[2] == endpoint || (endpoint == "" && strings.HasPrefix(host[2], "amazonaws.")) {
			return host[1]
		}
	}

	// check for vpce url
	host = strings.SplitN(u.Host, ".", 5)
	if len(host) == 5 && host[1] == "sqs" {
		if host[4] == endpoint || (endpoint == "" && strings.HasPrefix(host[4], "amazonaws.")) {
			return host[2]
		}
	}

	return ""
}

// readSQSMessages reads up to the requested number of SQS messages via
// ReceiveMessage. It always returns at least one result unless the
// context expires
func readSQSMessages(
	ctx context.Context,
	log *logp.Logger,
	sqs sqsAPI,
	metrics *inputMetrics,
	count int,
) []types.Message {
	if count <= 0 {
		return nil
	}
	msgs, err := sqs.ReceiveMessage(ctx, count)
	for (err != nil || len(msgs) == 0) && ctx.Err() == nil {
		if err != nil {
			log.Warnw("SQS ReceiveMessage returned an error. Will retry after a short delay.", "error", err)
		}
		// Wait for the retry delay, but stop early if the context is cancelled.
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(sqsRetryDelay):
		}
		msgs, err = sqs.ReceiveMessage(ctx, count)
	}
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
