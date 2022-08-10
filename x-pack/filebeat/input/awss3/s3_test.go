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
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/statestore"
	"github.com/elastic/beats/v7/libbeat/statestore/storetest"
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
			Next(gomock.Any()).
			Times(1).
			DoAndReturn(func(_ context.Context) bool {
				return true
			})

		mockPager.EXPECT().
			CurrentPage().
			Times(1).
			DoAndReturn(func() *s3.ListObjectsOutput {
				return &s3.ListObjectsOutput{
					Contents: []s3.Object{
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
				}
			})

		mockPager.EXPECT().
			Next(gomock.Any()).
			Times(1).
			DoAndReturn(func(_ context.Context) bool {
				return false
			})

		mockPager.EXPECT().
			Err().
			Times(1).
			DoAndReturn(func() error {
				return nil
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

		s3ObjProc := newS3ObjectProcessorFactory(logp.NewLogger(inputName), nil, mockAPI, mockPublisher, nil)
		receiver := newS3Poller(logp.NewLogger(inputName), nil, mockAPI, s3ObjProc, newStates(inputCtx), store, bucket, "key", "region", "provider", numberOfWorkers, pollInterval)
		require.Error(t, context.DeadlineExceeded, receiver.Poll(ctx))
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
			Next(gomock.Any()).
			Times(1).
			DoAndReturn(func(_ context.Context) bool {
				return false
			})
		mockPagerFirst.EXPECT().
			Err().
			Times(1).
			DoAndReturn(func() error {
				return errFakeConnectivityFailure
			})

		// After waiting for pollInterval, it retries.
		mockPagerSecond.EXPECT().
			Next(gomock.Any()).
			Times(1).
			DoAndReturn(func(_ context.Context) bool {
				return true
			})
		mockPagerSecond.EXPECT().
			CurrentPage().
			Times(1).
			DoAndReturn(func() *s3.ListObjectsOutput {
				return &s3.ListObjectsOutput{
					Contents: []s3.Object{
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
				}
			})

		mockPagerSecond.EXPECT().
			Next(gomock.Any()).
			Times(1).
			DoAndReturn(func(_ context.Context) bool {
				return false
			})

		mockPagerSecond.EXPECT().
			Err().
			Times(1).
			DoAndReturn(func() error {
				return nil
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

		s3ObjProc := newS3ObjectProcessorFactory(logp.NewLogger(inputName), nil, mockAPI, mockPublisher, nil)
		receiver := newS3Poller(logp.NewLogger(inputName), nil, mockAPI, s3ObjProc, newStates(inputCtx), store, bucket, "key", "region", "provider", numberOfWorkers, pollInterval)
		require.Error(t, context.DeadlineExceeded, receiver.Poll(ctx))
		assert.Equal(t, numberOfWorkers, receiver.workerSem.Available())
	})
}
