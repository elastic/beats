// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"context"
	"testing"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestS3Poller(t *testing.T) {

	const bucket = "bucket"
	const listPrefix = "key"
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
		pipeline := newFakePipeline()

		gomock.InOrder(
			mockAPI.EXPECT().
				ListObjectsPaginator(gomock.Eq(bucket), gomock.Eq("key"), gomock.Any()).
				Times(1).
				DoAndReturn(func(_, _, _ string) s3Pager {
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
			GetObject(gomock.Any(), gomock.Eq(""), gomock.Eq(bucket), gomock.Eq("key1")).
			Return(nil, errFakeConnectivityFailure)

		mockAPI.EXPECT().
			GetObject(gomock.Any(), gomock.Eq(""), gomock.Eq(bucket), gomock.Eq("key2")).
			Return(nil, errFakeConnectivityFailure)

		mockAPI.EXPECT().
			GetObject(gomock.Any(), gomock.Eq(""), gomock.Eq(bucket), gomock.Eq("key3")).
			Return(nil, errFakeConnectivityFailure)

		mockAPI.EXPECT().
			GetObject(gomock.Any(), gomock.Eq(""), gomock.Eq(bucket), gomock.Eq("key4")).
			Return(nil, errFakeConnectivityFailure)

		mockAPI.EXPECT().
			GetObject(gomock.Any(), gomock.Eq(""), gomock.Eq(bucket), gomock.Eq("key5")).
			Return(nil, errFakeConnectivityFailure)

		mockAPI.EXPECT().
			GetObject(gomock.Any(), gomock.Eq(""), gomock.Eq(bucket), gomock.Eq("2024-02-08T08:35:00+00:02.json.gz")).
			Return(nil, errFakeConnectivityFailure)

		s3ObjProc := newS3ObjectProcessorFactory(nil, mockAPI, nil, backupConfig{}, logp.NewNopLogger())
		registry, err := newStateRegistry(nil, store, listPrefix, false, 0)
		require.NoError(t, err, "registry creation must succeed")

		cfg := config{
			NumberOfWorkers:    numberOfWorkers,
			BucketListInterval: pollInterval,
			BucketARN:          bucket,
			BucketListPrefix:   listPrefix,
			RegionName:         "region",
		}
		log := logp.NewLogger(inputName)
		poller := &s3PollerInput{
			log:             log,
			config:          cfg,
			s3:              mockAPI,
			pipeline:        pipeline,
			s3ObjectHandler: s3ObjProc,
			registry:        registry,
			provider:        "provider",
			metrics:         newInputMetrics(monitoring.NewRegistry(), 0, logp.NewNopLogger()),
			filterProvider:  newFilterProvider(&cfg),
			strategy:        newPollingStrategy(cfg.LexicographicalOrdering, log),
			status:          &statusReporterHelperMock{},
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
		pipeline := newFakePipeline()

		gomock.InOrder(
			// Initial ListObjectPaginator gets an error.
			mockS3.EXPECT().
				ListObjectsPaginator(gomock.Eq(bucket), gomock.Eq("key"), gomock.Any()).
				Times(1).
				DoAndReturn(func(_, _, _ string) s3Pager {
					return mockErrorPager
				}),
			// After waiting for pollInterval, it retries.
			mockS3.EXPECT().
				ListObjectsPaginator(gomock.Eq(bucket), gomock.Eq("key"), gomock.Any()).
				Times(1).
				DoAndReturn(func(_, _, _ string) s3Pager {
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
			GetObject(gomock.Any(), gomock.Eq(""), gomock.Eq(bucket), gomock.Eq("key1")).
			Return(nil, errFakeConnectivityFailure)

		mockS3.EXPECT().
			GetObject(gomock.Any(), gomock.Eq(""), gomock.Eq(bucket), gomock.Eq("key2")).
			Return(nil, errFakeConnectivityFailure)

		mockS3.EXPECT().
			GetObject(gomock.Any(), gomock.Eq(""), gomock.Eq(bucket), gomock.Eq("key3")).
			Return(nil, errFakeConnectivityFailure)

		mockS3.EXPECT().
			GetObject(gomock.Any(), gomock.Eq(""), gomock.Eq(bucket), gomock.Eq("key4")).
			Return(nil, errFakeConnectivityFailure)

		mockS3.EXPECT().
			GetObject(gomock.Any(), gomock.Eq(""), gomock.Eq(bucket), gomock.Eq("key5")).
			Return(nil, errFakeConnectivityFailure)

		s3ObjProc := newS3ObjectProcessorFactory(nil, mockS3, nil, backupConfig{}, logp.NewNopLogger())
		registry, err := newStateRegistry(nil, store, listPrefix, false, 0)
		require.NoError(t, err, "registry creation must succeed")

		cfg := config{
			NumberOfWorkers:    numberOfWorkers,
			BucketListInterval: pollInterval,
			BucketARN:          bucket,
			BucketListPrefix:   "key",
			RegionName:         "region",
		}

		log := logp.NewLogger(inputName)
		poller := &s3PollerInput{
			log: log,
			config: config{
				NumberOfWorkers:    numberOfWorkers,
				BucketListInterval: pollInterval,
				BucketARN:          bucket,
				BucketListPrefix:   listPrefix,
				RegionName:         "region",
			},
			s3:              mockS3,
			pipeline:        pipeline,
			s3ObjectHandler: s3ObjProc,
			registry:        registry,
			provider:        "provider",
			metrics:         newInputMetrics(monitoring.NewRegistry(), 0, logp.NewNopLogger()),
			filterProvider:  newFilterProvider(&cfg),
			strategy:        newPollingStrategy(false, log),
			status:          &statusReporterHelperMock{},
		}
		poller.run(ctx)
	})

	t.Run("lexicographical ordering uses startAfterKey from oldest state", func(t *testing.T) {
		store := openTestStatestore()

		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()

		ctrl, ctx := gomock.WithContext(ctx, t)
		defer ctrl.Finish()
		mockAPI := NewMockS3API(ctrl)
		mockPager := NewMockS3Pager(ctrl)
		pipeline := newFakePipeline()

		registry, err := newStateRegistry(nil, store, "", true, 100)
		require.NoError(t, err, "registry creation must succeed")

		// This will be used as startAfterKey
		existingState := newState(bucket, "existing-key", "etag", time.Unix(1000, 0))
		existingState.Stored = true
		err = registry.AddState(existingState)
		require.NoError(t, err, "state add must succeed")

		// Mark an object in-flight and unmark it to trigger tail computation from completed state
		err = registry.MarkObjectInFlight("zzz-temp")
		require.NoError(t, err)
		err = registry.UnmarkObjectInFlight("zzz-temp")
		require.NoError(t, err)

		startAfterKey := registry.GetStartAfterKey()
		require.Equal(t, "existing-key", startAfterKey)

		// Expect ListObjectsPaginator to be called with startAfterKey = "existing-key"
		mockAPI.EXPECT().
			ListObjectsPaginator(gomock.Eq(bucket), gomock.Eq(""), gomock.Eq("existing-key")).
			Times(1).
			DoAndReturn(func(_, _, startAfterKey string) s3Pager {
				require.Equal(t, "existing-key", startAfterKey)
				return mockPager
			})

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
							ETag:         aws.String("etag-new"),
							Key:          aws.String("new-key"),
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
			GetObject(gomock.Any(), gomock.Eq(""), gomock.Eq(bucket), gomock.Eq("new-key")).
			Return(nil, errFakeConnectivityFailure)

		s3ObjProc := newS3ObjectProcessorFactory(nil, mockAPI, nil, backupConfig{}, logp.NewNopLogger())

		cfg := config{
			NumberOfWorkers:             numberOfWorkers,
			BucketListInterval:          pollInterval,
			BucketARN:                   bucket,
			BucketListPrefix:            "",
			RegionName:                  "region",
			LexicographicalOrdering:     true,
			LexicographicalLookbackKeys: 100,
		}
		log := logp.NewLogger(inputName)
		poller := &s3PollerInput{
			log:             log,
			config:          cfg,
			s3:              mockAPI,
			pipeline:        pipeline,
			s3ObjectHandler: s3ObjProc,
			registry:        registry,
			provider:        "provider",
			metrics:         newInputMetrics(monitoring.NewRegistry(), 0, logp.NewNopLogger()),
			filterProvider:  newFilterProvider(&cfg),
			strategy:        newPollingStrategy(cfg.LexicographicalOrdering, log),
			status:          &statusReporterHelperMock{},
		}
		poller.runPoll(ctx)
	})

	t.Run("lexicographical ordering with empty states uses empty startAfterKey", func(t *testing.T) {
		store := openTestStatestore()

		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()

		ctrl, ctx := gomock.WithContext(ctx, t)
		defer ctrl.Finish()
		mockAPI := NewMockS3API(ctrl)
		mockPager := NewMockS3Pager(ctrl)
		pipeline := newFakePipeline()

		// Create empty registry
		registry, err := newStateRegistry(nil, store, "", true, 100)
		require.NoError(t, err, "registry creation must succeed")

		startAfterKey := registry.GetStartAfterKey()
		require.Empty(t, startAfterKey)

		// Expect ListObjectsPaginator to be called with empty startAfterKey
		mockAPI.EXPECT().
			ListObjectsPaginator(gomock.Eq(bucket), gomock.Eq(""), gomock.Eq("")).
			Times(1).
			DoAndReturn(func(_, _, startAfterKey string) s3Pager {
				require.Equal(t, "", startAfterKey)
				return mockPager
			})

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
			GetObject(gomock.Any(), gomock.Eq(""), gomock.Eq(bucket), gomock.Eq("key1")).
			Return(nil, errFakeConnectivityFailure)

		s3ObjProc := newS3ObjectProcessorFactory(nil, mockAPI, nil, backupConfig{}, logp.NewNopLogger())

		cfg := config{
			NumberOfWorkers:             numberOfWorkers,
			BucketListInterval:          pollInterval,
			BucketARN:                   bucket,
			BucketListPrefix:            "",
			RegionName:                  "region",
			LexicographicalOrdering:     true,
			LexicographicalLookbackKeys: 100,
		}
		log := logp.NewLogger(inputName)
		poller := &s3PollerInput{
			log:             log,
			config:          cfg,
			s3:              mockAPI,
			pipeline:        pipeline,
			s3ObjectHandler: s3ObjProc,
			registry:        registry,
			provider:        "provider",
			metrics:         newInputMetrics(monitoring.NewRegistry(), 0, logp.NewNopLogger()),
			filterProvider:  newFilterProvider(&cfg),
			strategy:        newPollingStrategy(cfg.LexicographicalOrdering, log),
			status:          &statusReporterHelperMock{},
		}
		poller.runPoll(ctx)
	})

	t.Run("non-lexicographical ordering uses empty startAfterKey", func(t *testing.T) {
		store := openTestStatestore()

		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()

		ctrl, ctx := gomock.WithContext(ctx, t)
		defer ctrl.Finish()
		mockAPI := NewMockS3API(ctrl)
		mockPager := NewMockS3Pager(ctrl)
		pipeline := newFakePipeline()

		// Non-lexicographical mode
		registry, err := newStateRegistry(nil, store, "", false, 0)
		require.NoError(t, err, "registry creation must succeed")

		// Expect ListObjectsPaginator to be called with empty startAfterKey
		mockAPI.EXPECT().
			ListObjectsPaginator(gomock.Eq(bucket), gomock.Eq(""), gomock.Eq("")).
			Times(1).
			DoAndReturn(func(_, _, startAfterKey string) s3Pager {
				require.Equal(t, "", startAfterKey)
				return mockPager
			})

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
			GetObject(gomock.Any(), gomock.Eq(""), gomock.Eq(bucket), gomock.Eq("key1")).
			Return(nil, errFakeConnectivityFailure)

		s3ObjProc := newS3ObjectProcessorFactory(nil, mockAPI, nil, backupConfig{}, logp.NewNopLogger())

		cfg := config{
			NumberOfWorkers:         numberOfWorkers,
			BucketListInterval:      pollInterval,
			BucketARN:               bucket,
			BucketListPrefix:        "",
			RegionName:              "region",
			LexicographicalOrdering: false,
		}
		log := logp.NewLogger(inputName)
		poller := &s3PollerInput{
			log:             log,
			config:          cfg,
			s3:              mockAPI,
			pipeline:        pipeline,
			s3ObjectHandler: s3ObjProc,
			registry:        registry,
			provider:        "provider",
			metrics:         newInputMetrics(monitoring.NewRegistry(), 0, logp.NewNopLogger()),
			filterProvider:  newFilterProvider(&cfg),
			strategy:        newPollingStrategy(cfg.LexicographicalOrdering, log),
			status:          &statusReporterHelperMock{},
		}
		poller.runPoll(ctx)
	})

	t.Run("s3ObjectsListedPerRun metric is updated", func(t *testing.T) {
		store := openTestStatestore()

		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()

		ctrl, ctx := gomock.WithContext(ctx, t)
		defer ctrl.Finish()
		mockAPI := NewMockS3API(ctrl)
		mockPager := NewMockS3Pager(ctrl)
		pipeline := newFakePipeline()

		registry, err := newStateRegistry(nil, store, "", false, 0)
		require.NoError(t, err, "registry creation must succeed")

		// Expect ListObjectsPaginator to be called
		mockAPI.EXPECT().
			ListObjectsPaginator(gomock.Eq(bucket), gomock.Eq(""), gomock.Any()).
			Times(1).
			DoAndReturn(func(_, _, _ string) s3Pager {
				return mockPager
			})

		// First page with 3 objects
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
					},
				}, nil
			})

		// Second page with 2 more objects
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

		// No more pages
		mockPager.EXPECT().
			HasMorePages().
			Times(1).
			DoAndReturn(func() bool {
				return false
			})

		// Mock GetObject calls for all 5 objects
		mockAPI.EXPECT().
			GetObject(gomock.Any(), gomock.Eq(""), gomock.Eq(bucket), gomock.Eq("key1")).
			Return(nil, errFakeConnectivityFailure)
		mockAPI.EXPECT().
			GetObject(gomock.Any(), gomock.Eq(""), gomock.Eq(bucket), gomock.Eq("key2")).
			Return(nil, errFakeConnectivityFailure)
		mockAPI.EXPECT().
			GetObject(gomock.Any(), gomock.Eq(""), gomock.Eq(bucket), gomock.Eq("key3")).
			Return(nil, errFakeConnectivityFailure)
		mockAPI.EXPECT().
			GetObject(gomock.Any(), gomock.Eq(""), gomock.Eq(bucket), gomock.Eq("key4")).
			Return(nil, errFakeConnectivityFailure)
		mockAPI.EXPECT().
			GetObject(gomock.Any(), gomock.Eq(""), gomock.Eq(bucket), gomock.Eq("key5")).
			Return(nil, errFakeConnectivityFailure)

		s3ObjProc := newS3ObjectProcessorFactory(nil, mockAPI, nil, backupConfig{}, logp.NewNopLogger())

		// Create metrics and keep a reference to check later
		inputMetrics := newInputMetrics(monitoring.NewRegistry(), 0, logp.NewNopLogger())

		cfg := config{
			NumberOfWorkers:    numberOfWorkers,
			BucketListInterval: pollInterval,
			BucketARN:          bucket,
			BucketListPrefix:   "",
			RegionName:         "region",
		}
		log := logp.NewLogger(inputName)
		poller := &s3PollerInput{
			log:             log,
			config:          cfg,
			s3:              mockAPI,
			pipeline:        pipeline,
			s3ObjectHandler: s3ObjProc,
			registry:        registry,
			provider:        "provider",
			metrics:         inputMetrics,
			filterProvider:  newFilterProvider(&cfg),
			strategy:        newPollingStrategy(cfg.LexicographicalOrdering, log),
			status:          &statusReporterHelperMock{},
		}

		// Verify initial state of metric
		require.Equal(t, int64(0), inputMetrics.s3ObjectsListedPerRun.Count(), "s3ObjectsListedPerRun should start at 0")

		poller.runPoll(ctx)

		// Verify the metric was updated with the correct count (5 objects total: 3 + 2)
		require.Equal(t, int64(1), inputMetrics.s3ObjectsListedPerRun.Count(), "s3ObjectsListedPerRun should have 1 sample")
		require.Equal(t, int64(5), inputMetrics.s3ObjectsListedPerRun.Sum(), "s3ObjectsListedPerRun sum should be 5")
	})
}

func Test_S3StateHandling(t *testing.T) {
	bucket := "bucket"
	logger := logp.NewLogger(inputName)
	fixedTimeNow := time.Now()

	tests := []struct {
		name           string
		s3Objects      []types.Object
		config         *config
		initStates     []state
		runPollFor     int
		expectStateIDs []string
	}{
		{
			name: "State unchanged - registry backed state",
			s3Objects: []types.Object{
				{
					Key:          aws.String("obj-A"),
					ETag:         aws.String("etag-A"),
					LastModified: aws.Time(time.Unix(1732622400, 0)), // 2024-11-26T12:00:00Z
				},
			},
			config: &config{
				NumberOfWorkers:    1,
				BucketListInterval: 1 * time.Second,
				BucketARN:          bucket,
			},
			initStates:     []state{newState(bucket, "obj-A", "etag-A", time.Unix(1732622400, 0))}, // 2024-11-26T12:00:00Z
			runPollFor:     1,
			expectStateIDs: []string{stateID(bucket, "obj-A", "etag-A", time.Unix(1732622400, 0), false)}, // 2024-11-26T12:00:00Z
		},
		{
			name: "State cleanup - remove existing registry entry based on ignore older filter",
			s3Objects: []types.Object{
				{
					Key:          aws.String("obj-A"),
					ETag:         aws.String("etag-A"),
					LastModified: aws.Time(time.Unix(1732622400, 0)), // 2024-11-26T12:00:00Z
				},
			},
			config: &config{
				NumberOfWorkers:    1,
				BucketListInterval: 1 * time.Second,
				BucketARN:          bucket,
				IgnoreOlder:        1 * time.Second,
			},
			initStates:     []state{newState(bucket, "obj-A", "etag-A", time.Unix(1732622400, 0))}, // 2024-11-26T12:00:00Z
			runPollFor:     1,
			expectStateIDs: []string{},
		},
		{
			name: "State cleanup - remove existing registry entry based on timestamp filter",
			s3Objects: []types.Object{
				{
					Key:          aws.String("obj-A"),
					ETag:         aws.String("etag-A"),
					LastModified: aws.Time(time.Unix(1732622400, 0)), // 2024-11-26T12:00:00Z
				},
			},
			config: &config{
				NumberOfWorkers:    1,
				BucketListInterval: 1 * time.Second,
				BucketARN:          bucket,
				StartTimestamp:     "2024-11-27T12:00:00Z",
			},
			initStates:     []state{newState(bucket, "obj-A", "etag-A", time.Unix(1732622400, 0))}, // 2024-11-26T12:00:00Z
			runPollFor:     1,
			expectStateIDs: []string{},
		},
		{
			name: "State updated - no filters",
			s3Objects: []types.Object{
				{
					Key:          aws.String("obj-A"),
					ETag:         aws.String("etag-A"),
					LastModified: aws.Time(time.Unix(1732622400, 0)), // 2024-11-26T12:00:00Z
				},
			},
			config: &config{
				NumberOfWorkers:    1,
				BucketListInterval: 1 * time.Second,
				BucketARN:          bucket,
			},
			runPollFor:     1,
			expectStateIDs: []string{stateID(bucket, "obj-A", "etag-A", time.Unix(1732622400, 0), false)}, // 2024-11-26T12:00:00Z
		},
		{
			name: "State updated - ignore old filter",
			s3Objects: []types.Object{
				{
					Key:          aws.String("obj-A"),
					ETag:         aws.String("etag-A"),
					LastModified: aws.Time(fixedTimeNow),
				},
			},
			config: &config{
				NumberOfWorkers:    1,
				BucketListInterval: 1 * time.Second,
				BucketARN:          bucket,
				IgnoreOlder:        1 * time.Hour,
			},
			runPollFor:     1,
			expectStateIDs: []string{stateID(bucket, "obj-A", "etag-A", fixedTimeNow, false)},
		},
		{
			name: "State updated - timestamp filter",
			s3Objects: []types.Object{
				{
					Key:          aws.String("obj-A"),
					ETag:         aws.String("etag-A"),
					LastModified: aws.Time(fixedTimeNow),
				},
			},
			config: &config{
				NumberOfWorkers:    1,
				BucketListInterval: 1 * time.Second,
				BucketARN:          bucket,
				StartTimestamp:     "2024-11-26T12:00:00Z",
			},
			runPollFor:     1,
			expectStateIDs: []string{stateID(bucket, "obj-A", "etag-A", fixedTimeNow, false)},
		},
		{
			name: "State updated - combined filters of ignore old and timestamp entry exist after first run",
			s3Objects: []types.Object{
				{
					Key:          aws.String("obj-A"),
					ETag:         aws.String("etag-A"),
					LastModified: aws.Time(time.Unix(1732622400, 0)), // 2024-11-26T12:00:00Z
				},
			},
			config: &config{
				NumberOfWorkers:    1,
				BucketListInterval: 1 * time.Second,
				BucketARN:          bucket,
				IgnoreOlder:        1 * time.Hour,
				StartTimestamp:     "2024-11-20T12:00:00Z",
			},
			// run once to validate initial coverage of entries till start timestamp
			runPollFor:     1,
			expectStateIDs: []string{stateID(bucket, "obj-A", "etag-A", time.Unix(1732622400, 0), false)}, // 2024-11-26T12:00:00Z
		},
		{
			name: "State updated - combined filters of ignore old and timestamp remove entry after second run",
			s3Objects: []types.Object{
				{
					Key:          aws.String("obj-A"),
					ETag:         aws.String("etag-A"),
					LastModified: aws.Time(time.Unix(1732622400, 0)), // 2024-11-26T12:00:00Z
				},
			},
			config: &config{
				NumberOfWorkers:    1,
				BucketListInterval: 1 * time.Second,
				BucketARN:          bucket,
				IgnoreOlder:        1 * time.Hour,
				StartTimestamp:     "2024-11-20T12:00:00Z",
			},
			// run twice to validate removal by ignore old filter
			runPollFor:     2,
			expectStateIDs: []string{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// given - setup and mocks
			ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
			defer cancel()

			ctrl, ctx := gomock.WithContext(ctx, t)
			defer ctrl.Finish()

			mockS3API := NewMockS3API(ctrl)
			mockS3Pager := NewMockS3Pager(ctrl)
			mockObjHandler := NewMockS3ObjectHandlerFactory(ctrl)
			mockS3ObjectHandler := NewMockS3ObjectHandler(ctrl)

			gomock.InOrder(
				mockS3API.EXPECT().
					ListObjectsPaginator(gomock.Eq(bucket), gomock.Eq(""), gomock.Any()).
					AnyTimes().
					DoAndReturn(func(_, _, _ string) s3Pager {
						return mockS3Pager
					}),
			)

			for i := 0; i < test.runPollFor; i++ {
				mockS3Pager.EXPECT().HasMorePages().Times(1).DoAndReturn(func() bool { return true })
				mockS3Pager.EXPECT().HasMorePages().Times(1).DoAndReturn(func() bool { return false })
			}

			mockS3Pager.EXPECT().
				NextPage(gomock.Any()).
				Times(test.runPollFor).
				DoAndReturn(func(_ context.Context, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
					return &s3.ListObjectsV2Output{Contents: test.s3Objects}, nil
				})

			mockObjHandler.EXPECT().Create(gomock.Any(), gomock.Any()).AnyTimes().Return(mockS3ObjectHandler)
			mockS3ObjectHandler.EXPECT().ProcessS3Object(gomock.Any(), gomock.Any()).AnyTimes().
				DoAndReturn(func(log *logp.Logger, eventCallback func(e beat.Event)) error {
					eventCallback(beat.Event{})
					return nil
				})

			store := openTestStatestore()
			s3Registry, err := newStateRegistry(logger, store, "", false, 0)
			require.NoError(t, err, "Registry creation must succeed")

			// Note - add init states as if we are deriving them from registry
			for _, st := range test.initStates {
				err := s3Registry.AddState(st)
				require.NoError(t, err, "State add should not error")
			}

			poller := &s3PollerInput{
				log:             logger,
				config:          *test.config,
				s3:              mockS3API,
				pipeline:        newFakePipeline(),
				s3ObjectHandler: mockObjHandler,
				registry:        s3Registry,
				metrics:         newInputMetrics(monitoring.NewRegistry(), 0, logp.NewNopLogger()),
				filterProvider:  newFilterProvider(test.config),
				strategy:        newPollingStrategy(test.config.LexicographicalOrdering, logger),
				status:          &statusReporterHelperMock{},
			}

			// when - run polling for desired time
			for i := 0; i < test.runPollFor; i++ {
				poller.runPoll(ctx)
				<-time.After(500 * time.Millisecond)
			}

			// then - desired state entries

			// state must only contain expected state IDs
			normalRegistry, ok := s3Registry.(*normalStateRegistry)
			require.True(t, ok, "expected normalStateRegistry type")
			normalRegistry.statesLock.Lock()
			statesLen := len(normalRegistry.states)
			var missingIDs []string
			for _, id := range test.expectStateIDs {
				if normalRegistry.states[id] == nil {
					missingIDs = append(missingIDs, id)
				}
			}
			normalRegistry.statesLock.Unlock()

			require.Equal(t, len(test.expectStateIDs), statesLen)
			for _, id := range missingIDs {
				t.Errorf("state with ID %s should exist", id)
			}
		})
	}
}

func TestCreateS3API(t *testing.T) {
	t.Run("non-AWS bucket with configured region", func(t *testing.T) {
		ctx := context.Background()
		cfg := config{
			NonAWSBucketName: "my-bucket",
			RegionName:       "cn-shenzhen",
		}
		input := &s3PollerInput{
			config:    cfg,
			awsConfig: aws.Config{},
		}

		api, err := input.createS3API(ctx)
		require.NoError(t, err)
		require.NotNil(t, api)
		require.Equal(t, cfg.RegionName, input.awsConfig.Region)
	})

	// "non-AWS bucket without region" is invalid and rejected by config.Validate();
	t.Run("non-AWS bucket with configured region, aws region and configured region mismatch", func(t *testing.T) {
		ctx := context.Background()
		cfg := config{
			NonAWSBucketName: "my-bucket",
			RegionName:       "us-west-2",
		}
		input := &s3PollerInput{
			config: cfg,
			awsConfig: aws.Config{
				Region: "ap-southeast-1",
			},
		}

		api, err := input.createS3API(ctx)
		require.NoError(t, err)
		require.NotNil(t, api)
		require.Equal(t, cfg.RegionName, input.awsConfig.Region)
	})

	t.Run("access point ARN extracts region correctly", func(t *testing.T) {
		ctx := context.Background()
		cfg := config{
			AccessPointARN: "arn:aws:s3:eu-west-1:1234567890:accesspoint/my-bucket",
		}
		input := &s3PollerInput{
			config:    cfg,
			awsConfig: aws.Config{},
		}

		api, err := input.createS3API(ctx)
		require.NoError(t, err)
		require.NotNil(t, api)
		// Region should be extracted from ARN
		require.Equal(t, "eu-west-1", input.awsConfig.Region)
	})
}
