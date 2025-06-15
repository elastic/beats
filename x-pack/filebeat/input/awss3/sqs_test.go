// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/gofrs/uuid/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

const testTimeout = 10 * time.Second

var (
	errFakeConnectivityFailure = errors.New("fake connectivity failure")
	errFakeGetAttributeFailute = errors.New("something went wrong")
)

func TestSQSReceiver(t *testing.T) {
	err := logp.TestingSetup()
	require.NoError(t, err)

	const workerCount = 5

	t.Run("ReceiveMessage success", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()

		ctrl, ctx := gomock.WithContext(ctx, t)
		defer ctrl.Finish()
		mockSQS := NewMockSQSAPI(ctrl)
		mockMsgHandler := NewMockSQSProcessor(ctrl)
		msg, err := newSQSMessage(newS3Event("log.json"))
		require.NoError(t, err)

		// Initial ReceiveMessage call returns the mock message.
		mockSQS.EXPECT().
			ReceiveMessage(gomock.Any(), gomock.Any()).
			Times(1).
			DoAndReturn(func(_ context.Context, _ int) ([]types.Message, error) {
				// Return single message.
				return []types.Message{msg}, nil
			})

		// Follow up ReceiveMessages returns empty message and could be called any times till validation is completed.
		mockSQS.EXPECT().
			ReceiveMessage(gomock.Any(), gomock.Any()).
			AnyTimes().
			DoAndReturn(func(_ context.Context, _ int) ([]types.Message, error) {
				return nil, nil
			})

		mockSQS.EXPECT().
			GetQueueAttributes(gomock.Any(), gomock.Eq([]types.QueueAttributeName{sqsApproximateNumberOfMessages})).
			DoAndReturn(func(_ context.Context, _ []types.QueueAttributeName) (map[string]string, error) {
				return map[string]string{sqsApproximateNumberOfMessages: "10000"}, nil
			}).AnyTimes()

		// Deletion happens when message is fully processed. Cancel the context and mark for exit.
		mockSQS.EXPECT().
			DeleteMessage(gomock.Any(), gomock.Any()).Times(1).Do(
			func(_ context.Context, _ *types.Message) {
				cancel()
			})

		logger := logp.NewLogger(inputName)

		// Expect the one message returned to have been processed.
		mockMsgHandler.EXPECT().
			ProcessSQS(gomock.Any(), gomock.Eq(&msg), gomock.Any()).
			Times(1).
			DoAndReturn(
				func(_ context.Context, _ *types.Message, _ func(e beat.Event)) sqsProcessingResult {
					return sqsProcessingResult{
						keepaliveCancel: func() {},
						processor: &sqsS3EventProcessor{
							log: logger,
							sqs: mockSQS,
						},
					}
				})

		// Execute sqsReader and verify calls/state.
		sqsReader := newSQSReaderInput(config{NumberOfWorkers: workerCount}, aws.Config{})
		sqsReader.log = logger
		sqsReader.sqs = mockSQS
		sqsReader.metrics = newInputMetrics(v2.Context{MetricsRegistry: monitoring.NewRegistry()}, 0)
		sqsReader.pipeline = &fakePipeline{}
		sqsReader.msgHandler = mockMsgHandler
		sqsReader.run(ctx)

		select {
		case <-ctx.Done():
		case <-time.After(time.Second):
			require.Fail(t, "Never observed SQS DeleteMessage call")
		}
	})

	t.Run("retry after ReceiveMessage error", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), sqsRetryDelay+testTimeout)
		defer cancel()

		ctrl, ctx := gomock.WithContext(ctx, t)
		defer ctrl.Finish()
		mockSQS := NewMockSQSAPI(ctrl)
		mockMsgHandler := NewMockSQSProcessor(ctrl)

		gomock.InOrder(
			// Initial ReceiveMessage gets an error.
			mockSQS.EXPECT().
				ReceiveMessage(gomock.Any(), gomock.Any()).
				Times(1).
				DoAndReturn(func(_ context.Context, _ int) ([]types.Message, error) {
					return nil, errFakeConnectivityFailure
				}),
			// After waiting for sqsRetryDelay, it retries.
			mockSQS.EXPECT().
				ReceiveMessage(gomock.Any(), gomock.Any()).
				Times(1).
				DoAndReturn(func(_ context.Context, _ int) ([]types.Message, error) {
					cancel()
					return nil, nil
				}),
		)
		mockSQS.EXPECT().
			GetQueueAttributes(gomock.Any(), gomock.Eq([]types.QueueAttributeName{sqsApproximateNumberOfMessages})).
			DoAndReturn(func(_ context.Context, _ []types.QueueAttributeName) (map[string]string, error) {
				return map[string]string{sqsApproximateNumberOfMessages: "10000"}, nil
			}).AnyTimes()

		// Execute SQSReader and verify calls/state.
		sqsReader := newSQSReaderInput(config{NumberOfWorkers: workerCount}, aws.Config{})
		sqsReader.log = logp.NewLogger(inputName)
		sqsReader.sqs = mockSQS
		sqsReader.msgHandler = mockMsgHandler
		sqsReader.metrics = newInputMetrics(v2.Context{MetricsRegistry: monitoring.NewRegistry()}, 0)
		sqsReader.pipeline = &fakePipeline{}
		sqsReader.run(ctx)
	})
}

func TestGetApproximateMessageCount(t *testing.T) {
	err := logp.TestingSetup()
	require.NoError(t, err)

	const count = 500
	attrName := []types.QueueAttributeName{sqsApproximateNumberOfMessages}
	attr := map[string]string{"ApproximateNumberOfMessages": "500"}

	t.Run("getApproximateMessageCount success", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()

		ctrl, ctx := gomock.WithContext(ctx, t)
		defer ctrl.Finish()
		mockAPI := NewMockSQSAPI(ctrl)

		gomock.InOrder(
			mockAPI.EXPECT().
				GetQueueAttributes(gomock.Any(), gomock.Eq(attrName)).
				Times(1).
				DoAndReturn(func(_ context.Context, _ []types.QueueAttributeName) (map[string]string, error) {
					return attr, nil
				}),
		)

		receivedCount, err :=
			messageCountMonitor{sqs: mockAPI}.getApproximateMessageCount(ctx)
		assert.Equal(t, count, receivedCount)
		assert.NoError(t, err)
	})

	t.Run("GetApproximateMessageCount error", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()

		ctrl, ctx := gomock.WithContext(ctx, t)
		defer ctrl.Finish()

		mockAPI := NewMockSQSAPI(ctrl)

		gomock.InOrder(
			mockAPI.EXPECT().
				GetQueueAttributes(gomock.Any(), gomock.Eq(attrName)).
				Times(1).
				DoAndReturn(func(_ context.Context, _ []types.QueueAttributeName) (map[string]string, error) {
					return nil, errFakeGetAttributeFailute
				}),
		)

		receivedCount, err := messageCountMonitor{sqs: mockAPI}.getApproximateMessageCount(ctx)
		assert.Equal(t, -1, receivedCount)
		assert.NotNil(t, err)
	})
}

func newSQSMessage(events ...s3EventV2) (types.Message, error) {
	body, err := json.Marshal(s3EventsV2{Records: events})
	if err != nil {
		return types.Message{}, err
	}

	hash := sha256.Sum256(body)
	id, _ := uuid.FromBytes(hash[:16])
	messageID := id.String()
	receipt := "receipt-" + messageID
	bodyStr := string(body)

	return types.Message{
		Body:          &bodyStr,
		MessageId:     &messageID,
		ReceiptHandle: &receipt,
	}, nil
}

func newSNSSQSMessage() (types.Message, error) {
	body, err := json.Marshal(s3EventsV2{
		TopicArn: "arn:aws:sns:us-east-1:1234:sns-topic",
		Message:  "{\"Records\":[{\"eventSource\":\"aws:s3\",\"awsRegion\":\"us-east-1\",\"eventName\":\"ObjectCreated:Put\",\"s3\":{\"configurationId\":\"sns-notification-vpc-flow-logs\",\"bucket\":{\"name\":\"vpc-flow-logs-ks\",\"arn\":\"arn:aws:s3:::vpc-flow-logs-ks\"},\"object\":{\"key\":\"test-object-key\"}}}]}",
	})
	if err != nil {
		return types.Message{}, err
	}

	hash := sha256.Sum256(body)
	id, _ := uuid.FromBytes(hash[:16])
	messageID := id.String()
	receipt := "receipt-" + messageID
	bodyStr := string(body)

	return types.Message{
		Body:          &bodyStr,
		MessageId:     &messageID,
		ReceiptHandle: &receipt,
	}, nil
}

func newEventBridgeSQSMessage() (types.Message, error) {
	body := []byte("{ \"version\": \"0\", \"id\": \"f17994c0-7cb9-ee01-79fd-ae46df89e3a4\", \"detail-type\": \"Object Created\", \"source\": \"aws.s3\", \"account\": \"952856826737\", \"time\": \"2024-06-24T08:31:26Z\", \"region\": \"eu-west-1\", \"resources\": [\"arn:aws:s3:::vpc-flow-logs-ks\" ], \"detail\": {\"version\": \"0\",\"bucket\": { \"name\": \"vpc-flow-logs-ks\"},\"object\": { \"key\": \"test-object-key\", \"size\": 744, \"etag\": \"2ba6b152f13c75a9155b87b5b072963c\", \"version-id\": \"uoW5awQhqux4rhi4Nuh6il967FzQlsHJ\", \"sequencer\": \"0066792EC46EC0B037\"},\"request-id\": \"Q93BVWXD5G6FKWC2\",\"requester\": \"516635408544\",\"source-ip-address\": \"10.0.27.95\",\"reason\": \"PutObject\" }}")
	hash := sha256.Sum256(body)
	id, _ := uuid.FromBytes(hash[:16])
	messageID := id.String()
	receipt := "receipt-" + messageID
	bodyStr := string(body)

	return types.Message{
		Body:          &bodyStr,
		MessageId:     &messageID,
		ReceiptHandle: &receipt,
	}, nil
}

func newS3Event(key string) s3EventV2 {
	record := s3EventV2{
		AWSRegion:   "us-east-1",
		EventSource: "aws:s3",
		EventName:   "ObjectCreated:Put",
		Provider:    "aws",
	}
	record.S3.Bucket.Name = "foo"
	record.S3.Bucket.ARN = "arn:aws:s3:::foo"
	record.S3.Object.Key = key
	return record
}

func TestSQSReaderLoop(t *testing.T) {

}

func TestSQSWorkerLoop(t *testing.T) {

}

func TestCancelWithGrace(t *testing.T) {
	// TODO: Rewrite this to use testing/synctest when it is available without
	// GOEXPERIMENT=synctest. See https://go.dev/blog/synctest.

	const (
		wait    = time.Second
		tooLong = time.Second
		tol     = 100 * time.Millisecond
	)
	parentCtx, parentCancel := context.WithCancel(context.Background())
	childCtx, childCancel := cancelWithGrace(parentCtx, wait)
	defer childCancel()

	var parentCancelled, childCancelled time.Time
	parentCancel()
	select {
	case <-time.After(tooLong):
		t.Fatal("parent context failed to cancel within timeout")
	case <-parentCtx.Done():
		parentCancelled = time.Now()
	}
	select {
	case <-time.After(wait + tooLong):
		t.Fatal("child context failed to cancel within timeout after wait time")
	case <-childCtx.Done():
		childCancelled = time.Now()
	}
	waited := childCancelled.Sub(parentCancelled)
	if waited.Round(tol) != wait {
		t.Errorf("unexpected wait time between parent and child cancellation: got=%v want=%v", waited, wait)
	}
}
