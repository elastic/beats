// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cloudwatch

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	awsmiddleware "github.com/aws/aws-sdk-go-v2/aws/middleware"
	cloudwatchsdk "github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	cloudwatchtypes "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/smithy-go"
	"github.com/aws/smithy-go/middleware"
	smithyhttp "github.com/aws/smithy-go/transport/http"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/aws"
	"github.com/elastic/elastic-agent-libs/monitoring"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestCloudwatchMetrics() (*cloudwatchMetrics, *monitoring.Registry) {
	reg := monitoring.NewRegistry()
	return newCloudwatchMetrics(reg, &aws.MetricSet{Period: time.Minute, Latency: 4 * time.Minute, DataGranularity: time.Minute, IncludeLinkedAccounts: true, RegionsList: []string{"us-west-1"}}, []Config{{Namespace: "AWS/EC2"}, {Namespace: "AWS/SQS"}}), reg
}

func TestCloudwatchCollectionProgress(t *testing.T) {
	m, reg := newTestCloudwatchMetrics()
	start, end := aws.GetStartTimeEndTime(time.Now(), time.Minute, 4*time.Minute, time.Time{})
	previous := m.beginWindow(start.Add(-10*time.Minute), end.Add(-10*time.Minute))
	m.observeScope("us-west-1", "AWS/EC2", true, time.Now(), 0, errors.New("access denied"))
	m.endWindow(previous, nil)
	snapshot := monitoring.CollectStructSnapshot(reg, monitoring.Full, false)["cloudwatch"].(map[string]any)
	assert.EqualValues(t, 1, snapshot["incomplete_fetches_total"], "logged errors must prevent a fully successful window")
	assert.GreaterOrEqual(t, snapshot["collection_lag_ms"], int64((10 * time.Minute).Milliseconds()), "lag must exclude configured latency and alignment")
	assert.NotContains(t, snapshot, "last_successful_window_end", "attempted windows must not be advertised as successful")
	assert.Len(t, snapshot["scopes"], 3, "configured namespaces with no events must remain visible alongside the wildcard scope")
	previous = m.beginWindow(start, end)
	m.observeScope("us-west-1", "AWS/SQS", false, time.Now(), 0, nil)
	m.endWindow(previous, nil)
	snapshot = monitoring.CollectStructSnapshot(reg, monitoring.Full, false)["cloudwatch"].(map[string]any)
	assert.EqualValues(t, end.UnixMilli(), snapshot["last_successful_window_end"], "successful empty collections must advance successful progress")
}

func TestCloudwatchAPIMetrics(t *testing.T) {
	for _, tc := range []struct {
		name              string
		err               error
		errors, throttles int64
	}{
		{name: "success"},
		{name: "failure", err: errors.New("transport failure"), errors: 1},
		{name: "throttle", err: &smithy.GenericAPIError{Code: "ThrottlingException", Message: "slow down"}, errors: 1, throttles: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m, reg := newTestCloudwatchMetrics()
			stack := middleware.NewStack("test", smithyhttp.NewStackRequest)
			require.NoError(t, m.instrument(stack), "metrics middleware must install")
			ctx := awsmiddleware.SetOperationName(context.Background(), "ListMetrics")
			want := "unchanged result"
			result, _, err := stack.Initialize.HandleMiddleware(ctx, nil, middleware.HandlerFunc(func(context.Context, any) (any, middleware.Metadata, error) {
				return want, middleware.Metadata{}, tc.err
			}))
			assert.Equal(t, want, result, "instrumentation must preserve the SDK result")
			assert.Equal(t, tc.err, err, "instrumentation must preserve the SDK error")
			snapshot := monitoring.CollectStructSnapshot(reg, monitoring.Full, false)["cloudwatch"].(map[string]any)
			api := snapshot["api"].(map[string]any)["ListMetrics"].(map[string]any)
			assert.EqualValues(t, 1, api["calls_total"], "one SDK invocation must count once")
			assert.EqualValues(t, tc.errors, api["errors_total"], "API errors must be classified")
			assert.EqualValues(t, tc.throttles, api["throttles_total"], "final throttling errors must be classified")
			assert.EqualValues(t, 1, api["duration_ns_count"], "one SDK invocation must contribute one duration")
		})
	}
}

func TestCloudwatchIncompleteResults(t *testing.T) {
	m, reg := newTestCloudwatchMetrics()
	previous := m.beginWindow(time.Now().Add(-time.Minute), time.Now())
	m.observeResults([]cloudwatchtypes.MetricDataResult{{StatusCode: cloudwatchtypes.StatusCodePartialData, Values: []float64{1, 2}}})
	m.endWindow(previous, nil)
	snapshot := monitoring.CollectStructSnapshot(reg, monitoring.Full, false)["cloudwatch"].(map[string]any)
	assert.EqualValues(t, 2, snapshot["datapoints_received_total"], "partial datapoints must be counted")
	assert.EqualValues(t, 1, snapshot["incomplete_results_total"], "partial AWS results must be visible despite a successful transport response")
	assert.NotContains(t, snapshot, "last_successful_window_end", "partial results must not mark a fully successful window")
}

func TestCloudwatchConcurrentSnapshots(t *testing.T) {
	m, reg := newTestCloudwatchMetrics()
	var wg sync.WaitGroup
	for range 4 {
		wg.Go(func() {
			for range 50 {
				m.observeScope("us-west-1", "AWS/EC2", false, time.Now(), 1, nil)
				monitoring.CollectStructSnapshot(reg, monitoring.Full, false)
			}
		})
	}
	wg.Wait()
	assert.EqualValues(t, 200, m.scopes[scopeKey{"us-west-1", "AWS/EC2"}].events, "snapshots must not lose concurrent observations")
}

func TestCloudwatchSDKInstrumentation(t *testing.T) {
	m, reg := newTestCloudwatchMetrics()
	want := &cloudwatchsdk.ListMetricsOutput{}
	client := cloudwatchsdk.NewFromConfig(awssdk.Config{Region: "us-west-1", Credentials: awssdk.AnonymousCredentials{}}, func(options *cloudwatchsdk.Options) {
		options.APIOptions = append(options.APIOptions, m.instrument, func(stack *middleware.Stack) error {
			return stack.Initialize.Add(middleware.InitializeMiddlewareFunc("StubListMetrics", func(context.Context, middleware.InitializeInput, middleware.InitializeHandler) (middleware.InitializeOutput, middleware.Metadata, error) {
				return middleware.InitializeOutput{Result: want}, middleware.Metadata{}, nil
			}), middleware.After)
		})
	})
	got, err := client.ListMetrics(context.Background(), &cloudwatchsdk.ListMetricsInput{})
	require.NoError(t, err, "instrumentation must allow a successful SDK call")
	assert.Same(t, want, got, "instrumentation must preserve the SDK's typed result")
	snapshot := monitoring.CollectStructSnapshot(reg, monitoring.Full, false)["cloudwatch"].(map[string]any)
	api := snapshot["api"].(map[string]any)
	require.Contains(t, api, "ListMetrics", "the real SDK must expose its operation name to the metrics middleware")
	assert.EqualValues(t, 1, api["ListMetrics"].(map[string]any)["calls_total"], "one ListMetrics call must count once")
}
