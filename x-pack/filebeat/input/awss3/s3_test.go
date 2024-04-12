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
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/statestore"
	"github.com/elastic/beats/v7/libbeat/statestore/storetest"
	"github.com/elastic/elastic-agent-libs/logp"
)

func TestS3Poller(t *testing.T) {
	logp.TestingSetup()

	const bucket = "bucket"
	const numberOfWorkers = 5
	const pollInterval = 2 * time.Second
	const testTimeout = 1 * time.Second

	t.Run("Poll success", func(t *testing.T) {
		storeReg := statestore.NewRegistry(storetest.NewMemoryStoreBackend())
		store, err := storeReg.Get("test")
		if err != nil {
			t.Fatalf("Failed to access store: %v", err)
		}

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
		receiver := newS3Poller(logp.NewLogger(inputName), nil, mockAPI, mockPublisher, s3ObjProc, newStates(inputCtx), store, bucket, "key", "region", "provider", numberOfWorkers, pollInterval)
		receiver.Poll(ctx)
		assert.Equal(t, numberOfWorkers, receiver.workerSem.Available())
	})

	t.Run("retry after Poll error", func(t *testing.T) {
		storeReg := statestore.NewRegistry(storetest.NewMemoryStoreBackend())
		store, err := storeReg.Get("test")
		if err != nil {
			t.Fatalf("Failed to access store: %v", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), testTimeout+pollInterval)
		defer cancel()

		ctrl, ctx := gomock.WithContext(ctx, t)
		defer ctrl.Finish()
		mockAPI := NewMockS3API(ctrl)
		mockPagerFirst := NewMockS3Pager(ctrl)
		mockPagerSecond := NewMockS3Pager(ctrl)
		mockPublisher := NewMockBeatClient(ctrl)

		gomock.InOrder(
			// Initial ListObjectPaginator gets an error.
			mockAPI.EXPECT().
				ListObjectsPaginator(gomock.Eq(bucket), gomock.Eq("key")).
				Times(1).
				DoAndReturn(func(_, _ string) s3Pager {
					return mockPagerFirst
				}),
			// After waiting for pollInterval, it retries.
			mockAPI.EXPECT().
				ListObjectsPaginator(gomock.Eq(bucket), gomock.Eq("key")).
				Times(1).
				DoAndReturn(func(_, _ string) s3Pager {
					return mockPagerSecond
				}),
		)

		// Initial Next gets an error.
		mockPagerFirst.EXPECT().
			HasMorePages().
			Times(10).
			DoAndReturn(func() bool {
				return true
			})
		mockPagerFirst.EXPECT().
			NextPage(gomock.Any()).
			Times(5).
			DoAndReturn(func(_ context.Context, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
				return nil, errFakeConnectivityFailure
			})

		// After waiting for pollInterval, it retries.
		mockPagerSecond.EXPECT().
			HasMorePages().
			Times(1).
			DoAndReturn(func() bool {
				return true
			})
		mockPagerSecond.EXPECT().
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

		mockPagerSecond.EXPECT().
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

		s3ObjProc := newS3ObjectProcessorFactory(logp.NewLogger(inputName), nil, mockAPI, nil, backupConfig{})
		receiver := newS3Poller(logp.NewLogger(inputName), nil, mockAPI, mockPublisher, s3ObjProc, newStates(inputCtx), store, bucket, "key", "region", "provider", numberOfWorkers, pollInterval)
		receiver.Poll(ctx)
	})
}
