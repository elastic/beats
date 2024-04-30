// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/aws/smithy-go"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
)

const (
	sqsAccessDeniedErrorCode       = "AccessDeniedException"
	sqsRetryDelay                  = 10 * time.Second
	sqsApproximateNumberOfMessages = "ApproximateNumberOfMessages"
)

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

func pollSqsWaitingMetric(canceler v2.Canceler, sqs sqsAPI, metrics *inputMetrics) {
	t := time.NewTicker(time.Minute)
	defer t.Stop()
	for {
		if err := updateMessageCount(canceler, sqs, metrics); isSQSAuthError(err) {
			// stop polling if auth error is encountered
			// Set it back to -1 because there is a permission error
			metrics.sqsMessagesWaiting.Set(int64(-1))
			return
		}
		select {
		case <-canceler.Done():
			return
		case <-t.C:
		}
	}
}

// updateMessageCount runs GetApproximateMessageCount for the given context and updates the receiver metric with the count returning false on no error
// If there is an error, the metric is reinitialized to -1 and true is returned
func updateMessageCount(canceler v2.Canceler, sqs sqsAPI, metrics *inputMetrics) error {
	count, err := getApproximateMessageCount(canceler, sqs)
	if err == nil {
		metrics.sqsMessagesWaiting.Set(int64(count))
	}
	return err
}

func getApproximateMessageCount(canceler v2.Canceler, sqs sqsAPI) (int, error) {
	ctx := v2.GoContextFromCanceler(canceler)
	attributes, err := sqs.GetQueueAttributes(ctx, []types.QueueAttributeName{sqsApproximateNumberOfMessages})
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
