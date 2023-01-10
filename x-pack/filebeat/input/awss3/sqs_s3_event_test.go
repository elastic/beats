// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/aws/smithy-go"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	awscommon "github.com/elastic/beats/v7/x-pack/libbeat/common/aws"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/go-concert/timed"
)

func TestSQSS3EventProcessor(t *testing.T) {
	require.NoError(t, logp.TestingSetup())

	msg := newSQSMessage(newS3Event("log.json"))

	t.Run("s3 events are processed and sqs msg is deleted", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()

		ctrl, ctx := gomock.WithContext(ctx, t)
		defer ctrl.Finish()
		mockAPI := NewMockSQSAPI(ctrl)
		mockS3HandlerFactory := NewMockS3ObjectHandlerFactory(ctrl)
		mockClient := NewMockBeatClient(ctrl)
		mockBeatPipeline := NewMockBeatPipeline(ctrl)

		gomock.InOrder(
			mockBeatPipeline.EXPECT().ConnectWith(gomock.Any()).Return(mockClient, nil),
			mockS3HandlerFactory.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil),
			mockClient.EXPECT().Close(),
			mockAPI.EXPECT().DeleteMessage(gomock.Any(), gomock.Eq(&msg)).Return(nil),
		)

		p := newSQSS3EventProcessor(logp.NewLogger(inputName), nil, mockAPI, nil, time.Minute, 5, mockBeatPipeline, mockS3HandlerFactory)
		require.NoError(t, p.ProcessSQS(ctx, &msg))
	})

	t.Run("invalid SQS JSON body does not retry", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()

		ctrl, ctx := gomock.WithContext(ctx, t)
		defer ctrl.Finish()
		mockAPI := NewMockSQSAPI(ctrl)
		mockS3HandlerFactory := NewMockS3ObjectHandlerFactory(ctrl)
		mockBeatPipeline := NewMockBeatPipeline(ctrl)

		invalidBodyMsg := newSQSMessage(newS3Event("log.json"))
		body := *invalidBodyMsg.Body
		body = body[10:]
		invalidBodyMsg.Body = &body

		gomock.InOrder(
			mockAPI.EXPECT().DeleteMessage(gomock.Any(), gomock.Eq(&invalidBodyMsg)).Return(nil),
		)

		p := newSQSS3EventProcessor(logp.NewLogger(inputName), nil, mockAPI, nil, time.Minute, 5, mockBeatPipeline, mockS3HandlerFactory)
		err := p.ProcessSQS(ctx, &invalidBodyMsg)
		require.Error(t, err)
		t.Log(err)
	})

	t.Run("zero S3 events in body", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()

		ctrl, ctx := gomock.WithContext(ctx, t)
		defer ctrl.Finish()
		mockAPI := NewMockSQSAPI(ctrl)
		mockS3HandlerFactory := NewMockS3ObjectHandlerFactory(ctrl)
		mockBeatPipeline := NewMockBeatPipeline(ctrl)

		emptyRecordsMsg := newSQSMessage([]s3EventV2{}...)

		gomock.InOrder(
			mockAPI.EXPECT().DeleteMessage(gomock.Any(), gomock.Eq(&emptyRecordsMsg)).Return(nil),
		)

		p := newSQSS3EventProcessor(logp.NewLogger(inputName), nil, mockAPI, nil, time.Minute, 5, mockBeatPipeline, mockS3HandlerFactory)
		require.NoError(t, p.ProcessSQS(ctx, &emptyRecordsMsg))
	})

	t.Run("visibility is extended after half expires", func(t *testing.T) {
		const visibilityTimeout = time.Second

		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()

		ctrl, ctx := gomock.WithContext(ctx, t)
		defer ctrl.Finish()
		mockAPI := NewMockSQSAPI(ctrl)
		mockS3HandlerFactory := NewMockS3ObjectHandlerFactory(ctrl)
		mockS3Handler := NewMockS3ObjectHandler(ctrl)
		mockClient := NewMockBeatClient(ctrl)
		mockBeatPipeline := NewMockBeatPipeline(ctrl)

		mockAPI.EXPECT().ChangeMessageVisibility(gomock.Any(), gomock.Eq(&msg), gomock.Eq(visibilityTimeout)).AnyTimes().Return(nil)

		gomock.InOrder(
			mockBeatPipeline.EXPECT().ConnectWith(gomock.Any()).Return(mockClient, nil),
			mockS3HandlerFactory.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Do(func(ctx context.Context, _ *logp.Logger, _ beat.Client, _ *awscommon.EventACKTracker, _ s3EventV2) {
					require.NoError(t, timed.Wait(ctx, 5*visibilityTimeout))
				}).Return(mockS3Handler),
			mockS3Handler.EXPECT().ProcessS3Object().Return(nil),
			mockClient.EXPECT().Close(),
			mockAPI.EXPECT().DeleteMessage(gomock.Any(), gomock.Eq(&msg)).Return(nil),
			mockS3Handler.EXPECT().FinalizeS3Object().Return(nil),
		)

		p := newSQSS3EventProcessor(logp.NewLogger(inputName), nil, mockAPI, nil, visibilityTimeout, 5, mockBeatPipeline, mockS3HandlerFactory)
		require.NoError(t, p.ProcessSQS(ctx, &msg))
	})

	t.Run("message returns to queue on error", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()

		ctrl, ctx := gomock.WithContext(ctx, t)
		defer ctrl.Finish()
		mockAPI := NewMockSQSAPI(ctrl)
		mockS3HandlerFactory := NewMockS3ObjectHandlerFactory(ctrl)
		mockS3Handler := NewMockS3ObjectHandler(ctrl)
		mockClient := NewMockBeatClient(ctrl)
		mockBeatPipeline := NewMockBeatPipeline(ctrl)

		gomock.InOrder(
			mockBeatPipeline.EXPECT().ConnectWith(gomock.Any()).Return(mockClient, nil),
			mockS3HandlerFactory.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(mockS3Handler),
			mockS3Handler.EXPECT().ProcessS3Object().Return(errors.New("fake connectivity problem")),
			mockClient.EXPECT().Close(),
		)

		p := newSQSS3EventProcessor(logp.NewLogger(inputName), nil, mockAPI, nil, time.Minute, 5, mockBeatPipeline, mockS3HandlerFactory)
		err := p.ProcessSQS(ctx, &msg)
		t.Log(err)
		require.Error(t, err)
	})

	t.Run("message is deleted after multiple receives", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()

		ctrl, ctx := gomock.WithContext(ctx, t)
		defer ctrl.Finish()
		mockAPI := NewMockSQSAPI(ctrl)
		mockS3HandlerFactory := NewMockS3ObjectHandlerFactory(ctrl)
		mockS3Handler := NewMockS3ObjectHandler(ctrl)
		mockClient := NewMockBeatClient(ctrl)
		mockBeatPipeline := NewMockBeatPipeline(ctrl)

		msg := msg
		msg.Attributes = map[string]string{
			sqsApproximateReceiveCountAttribute: "10",
		}

		gomock.InOrder(
			mockBeatPipeline.EXPECT().ConnectWith(gomock.Any()).Return(mockClient, nil),
			mockS3HandlerFactory.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(mockS3Handler),
			mockS3Handler.EXPECT().ProcessS3Object().Return(errors.New("fake connectivity problem")),
			mockClient.EXPECT().Close(),
			mockAPI.EXPECT().DeleteMessage(gomock.Any(), gomock.Eq(&msg)).Return(nil),
		)

		p := newSQSS3EventProcessor(logp.NewLogger(inputName), nil, mockAPI, nil, time.Minute, 5, mockBeatPipeline, mockS3HandlerFactory)
		err := p.ProcessSQS(ctx, &msg)
		t.Log(err)
		require.Error(t, err)
	})
}

func TestSqsProcessor_keepalive(t *testing.T) {
	msg := newSQSMessage(newS3Event("log.json"))

	// Ensure both ReceiptHandleIsInvalid and InvalidParameterValue error codes trigger stops.
	// See https://github.com/elastic/beats/issues/30675.
	testCases := []struct {
		Name string
		Err  error
	}{
		{
			Name: "keepalive stop after ReceiptHandleIsInvalid",
			Err:  &types.ReceiptHandleIsInvalid{Message: aws.String("fake receipt handle is invalid.")},
		},
		{
			Name: "keepalive stop after InvalidParameterValue",
			Err:  &smithy.GenericAPIError{Code: sqsInvalidParameterValueErrorCode, Message: "The receipt handle has expired."},
		},
	}

	for _, tc := range testCases {
		tc := tc

		// Test will call ChangeMessageVisibility once and then keepalive will
		// exit because the SQS receipt handle is not usable.
		t.Run(tc.Name, func(t *testing.T) {
			const visibilityTimeout = time.Second

			ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
			defer cancel()

			ctrl, ctx := gomock.WithContext(ctx, t)
			defer ctrl.Finish()
			mockAPI := NewMockSQSAPI(ctrl)
			mockS3HandlerFactory := NewMockS3ObjectHandlerFactory(ctrl)
			mockBeatPipeline := NewMockBeatPipeline(ctrl)

			mockAPI.EXPECT().ChangeMessageVisibility(gomock.Any(), gomock.Eq(&msg), gomock.Eq(visibilityTimeout)).
				Times(1).Return(tc.Err)

			p := newSQSS3EventProcessor(logp.NewLogger(inputName), nil, mockAPI, nil, visibilityTimeout, 5, mockBeatPipeline, mockS3HandlerFactory)
			var wg sync.WaitGroup
			wg.Add(1)
			p.keepalive(ctx, p.log, &wg, &msg)
			wg.Wait()
		})
	}
}

func TestSqsProcessor_getS3Notifications(t *testing.T) {
	require.NoError(t, logp.TestingSetup())

	p := newSQSS3EventProcessor(logp.NewLogger(inputName), nil, nil, nil, time.Minute, 5, nil, nil)

	t.Run("s3 key is url unescaped", func(t *testing.T) {
		msg := newSQSMessage(newS3Event("Happy+Face.jpg"))

		events, err := p.getS3Notifications(*msg.Body)
		require.NoError(t, err)
		assert.Len(t, events, 1)
		assert.Equal(t, "Happy Face.jpg", events[0].S3.Object.Key)
	})

	t.Run("non-ObjectCreated event types are ignored", func(t *testing.T) {
		event := newS3Event("HappyFace.jpg")
		event.EventName = "ObjectRemoved:Delete"
		msg := newSQSMessage(event)

		events, err := p.getS3Notifications(*msg.Body)
		require.NoError(t, err)
		assert.Len(t, events, 0)
	})

	t.Run("sns-sqs notification", func(t *testing.T) {
		msg := newSNSSQSMessage()
		events, err := p.getS3Notifications(*msg.Body)
		require.NoError(t, err)
		assert.Len(t, events, 1)
		assert.Equal(t, "test-object-key", events[0].S3.Object.Key)
		assert.Equal(t, "arn:aws:s3:::vpc-flow-logs-ks", events[0].S3.Bucket.ARN)
		assert.Equal(t, "vpc-flow-logs-ks", events[0].S3.Bucket.Name)
	})

	t.Run("missing Records fail", func(t *testing.T) {
		msg := `{"message":"missing records"}`
		_, err := p.getS3Notifications(msg)
		require.Error(t, err)
		assert.EqualError(t, err, "the message is an invalid S3 notification: missing Records field")
		msg = `{"message":"null records", "Records": null}`
		_, err = p.getS3Notifications(msg)
		require.Error(t, err)
		assert.EqualError(t, err, "the message is an invalid S3 notification: missing Records field")
	})

	t.Run("empty Records does not fail", func(t *testing.T) {
		msg := `{"Records":[]}`
		events, err := p.getS3Notifications(msg)
		require.NoError(t, err)
		assert.Equal(t, 0, len(events))
	})
}

func TestNonRecoverableError(t *testing.T) {
	e := nonRetryableErrorWrap(fmt.Errorf("failed"))
	assert.True(t, errors.Is(e, &nonRetryableError{}))

	var e2 *nonRetryableError
	assert.True(t, errors.As(e, &e2))
}
