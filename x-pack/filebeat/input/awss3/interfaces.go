// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

	awscommon "github.com/elastic/beats/v7/x-pack/libbeat/common/aws"

	smithyhttp "github.com/aws/smithy-go/transport/http"

	"github.com/aws/smithy-go/middleware"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"

	"github.com/elastic/beats/v7/libbeat/beat"

	"github.com/elastic/elastic-agent-libs/logp"
)

// Run 'go generate' to create mocks that are used in tests.
//go:generate go install github.com/golang/mock/mockgen@v1.6.0
//go:generate mockgen -source=interfaces.go -destination=mock_interfaces_test.go -package awss3 -mock_names=sqsAPI=MockSQSAPI,sqsProcessor=MockSQSProcessor,s3API=MockS3API,s3Pager=MockS3Pager,s3ObjectHandlerFactory=MockS3ObjectHandlerFactory,s3ObjectHandler=MockS3ObjectHandler
//go:generate mockgen -destination=mock_publisher_test.go -package=awss3 -mock_names=Client=MockBeatClient,Pipeline=MockBeatPipeline github.com/elastic/beats/v7/libbeat/beat Client,Pipeline

// ------
// SQS interfaces
// ------

const s3RequestURLMetadataKey = `x-beat-s3-request-url`

type sqsAPI interface {
	sqsReceiver
	sqsDeleter
	sqsVisibilityChanger
	sqsAttributeGetter
}

type sqsReceiver interface {
	ReceiveMessage(ctx context.Context, maxMessages int) ([]types.Message, error)
}

type sqsDeleter interface {
	DeleteMessage(ctx context.Context, msg *types.Message) error
}

type sqsVisibilityChanger interface {
	ChangeMessageVisibility(ctx context.Context, msg *types.Message, timeout time.Duration) error
}

type sqsAttributeGetter interface {
	GetQueueAttributes(ctx context.Context, attr []types.QueueAttributeName) (map[string]string, error)
}

type sqsProcessor interface {
	// ProcessSQS processes and SQS message. It takes fully ownership of the
	// given message and is responsible for updating the message's visibility
	// timeout while it is being processed and for deleting it when processing
	// completes successfully.
	ProcessSQS(ctx context.Context, msg *types.Message, client beat.Client, acker *EventACKTracker, start time.Time) (uint64, error)
	DeleteSQS(msg *types.Message, receiveCount int, processingErr error, handles []s3ObjectHandler) error
}

// ------
// S3 interfaces
// ------

type s3API interface {
	s3Getter
	s3Mover
	s3Lister
}

type s3Getter interface {
	GetObject(ctx context.Context, bucket, key string) (*s3.GetObjectOutput, error)
}

type s3Mover interface {
	CopyObject(ctx context.Context, from_bucket, to_bucket, from_key, to_key string) (*s3.CopyObjectOutput, error)
	DeleteObject(ctx context.Context, bucket, key string) (*s3.DeleteObjectOutput, error)
}

type s3Lister interface {
	ListObjectsPaginator(bucket, prefix string) s3Pager
}

type s3Pager interface {
	HasMorePages() bool // NextPage retrieves the next ListObjectsV2 page.
	NextPage(ctx context.Context, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error)
}

type s3ObjectHandlerFactory interface {
	// Create returns a new s3ObjectHandler that can be used to process the
	// specified S3 object. If the handler is not configured to process the
	// given S3 object (based on key name) then it will return nil.
	Create(ctx context.Context, log *logp.Logger, client beat.Client, acker *EventACKTracker, obj s3EventV2) s3ObjectHandler
	CreateForS3Polling(ctx context.Context, log *logp.Logger, client beat.Client, acker *awscommon.EventACKTracker, obj s3EventV2) s3ObjectHandler
}

type s3ObjectHandler interface {
	// ProcessS3Object downloads the S3 object, parses it, creates events, and
	// publishes them. It returns when processing finishes or when it encounters
	// an unrecoverable error. It does not wait for the events to be ACKed by
	// the publisher before returning.
	ProcessS3Object() (uint64, error)

	// FinalizeS3Object finalizes processing of an S3 object after the current
	// batch is finished.
	FinalizeS3Object() error

	// Wait waits for every event published by ProcessS3Object() to be ACKed
	// by the publisher before returning. Internally it uses the
	// s3ObjectHandler ackerForPolling's Wait() method
	Wait()
}

// ------
// AWS SQS implementation
// ------

type awsSQSAPI struct {
	client            *sqs.Client
	queueURL          string
	apiTimeout        time.Duration
	visibilityTimeout time.Duration
	longPollWaitTime  time.Duration
}

func (a *awsSQSAPI) ReceiveMessage(ctx context.Context, maxMessages int) ([]types.Message, error) {
	const sqsMaxNumberOfMessagesLimit = 10
	ctx, cancel := context.WithTimeout(ctx, a.apiTimeout)
	defer cancel()

	receiveMessageOutput, err := a.client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
		QueueUrl:            awssdk.String(a.queueURL),
		MaxNumberOfMessages: int32(min(maxMessages, sqsMaxNumberOfMessagesLimit)),
		VisibilityTimeout:   int32(a.visibilityTimeout.Seconds()),
		WaitTimeSeconds:     int32(a.longPollWaitTime.Seconds()),
		AttributeNames:      []types.QueueAttributeName{sqsApproximateReceiveCountAttribute, sqsSentTimestampAttribute},
	})
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			err = fmt.Errorf("api_timeout exceeded: %w", err)
		}
		return nil, fmt.Errorf("sqs ReceiveMessage failed: %w", err)
	}

	return receiveMessageOutput.Messages, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (a *awsSQSAPI) DeleteMessage(ctx context.Context, msg *types.Message) error {
	ctx, cancel := context.WithTimeout(ctx, a.apiTimeout)
	defer cancel()
	_, err := a.client.DeleteMessage(ctx,
		&sqs.DeleteMessageInput{
			QueueUrl:      awssdk.String(a.queueURL),
			ReceiptHandle: msg.ReceiptHandle,
		})

	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			err = fmt.Errorf("api_timeout exceeded: %w", err)
		}
		return fmt.Errorf("sqs DeleteMessage failed: %w", err)
	}

	return nil
}

func (a *awsSQSAPI) ChangeMessageVisibility(ctx context.Context, msg *types.Message, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, a.apiTimeout)
	defer cancel()

	_, err := a.client.ChangeMessageVisibility(ctx,
		&sqs.ChangeMessageVisibilityInput{
			QueueUrl:          awssdk.String(a.queueURL),
			ReceiptHandle:     msg.ReceiptHandle,
			VisibilityTimeout: int32(timeout.Seconds()),
		})

	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			err = fmt.Errorf("api_timeout exceeded: %w", err)
		}
		return fmt.Errorf("sqs ChangeMessageVisibility failed: %w", err)
	}

	return nil
}

func (a *awsSQSAPI) GetQueueAttributes(ctx context.Context, attr []types.QueueAttributeName) (map[string]string, error) {
	ctx, cancel := context.WithTimeout(ctx, a.apiTimeout)
	defer cancel()

	attributeOutput, err := a.client.GetQueueAttributes(ctx, &sqs.GetQueueAttributesInput{
		AttributeNames: attr,
		QueueUrl:       awssdk.String(a.queueURL),
	})

	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			err = fmt.Errorf("api_timeout exceeded: %w", err)
		}
		return nil, fmt.Errorf("sqs GetQueueAttributes failed: %w", err)
	}

	return attributeOutput.Attributes, nil
}

// ------
// AWS S3 implementation
// ------

type awsS3API struct {
	client *s3.Client
}

func (a *awsS3API) GetObject(ctx context.Context, bucket, key string) (*s3.GetObjectOutput, error) {
	getObjectOutput, err := a.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: awssdk.String(bucket),
		Key:    awssdk.String(key),
	}, s3.WithAPIOptions(
		func(stack *middleware.Stack) error {
			// adds AFTER operation finalize middleware
			return stack.Finalize.Add(middleware.FinalizeMiddlewareFunc("add s3 request url to metadata",
				func(ctx context.Context, in middleware.FinalizeInput, next middleware.FinalizeHandler) (
					out middleware.FinalizeOutput, metadata middleware.Metadata, err error,
				) {
					out, metadata, err = next.HandleFinalize(ctx, in)
					requestURL, parseErr := url.Parse(in.Request.(*smithyhttp.Request).URL.String())
					if parseErr != nil {
						return out, metadata, err
					}

					requestURL.RawQuery = ""

					metadata.Set(s3RequestURLMetadataKey, requestURL.String())

					return out, metadata, err
				},
			), middleware.After)
		}))

	if err != nil {
		return nil, fmt.Errorf("s3 GetObject failed: %w", err)
	}

	return getObjectOutput, nil
}

func (a *awsS3API) CopyObject(ctx context.Context, from_bucket, to_bucket, from_key, to_key string) (*s3.CopyObjectOutput, error) {
	copyObjectOutput, err := a.client.CopyObject(ctx, &s3.CopyObjectInput{
		Bucket:     awssdk.String(to_bucket),
		CopySource: awssdk.String(fmt.Sprintf("%s/%s", from_bucket, from_key)),
		Key:        awssdk.String(to_key),
	})
	if err != nil {
		return nil, fmt.Errorf("s3 CopyObject failed: %w", err)
	}
	return copyObjectOutput, nil
}

func (a *awsS3API) DeleteObject(ctx context.Context, bucket, key string) (*s3.DeleteObjectOutput, error) {
	deleteObjectOutput, err := a.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: awssdk.String(bucket),
		Key:    awssdk.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("s3 DeleteObject failed: %w", err)
	}
	return deleteObjectOutput, nil
}

func (a *awsS3API) ListObjectsPaginator(bucket, prefix string) s3Pager {
	pager := s3.NewListObjectsV2Paginator(a.client, &s3.ListObjectsV2Input{
		Bucket: awssdk.String(bucket),
		Prefix: awssdk.String(prefix),
	})

	return pager
}
