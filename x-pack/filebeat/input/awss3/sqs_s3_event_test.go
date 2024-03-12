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
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/go-concert/timed"
)

func TestSQSS3EventProcessor(t *testing.T) {
	logp.TestingSetup()

	msg := newSQSMessage(newS3Event("log.json"))

	t.Run("s3 events are processed and sqs msg is deleted", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()

		ctrl, ctx := gomock.WithContext(ctx, t)
		defer ctrl.Finish()
		mockAPI := NewMockSQSAPI(ctrl)
		mockS3Handler := NewMockS3ObjectHandler(ctrl)
		mockS3HandlerFactory := NewMockS3ObjectHandlerFactory(ctrl)
		mockClient := NewMockBeatClient(ctrl)

		expectedEventsPublishedTotal := uint64(10)
		gomock.InOrder(
			mockS3HandlerFactory.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(mockS3Handler),
			mockS3Handler.EXPECT().ProcessS3Object().Return(expectedEventsPublishedTotal, nil),
			mockS3Handler.EXPECT().FinalizeS3Object().Return(nil),
		)

		p := newSQSS3EventProcessor(logp.NewLogger(inputName), nil, mockAPI, nil, time.Minute, 5, mockS3HandlerFactory)

		acker := NewEventACKTracker(ctx, new(sync.WaitGroup))
		eventsPublishedTotal, processingErr := p.ProcessSQS(ctx, &msg, mockClient, acker, time.Now())
		require.NoError(t, processingErr)
		require.Equal(t, expectedEventsPublishedTotal, eventsPublishedTotal)

		mockAPI.EXPECT().DeleteMessage(gomock.Any(), gomock.Eq(&msg)).Return(nil)

		require.NoError(t, p.DeleteSQS(&msg, acker.ReceiveCount, processingErr, acker.Handles))
	})

	t.Run("invalid SQS JSON body does not retry", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()

		ctrl, ctx := gomock.WithContext(ctx, t)
		defer ctrl.Finish()
		mockAPI := NewMockSQSAPI(ctrl)
		mockS3HandlerFactory := NewMockS3ObjectHandlerFactory(ctrl)

		invalidBodyMsg := newSQSMessage(newS3Event("log.json"))
		body := *invalidBodyMsg.Body
		body = body[10:]
		invalidBodyMsg.Body = &body

		mockClient := NewMockBeatClient(ctrl)

		p := newSQSS3EventProcessor(logp.NewLogger(inputName), nil, mockAPI, nil, time.Minute, 5, mockS3HandlerFactory)
		acker := NewEventACKTracker(ctx, new(sync.WaitGroup))
		eventsPublishedTotal, processingErr := p.ProcessSQS(ctx, &invalidBodyMsg, mockClient, acker, time.Now())
		t.Log(processingErr)
		require.Error(t, processingErr)
		require.Equal(t, uint64(0), eventsPublishedTotal)

		mockAPI.EXPECT().DeleteMessage(gomock.Any(), gomock.Eq(&invalidBodyMsg)).Return(nil)

		require.Error(t, p.DeleteSQS(&invalidBodyMsg, acker.ReceiveCount, processingErr, acker.Handles))
	})

	t.Run("zero S3 events in body", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()

		ctrl, ctx := gomock.WithContext(ctx, t)
		defer ctrl.Finish()
		mockAPI := NewMockSQSAPI(ctrl)
		mockS3HandlerFactory := NewMockS3ObjectHandlerFactory(ctrl)

		emptyRecordsMsg := newSQSMessage([]s3EventV2{}...)

		mockClient := NewMockBeatClient(ctrl)

		p := newSQSS3EventProcessor(logp.NewLogger(inputName), nil, mockAPI, nil, time.Minute, 5, mockS3HandlerFactory)
		acker := NewEventACKTracker(ctx, new(sync.WaitGroup))
		eventsPublishedTotal, processingErr := p.ProcessSQS(ctx, &emptyRecordsMsg, mockClient, acker, time.Now())
		require.NoError(t, processingErr)
		require.Equal(t, uint64(0), eventsPublishedTotal)

		mockAPI.EXPECT().DeleteMessage(gomock.Any(), gomock.Eq(&emptyRecordsMsg)).Return(nil)

		require.NoError(t, p.DeleteSQS(&emptyRecordsMsg, acker.ReceiveCount, processingErr, acker.Handles))
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

		mockAPI.EXPECT().ChangeMessageVisibility(gomock.Any(), gomock.Eq(&msg), gomock.Eq(visibilityTimeout)).AnyTimes().Return(nil)

		gomock.InOrder(
			mockS3HandlerFactory.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Do(func(ctx context.Context, _ *logp.Logger, _ beat.Client, _ *EventACKTracker, _ s3EventV2) {
					require.NoError(t, timed.Wait(ctx, 5*visibilityTimeout))
				}).Return(mockS3Handler),
			mockS3Handler.EXPECT().ProcessS3Object().Return(uint64(1), nil),
		)

		p := newSQSS3EventProcessor(logp.NewLogger(inputName), nil, mockAPI, nil, visibilityTimeout, 5, mockS3HandlerFactory)
		acker := NewEventACKTracker(ctx, new(sync.WaitGroup))
		eventsPublishedTotal, processingErr := p.ProcessSQS(ctx, &msg, mockClient, acker, time.Now())
		require.NoError(t, processingErr)
		require.Equal(t, uint64(1), eventsPublishedTotal)

		gomock.InOrder(
			mockAPI.EXPECT().DeleteMessage(gomock.Any(), gomock.Eq(&msg)).Return(nil),
			mockS3Handler.EXPECT().FinalizeS3Object().Return(nil),
		)

		require.NoError(t, p.DeleteSQS(&msg, acker.ReceiveCount, processingErr, acker.Handles))
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

		gomock.InOrder(
			mockS3HandlerFactory.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(mockS3Handler),
			mockS3Handler.EXPECT().ProcessS3Object().Return(uint64(0), errors.New("fake connectivity problem")),
		)

		p := newSQSS3EventProcessor(logp.NewLogger(inputName), nil, mockAPI, nil, time.Minute, 5, mockS3HandlerFactory)
		acker := NewEventACKTracker(ctx, new(sync.WaitGroup))
		eventsPublishedTotal, processingErr := p.ProcessSQS(ctx, &msg, mockClient, acker, time.Now())
		t.Log(processingErr)
		require.Error(t, processingErr)
		require.Equal(t, uint64(0), eventsPublishedTotal)

		require.Error(t, p.DeleteSQS(&msg, acker.ReceiveCount, processingErr, acker.Handles))
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

		msg := msg
		msg.Attributes = map[string]string{
			sqsApproximateReceiveCountAttribute: "10",
		}

		gomock.InOrder(
			mockS3HandlerFactory.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(mockS3Handler),
			mockS3Handler.EXPECT().ProcessS3Object().Return(uint64(0), errors.New("fake connectivity problem")),
		)

		p := newSQSS3EventProcessor(logp.NewLogger(inputName), nil, mockAPI, nil, time.Minute, 5, mockS3HandlerFactory)
		acker := NewEventACKTracker(ctx, new(sync.WaitGroup))
		eventsPublishedTotal, processingErr := p.ProcessSQS(ctx, &msg, mockClient, acker, time.Now())
		t.Log(processingErr)
		require.Error(t, processingErr)
		require.Equal(t, uint64(0), eventsPublishedTotal)

		mockAPI.EXPECT().DeleteMessage(gomock.Any(), gomock.Eq(&msg)).Return(nil)

		require.Error(t, p.DeleteSQS(&msg, acker.ReceiveCount, processingErr, acker.Handles))
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

			mockAPI.EXPECT().ChangeMessageVisibility(gomock.Any(), gomock.Eq(&msg), gomock.Eq(visibilityTimeout)).
				Times(1).Return(tc.Err)

			p := newSQSS3EventProcessor(logp.NewLogger(inputName), nil, mockAPI, nil, visibilityTimeout, 5, mockS3HandlerFactory)
			var wg sync.WaitGroup
			wg.Add(1)
			p.keepalive(ctx, p.log, &wg, &msg)
			wg.Wait()
		})
	}
}

func TestSqsProcessor_getS3Notifications(t *testing.T) {
	logp.TestingSetup()

	p := newSQSS3EventProcessor(logp.NewLogger(inputName), nil, nil, nil, time.Minute, 5, nil)

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
