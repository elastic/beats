// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"context"
	"testing"
	"time"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
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
	logp.TestingSetup()

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

		s3ObjProc := newS3ObjectProcessorFactory(nil, mockAPI, nil, backupConfig{})
		states, err := newStates(nil, store, listPrefix)
		require.NoError(t, err, "states creation must succeed")

		cfg := config{
			NumberOfWorkers:    numberOfWorkers,
			BucketListInterval: pollInterval,
			BucketARN:          bucket,
			BucketListPrefix:   listPrefix,
			RegionName:         "region",
		}
		poller := &s3PollerInput{
			log:             logp.NewLogger(inputName),
			config:          cfg,
			s3:              mockAPI,
			pipeline:        pipeline,
			s3ObjectHandler: s3ObjProc,
			states:          states,
			provider:        "provider",
			metrics:         newInputMetrics(v2.Context{MetricsRegistry: monitoring.NewRegistry()}, 0),
			filterProvider:  newFilterProvider(&cfg),
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

		s3ObjProc := newS3ObjectProcessorFactory(nil, mockS3, nil, backupConfig{})
		states, err := newStates(nil, store, listPrefix)
		require.NoError(t, err, "states creation must succeed")

		cfg := config{
			NumberOfWorkers:    numberOfWorkers,
			BucketListInterval: pollInterval,
			BucketARN:          bucket,
			BucketListPrefix:   "key",
			RegionName:         "region",
		}

		v2ctx := v2.Context{MetricsRegistry: monitoring.NewRegistry()}
		poller := &s3PollerInput{
			log: logp.NewLogger(inputName),
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
			states:          states,
			provider:        "provider",
			metrics:         newInputMetrics(v2ctx, 0),
			filterProvider:  newFilterProvider(&cfg),
		}
		poller.run(ctx)
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
			expectStateIDs: []string{stateID(bucket, "obj-A", "etag-A", time.Unix(1732622400, 0))}, // 2024-11-26T12:00:00Z
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
			expectStateIDs: []string{stateID(bucket, "obj-A", "etag-A", time.Unix(1732622400, 0))}, // 2024-11-26T12:00:00Z
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
			expectStateIDs: []string{stateID(bucket, "obj-A", "etag-A", fixedTimeNow)},
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
			expectStateIDs: []string{stateID(bucket, "obj-A", "etag-A", fixedTimeNow)},
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
			expectStateIDs: []string{stateID(bucket, "obj-A", "etag-A", time.Unix(1732622400, 0))}, // 2024-11-26T12:00:00Z
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
					ListObjectsPaginator(gomock.Eq(bucket), "").
					AnyTimes().
					DoAndReturn(func(_, _ string) s3Pager {
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
			s3States, err := newStates(logger, store, "")
			require.NoError(t, err, "States creation must succeed")

			// Note - add init states as if we are deriving them from registry
			for _, st := range test.initStates {
				err := s3States.AddState(st)
				require.NoError(t, err, "State add should not error")
			}

			v2ctx := v2.Context{MetricsRegistry: monitoring.NewRegistry()}
			poller := &s3PollerInput{
				log:             logger,
				config:          *test.config,
				s3:              mockS3API,
				pipeline:        newFakePipeline(),
				s3ObjectHandler: mockObjHandler,
				states:          s3States,
				metrics:         newInputMetrics(v2ctx, 0),
				filterProvider:  newFilterProvider(test.config),
			}

			// when - run polling for desired time
			for i := 0; i < test.runPollFor; i++ {
				poller.runPoll(ctx)
				<-time.After(500 * time.Millisecond)
			}

			// then - desired state entries

			// state must only contain expected state IDs
			require.Equal(t, len(test.expectStateIDs), len(s3States.states))
			for _, id := range test.expectStateIDs {
				if s3States.states[id] == nil {
					t.Errorf("state with ID %s should exist", id)
				}
			}
		})
	}
}
