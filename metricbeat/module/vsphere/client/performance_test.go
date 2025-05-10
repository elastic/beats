// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package client

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vmware/govmomi/performance"
	"github.com/vmware/govmomi/vim25/types"
	"go.uber.org/mock/gomock"

	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

// Run 'go generate' to create mocks that are used in tests.
//go:generate go run go.uber.org/mock/mockgen -source=performance.go -destination=mock_performance.go -package client -mock_names=Logouter=MockPerfManager

func TestGetPerfMetrics(t *testing.T) {
	var tPeriod int32 = 5
	tObjType := "test-type"
	tObjName := "test-name"
	tObjRef := types.ManagedObjectReference{Type: tObjType, Value: "test-obj-ref-value", ServerGUID: "test-guid-1"}
	metrics := map[string]*types.PerfCounterInfo{
		"metric1": {
			Key: 111,
		},
	}
	metricSet := map[string]struct{}{
		"metric1": {},
	}

	mSamples := []types.BasePerfEntityMetricBase{
		&types.PerfEntityMetric{
			Value: []types.BasePerfMetricSeries{
				&types.PerfMetricSeries{
					Id: types.PerfMetricId{
						CounterId: 111,
						Instance:  "*",
					},
				},
			},
		},
	}

	tests := []struct {
		name            string
		mockPerfManager func(manager *MockPerfManager)
		assertResults   func(t2 *testing.T, metricsMap map[string]interface{}, err error)
	}{
		{
			name: "success (percentage metric)",
			mockPerfManager: func(manager *MockPerfManager) {
				manager.EXPECT().AvailableMetric(gomock.Any(), tObjRef, tPeriod).Return([]types.PerfMetricId{
					{
						CounterId: 111,
						Instance:  "*",
					},
				}, nil)
				manager.EXPECT().Query(gomock.Any(), []types.PerfQuerySpec{
					{
						Entity: tObjRef,
						MetricId: []types.PerfMetricId{
							{
								CounterId: 111,
								Instance:  "*",
							},
						},
						MaxSample:  1,
						IntervalId: tPeriod,
					},
				}).Return(mSamples, nil)
				manager.EXPECT().ToMetricSeries(gomock.Any(), mSamples).Return([]performance.EntityMetric{
					{
						Entity: tObjRef,
						Value: []performance.MetricSeries{
							{
								Name:  "metric1",
								Unit:  string(types.PerformanceManagerUnitPercent),
								Value: []int64{5320},
							},
						},
					},
				}, nil)
			},
			assertResults: func(t2 *testing.T, metricMap map[string]interface{}, err error) {
				require.NoError(t, err)
				assert.InDeltaMapValues(t, map[string]interface{}{
					"metric1": 53,
				}, metricMap, 0)
			},
		},
		{
			name: "success (not percentage metric)",
			mockPerfManager: func(manager *MockPerfManager) {
				manager.EXPECT().AvailableMetric(gomock.Any(), tObjRef, tPeriod).Return([]types.PerfMetricId{
					{
						CounterId: 111,
						Instance:  "*",
					},
				}, nil)
				manager.EXPECT().Query(gomock.Any(), []types.PerfQuerySpec{
					{
						Entity: tObjRef,
						MetricId: []types.PerfMetricId{
							{
								CounterId: 111,
								Instance:  "*",
							},
						},
						MaxSample:  1,
						IntervalId: tPeriod,
					},
				}).Return(mSamples, nil)
				manager.EXPECT().ToMetricSeries(gomock.Any(), mSamples).Return([]performance.EntityMetric{
					{
						Entity: tObjRef,
						Value: []performance.MetricSeries{
							{
								Name:  "metric1",
								Unit:  string(types.PerformanceManagerUnitKiloBytesPerSecond),
								Value: []int64{1024},
							},
						},
					},
				}, nil)
			},
			assertResults: func(t2 *testing.T, metricMap map[string]interface{}, err error) {
				require.NoError(t, err)
				assert.InDeltaMapValues(t, map[string]interface{}{
					"metric1": 1024,
				}, metricMap, 0)
			},
		},
		{
			name: "no available metrics",
			mockPerfManager: func(manager *MockPerfManager) {
				manager.EXPECT().AvailableMetric(gomock.Any(), tObjRef, tPeriod).Return(nil, nil)
				manager.EXPECT().Query(gomock.Any(), []types.PerfQuerySpec{
					{
						Entity:     tObjRef,
						MetricId:   nil,
						MaxSample:  1,
						IntervalId: tPeriod,
					},
				}).Return(nil, nil)
			},
			assertResults: func(t2 *testing.T, metricMap map[string]interface{}, err error) {
				require.NoError(t, err)
				assert.InDeltaMapValues(t, map[string]interface{}{}, metricMap, 0)
			},
		},
		{
			name: "query error",
			mockPerfManager: func(manager *MockPerfManager) {
				manager.EXPECT().AvailableMetric(gomock.Any(), tObjRef, tPeriod).Return([]types.PerfMetricId{
					{
						CounterId: 111,
						Instance:  "*",
					},
				}, nil)
				manager.EXPECT().Query(gomock.Any(), []types.PerfQuerySpec{
					{
						Entity: tObjRef,
						MetricId: []types.PerfMetricId{
							{
								CounterId: 111,
								Instance:  "*",
							},
						},
						MaxSample:  1,
						IntervalId: tPeriod,
					},
				}).Return(nil, errors.New("query error"))
			},
			assertResults: func(t *testing.T, metricMap map[string]interface{}, err error) {
				assert.Error(t, err, "query error")
			},
		},
		{
			name: "error converting to metric series",
			mockPerfManager: func(manager *MockPerfManager) {
				manager.EXPECT().AvailableMetric(gomock.Any(), tObjRef, tPeriod).Return([]types.PerfMetricId{
					{
						CounterId: 111,
						Instance:  "*",
					},
				}, nil)
				manager.EXPECT().Query(gomock.Any(), []types.PerfQuerySpec{
					{
						Entity: tObjRef,
						MetricId: []types.PerfMetricId{
							{
								CounterId: 111,
								Instance:  "*",
							},
						},
						MaxSample:  1,
						IntervalId: tPeriod,
					},
				}).Return(mSamples, nil)
				manager.EXPECT().ToMetricSeries(gomock.Any(), mSamples).Return(nil, errors.New("ToMetricSeries error"))
			},
			assertResults: func(t2 *testing.T, metricMap map[string]interface{}, err error) {
				assert.Error(t, err, "ToMetricSeries error")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			mPerfManager := NewMockPerfManager(ctrl)
			if tt.mockPerfManager != nil {
				tt.mockPerfManager(mPerfManager)
			}

			manager := &PerformanceDataFetcher{
				logger:      logptest.NewTestingLogger(t, ""),
				perfManager: mPerfManager,
			}

			metricMap, err := manager.GetPerfMetrics(context.Background(), tPeriod, tObjType, tObjName, tObjRef, metrics, metricSet)
			tt.assertResults(t, metricMap, err)
		})
	}
}
