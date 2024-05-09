// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/logp"
)

func TestS3Poller(t *testing.T) {
	logp.TestingSetup()

	const bucket = "bucket"
	const numberOfWorkers = 5
	const pollInterval = 2 * time.Second
	const testTimeout = 1 * time.Second

	t.Run("Poll success", func(t *testing.T) {
		store := openTestStatestore()

		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()

		ctrl, ctx := gomock.WithContext(ctx, t)
		defer ctrl.Finish()
		mockAPI := NewMockS3API(ctrl)
		mockPager := NewMockS3Pager(ctrl)
		mockPublisher := NewMockBeatClient(ctrl)

		gomock.InOrder(
			mockAPI.EXPECT().
				ListObjectsPaginator(gomock.Eq(bucket), gomock.Eq("key")).
				Times(1).
				DoAndReturn(func(_, _ string) s3Pager {
					return mockPager
				}),
		)

		// Initial Poll
		mockPager.EXPECT().
			HasMorePages().
			Times(1).
			DoAndReturn(func() bool {
				return true
			})

		mockPager.EXPECT().
			NextPage(gomock.Any()).
			Times(1).
			DoAndReturn(func(_ context.Context, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
				return &s3.ListObjectsV2Output{
					Contents: []types.Object{
						{
							ETag:         aws.String("etag1"),
							Key:          aws.String("key1"),
							LastModified: aws.Time(time.Now()),
						},
						{
							ETag:         aws.String("etag2"),
							Key:          aws.String("key2"),
							LastModified: aws.Time(time.Now()),
						},
						{
							ETag:         aws.String("etag3"),
							Key:          aws.String("key3"),
							LastModified: aws.Time(time.Now()),
						},
						{
							ETag:         aws.String("etag4"),
							Key:          aws.String("key4"),
							LastModified: aws.Time(time.Now()),
						},
						{
							ETag:         aws.String("etag5"),
							Key:          aws.String("key5"),
							LastModified: aws.Time(time.Now()),
						},
						{
							ETag:         aws.String("etag6"),
							Key:          aws.String("2024-02-08T08:35:00+00:02.json.gz"),
							LastModified: aws.Time(time.Now()),
						},
					},
				}, nil
			})

		mockPager.EXPECT().
			HasMorePages().
			Times(1).
			DoAndReturn(func() bool {
				return false
			})

		mockAPI.EXPECT().
			GetObject(gomock.Any(), gomock.Eq(bucket), gomock.Eq("key1")).
			Return(nil, errFakeConnectivityFailure)

		mockAPI.EXPECT().
			GetObject(gomock.Any(), gomock.Eq(bucket), gomock.Eq("key2")).
			Return(nil, errFakeConnectivityFailure)

		mockAPI.EXPECT().
			GetObject(gomock.Any(), gomock.Eq(bucket), gomock.Eq("key3")).
			Return(nil, errFakeConnectivityFailure)

		mockAPI.EXPECT().
			GetObject(gomock.Any(), gomock.Eq(bucket), gomock.Eq("key4")).
			Return(nil, errFakeConnectivityFailure)

		mockAPI.EXPECT().
			GetObject(gomock.Any(), gomock.Eq(bucket), gomock.Eq("key5")).
			Return(nil, errFakeConnectivityFailure)

		mockAPI.EXPECT().
			GetObject(gomock.Any(), gomock.Eq(bucket), gomock.Eq("2024-02-08T08:35:00+00:02.json.gz")).
			Return(nil, errFakeConnectivityFailure)

		s3ObjProc := newS3ObjectProcessorFactory(logp.NewLogger(inputName), nil, mockAPI, nil, backupConfig{})
		states, err := newStates(nil, store)
		require.NoError(t, err, "states creation must succeed")
		poller := &s3PollerInput{
			log: logp.NewLogger(inputName),
			config: config{
				NumberOfWorkers:    numberOfWorkers,
				BucketListInterval: pollInterval,
				BucketARN:          bucket,
				BucketListPrefix:   "key",
				RegionName:         "region",
			},
			s3:              mockAPI,
			client:          mockPublisher,
			s3ObjectHandler: s3ObjProc,
			states:          states,
			provider:        "provider",
			metrics:         newInputMetrics("", nil, 0),
		}
		poller.runPoll(ctx)
	})

	t.Run("restart bucket scan after paging errors", func(t *testing.T) {
		// Change the restart limit to 2 consecutive errors, so the test doesn't
		// take too long to run
		readerLoopMaxCircuitBreaker = 2
		store := openTestStatestore()

		ctx, cancel := context.WithTimeout(context.Background(), testTimeout+pollInterval)
		defer cancel()

		ctrl, ctx := gomock.WithContext(ctx, t)
		defer ctrl.Finish()
		mockS3 := NewMockS3API(ctrl)
		mockErrorPager := NewMockS3Pager(ctrl)
		mockSuccessPager := NewMockS3Pager(ctrl)
		mockPublisher := NewMockBeatClient(ctrl)

		gomock.InOrder(
			// Initial ListObjectPaginator gets an error.
			mockS3.EXPECT().
				ListObjectsPaginator(gomock.Eq(bucket), gomock.Eq("key")).
				Times(1).
				DoAndReturn(func(_, _ string) s3Pager {
					return mockErrorPager
				}),
			// After waiting for pollInterval, it retries.
			mockS3.EXPECT().
				ListObjectsPaginator(gomock.Eq(bucket), gomock.Eq("key")).
				Times(1).
				DoAndReturn(func(_, _ string) s3Pager {
					return mockSuccessPager
				}),
		)

		// Initial Next gets an error.
		mockErrorPager.EXPECT().
			HasMorePages().
			Times(2).
			DoAndReturn(func() bool {
				return true
			})
		mockErrorPager.EXPECT().
			NextPage(gomock.Any()).
			Times(2).
			DoAndReturn(func(_ context.Context, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
				return nil, errFakeConnectivityFailure
			})

		// After waiting for pollInterval, it retries.
		mockSuccessPager.EXPECT().
			HasMorePages().
			Times(1).
			DoAndReturn(func() bool {
				return true
			})
		mockSuccessPager.EXPECT().
			NextPage(gomock.Any()).
			Times(1).
			DoAndReturn(func(_ context.Context, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
				return &s3.ListObjectsV2Output{
					Contents: []types.Object{
						{
							ETag:         aws.String("etag1"),
							Key:          aws.String("key1"),
							LastModified: aws.Time(time.Now()),
						},
						{
							ETag:         aws.String("etag2"),
							Key:          aws.String("key2"),
							LastModified: aws.Time(time.Now()),
						},
						{
							ETag:         aws.String("etag3"),
							Key:          aws.String("key3"),
							LastModified: aws.Time(time.Now()),
						},
						{
							ETag:         aws.String("etag4"),
							Key:          aws.String("key4"),
							LastModified: aws.Time(time.Now()),
						},
						{
							ETag:         aws.String("etag5"),
							Key:          aws.String("key5"),
							LastModified: aws.Time(time.Now()),
						},
					},
				}, nil
			})

		mockSuccessPager.EXPECT().
			HasMorePages().
			Times(1).
			DoAndReturn(func() bool {
				return false
			})

		mockS3.EXPECT().
			GetObject(gomock.Any(), gomock.Eq(bucket), gomock.Eq("key1")).
			Return(nil, errFakeConnectivityFailure)

		mockS3.EXPECT().
			GetObject(gomock.Any(), gomock.Eq(bucket), gomock.Eq("key2")).
			Return(nil, errFakeConnectivityFailure)

		mockS3.EXPECT().
			GetObject(gomock.Any(), gomock.Eq(bucket), gomock.Eq("key3")).
			Return(nil, errFakeConnectivityFailure)

		mockS3.EXPECT().
			GetObject(gomock.Any(), gomock.Eq(bucket), gomock.Eq("key4")).
			Return(nil, errFakeConnectivityFailure)

		mockS3.EXPECT().
			GetObject(gomock.Any(), gomock.Eq(bucket), gomock.Eq("key5")).
			Return(nil, errFakeConnectivityFailure)

		s3ObjProc := newS3ObjectProcessorFactory(logp.NewLogger(inputName), nil, mockS3, nil, backupConfig{})
		states, err := newStates(nil, store)
		require.NoError(t, err, "states creation must succeed")
		poller := &s3PollerInput{
			log: logp.NewLogger(inputName),
			config: config{
				NumberOfWorkers:    numberOfWorkers,
				BucketListInterval: pollInterval,
				BucketARN:          bucket,
				BucketListPrefix:   "key",
				RegionName:         "region",
			},
			s3:              mockS3,
			client:          mockPublisher,
			s3ObjectHandler: s3ObjProc,
			states:          states,
			provider:        "provider",
			metrics:         newInputMetrics("", nil, 0),
		}
		poller.run(ctx)
	})
}

func TestS3ReaderLoop(t *testing.T) {

}

func TestS3WorkerLoop(t *testing.T) {

}
