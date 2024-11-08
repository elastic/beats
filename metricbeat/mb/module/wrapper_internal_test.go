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

package module

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/management/status"
	"github.com/elastic/beats/v7/metricbeat/mb"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const mockModuleName = "MockModule"
const mockMetricSetName = "MockMetricSet"
const mockMetricSetWithContextName = "MockMetricSetWithContext"

// mockReportingFetcher
type mockReportingFetcher struct {
	mb.BaseMetricSet
	mock.Mock
}

func (mrf *mockReportingFetcher) Fetch(r mb.ReporterV2) error {
	args := mrf.Called(r)
	return args.Error(0)
}

// mockReportingFetcherWithContext
type mockReportingFetcherWithContext struct {
	mb.BaseMetricSet
	mock.Mock
}

func (mrf *mockReportingFetcherWithContext) Fetch(ctx context.Context, r mb.ReporterV2) error {
	args := mrf.Called(ctx, r)
	return args.Error(0)
}

// mockReporter
type mockReporter struct {
	mock.Mock
}

func (mr *mockReporter) StartFetchTimer() {
	mr.Called()
}

func (mr *mockReporter) V1() mb.PushReporter { //nolint:staticcheck // PushReporter is deprecated but not removed
	args := mr.Called()
	return args.Get(0).(mb.PushReporter)
}

func (mr *mockReporter) V2() mb.PushReporterV2 {
	args := mr.Called()
	return args.Get(0).(mb.PushReporterV2)
}

// mockPushReporterV2
type mockPushReporterV2 struct {
	mock.Mock
}

func (mpr *mockPushReporterV2) Event(event mb.Event) bool {
	args := mpr.Called(event)
	return args.Bool(0)
}

func (mpr *mockPushReporterV2) Error(err error) bool {
	args := mpr.Called(err)
	return args.Bool(0)
}

func (mpr *mockPushReporterV2) Done() <-chan struct{} {
	args := mpr.Called()
	return args.Get(0).(<-chan struct{})
}

// mockStatusReporterV2
type mockStatusReporter struct {
	mock.Mock
}

func (m *mockStatusReporter) UpdateStatus(status status.Status, msg string) {
	m.Called(status, msg)
}

func TestWrapperHandleFetchErrorSync(t *testing.T) {

	fetchError := errors.New("fetch has gone all wrong")

	type setupFunc func(t *testing.T, fetcher *mockReportingFetcher, pushReporter *mockPushReporterV2, statusReporter *mockStatusReporter)
	type postIterationAssertFunc func(t *testing.T, i int, msWrapper *metricSetWrapper, fetcher *mockReportingFetcher, pushReporter *mockPushReporterV2, statusReporter *mockStatusReporter)

	testcases := []struct {
		name            string
		config          *conf.C
		setup           setupFunc
		iterations      int
		assertIteration postIterationAssertFunc
	}{
		{
			name: "no failureThreshold: status DEGRADED after first error",
			config: newConfig(t, map[string]interface{}{
				"module":     mockModuleName,
				"metricsets": []string{mockMetricSetName},
				"period":     "100ms",
				"hosts":      []string{"testhost"},
			}),
			setup: func(t *testing.T, fetcher *mockReportingFetcher, pushReporter *mockPushReporterV2, statusReporter *mockStatusReporter) {
				// fetcher will immediately error out
				fetcher.On("Fetch", pushReporter).Return(fetchError).Once()
				// expect the error to be propagated via the pushReporter
				pushReporter.On("Error", fetchError).Return(true).Once()
				// expect the status degraded to be set
				statusReporter.On("UpdateStatus", status.Degraded, mock.AnythingOfType("string")).Once()
			},
			iterations:      1,
			assertIteration: nil,
		},
		{
			name: "no failureThreshold: status DEGRADED after first error, reset to Running after first successful fetch",
			config: newConfig(t, map[string]interface{}{
				"module":     mockModuleName,
				"metricsets": []string{mockMetricSetName},
				"period":     "100ms",
				"hosts":      []string{"testhost"},
			}),
			setup: func(t *testing.T, fetcher *mockReportingFetcher, pushReporter *mockPushReporterV2, statusReporter *mockStatusReporter) {
				// fetcher will immediately error out 3 times
				fetcher.On("Fetch", pushReporter).Return(fetchError).Times(3)
				// fetcher will never error again afterwards
				fetcher.On("Fetch", pushReporter).Return(nil)
				// expect the error to be propagated via the pushReporter
				pushReporter.On("Error", fetchError).Return(true).Times(3)
				// expect the status degraded to be set 3 times
				statusReporter.On("UpdateStatus", status.Degraded, mock.AnythingOfType("string")).Times(3)
				// expect the status Running to be set once fetch recovers
				statusReporter.On("UpdateStatus", status.Running, mock.AnythingOfType("string")).Twice()
			},
			iterations: 5,
			assertIteration: func(t *testing.T, i int, msWrapper *metricSetWrapper, fetcher *mockReportingFetcher, pushReporter *mockPushReporterV2, statusReporter *mockStatusReporter) {
				t.Logf("Assertion after iteration %d", i)
				switch {
				case i < 3:
					assert.Truef(t, statusReporter.AssertCalled(t, "UpdateStatus", status.Degraded, mock.AnythingOfType("string")), "stream degraded at iteration %d", i)
				case i >= 3:
					assert.Truef(t, statusReporter.AssertCalled(t, "UpdateStatus", status.Running, mock.AnythingOfType("string")), "stream set to running at iteration %d", i)
				}
			},
		},
		{
			name: "failureThreshold = 2: status DEGRADED at the 3rd error",
			config: newConfig(t, map[string]interface{}{
				"module":           mockModuleName,
				"metricsets":       []string{mockMetricSetName},
				"period":           "100ms",
				"hosts":            []string{"testhost"},
				"failureThreshold": 2,
			}),
			setup: func(t *testing.T, fetcher *mockReportingFetcher, pushReporter *mockPushReporterV2, statusReporter *mockStatusReporter) {
				// fetcher will immediately error out 3 times in a row
				fetcher.On("Fetch", pushReporter).Return(fetchError).Times(3)
				// expect the error to be propagated via the pushReporter at every iteration
				pushReporter.On("Error", fetchError).Return(true).Times(3)
				// expect the status degraded to be set
				statusReporter.On("UpdateStatus", status.Degraded, mock.AnythingOfType("string")).Once()
			},
			iterations: 3,
			assertIteration: func(t *testing.T, i int, msWrapper *metricSetWrapper, fetcher *mockReportingFetcher, pushReporter *mockPushReporterV2, statusReporter *mockStatusReporter) {
				t.Logf("Assertion after iteration %d", i)
				switch {
				case i < 2:
					assert.Truef(t, statusReporter.AssertNotCalled(t, "UpdateStatus", status.Degraded, mock.AnythingOfType("string")), "stream degraded at iteration %d", i)
				case i == 2:
					assert.Truef(t, statusReporter.AssertCalled(t, "UpdateStatus", status.Degraded, mock.AnythingOfType("string")), "stream not yet degraded at iteration %d", i)
				}
			},
		},
		{
			name: "failureThreshold = 2: status HEALTHY after 2 errors, 1 success and 2 more errors, DEGRADED at the 3rd consecutive error",
			config: newConfig(t, map[string]interface{}{
				"module":           mockModuleName,
				"metricsets":       []string{mockMetricSetName},
				"period":           "100ms",
				"hosts":            []string{"testhost"},
				"failureThreshold": 2,
			}),
			setup: func(t *testing.T, fetcher *mockReportingFetcher, pushReporter *mockPushReporterV2, statusReporter *mockStatusReporter) {
				// fetcher will error out 2 times in a row
				fetcher.On("Fetch", pushReporter).Return(fetchError).Times(2)
				// fetcher will then succeed once
				fetcher.On("Fetch", pushReporter).Return(nil).Once()
				// fetcher will error out 3 more times in a row
				fetcher.On("Fetch", pushReporter).Return(fetchError).Times(3)

				// expect the error to be propagated via the pushReporter at every failing iteration
				pushReporter.On("Error", fetchError).Return(true).Times(5)
				// expect the status running to be set when there's no error returned by the fetcher at the 3rd iteration
				statusReporter.On("UpdateStatus", status.Running, mock.AnythingOfType("string")).Once()
				// expect the status degraded to be set only once
				statusReporter.On("UpdateStatus", status.Degraded, mock.AnythingOfType("string")).Once()
			},
			iterations: 6,
			assertIteration: func(t *testing.T, i int, msWrapper *metricSetWrapper, fetcher *mockReportingFetcher, pushReporter *mockPushReporterV2, statusReporter *mockStatusReporter) {
				t.Logf("Assertion after iteration %d", i)
				switch {
				case i < 2:
					assert.Truef(t, statusReporter.AssertNotCalled(t, "UpdateStatus", status.Degraded, mock.AnythingOfType("string")), "stream degraded at iteration %d", i)
				case i >= 2 && i < 5:
					assert.Truef(t, statusReporter.AssertNotCalled(t, "UpdateStatus", status.Degraded, mock.AnythingOfType("string")), "stream degraded at iteration %d", i)
					assert.Truef(t, statusReporter.AssertCalled(t, "UpdateStatus", status.Running, mock.AnythingOfType("string")), "stream degraded at iteration %d", i)
				case i == 5:
					assert.Truef(t, statusReporter.AssertCalled(t, "UpdateStatus", status.Degraded, mock.AnythingOfType("string")), "stream not yet degraded at iteration %d", i)
				}
			},
		},
		{
			name: "failureThreshold = -1: stream status update never become DEGRADED",
			config: newConfig(t, map[string]interface{}{
				"module":           mockModuleName,
				"metricsets":       []string{mockMetricSetName},
				"period":           "100ms",
				"hosts":            []string{"testhost"},
				"failureThreshold": -1,
			}),
			setup: func(t *testing.T, fetcher *mockReportingFetcher, pushReporter *mockPushReporterV2, statusReporter *mockStatusReporter) {
				// fetcher will error out 9 times in a row
				fetcher.On("Fetch", pushReporter).Return(fetchError).Times(9)
				// fetcher will then succeed once
				fetcher.On("Fetch", pushReporter).Return(nil).Once()

				// expect the error to be propagated via the pushReporter at every failing iteration
				pushReporter.On("Error", fetchError).Return(true).Times(9)
				// expect the status running to be set when there's no error returned by the fetcher at the 10th iteration
				statusReporter.On("UpdateStatus", status.Running, mock.AnythingOfType("string")).Once()
			},
			iterations: 10,
			assertIteration: func(t *testing.T, i int, msWrapper *metricSetWrapper, fetcher *mockReportingFetcher, pushReporter *mockPushReporterV2, statusReporter *mockStatusReporter) {
				t.Logf("Assertion after iteration %d", i)
				switch {
				case i < 9:
					assert.Truef(t, statusReporter.AssertNotCalled(t, "UpdateStatus", status.Degraded, mock.AnythingOfType("string")), "stream degraded at iteration %d", i)
				case i == 9:
					assert.Truef(t, statusReporter.AssertNotCalled(t, "UpdateStatus", status.Degraded, mock.AnythingOfType("string")), "stream degraded at iteration %d", i)
					assert.Truef(t, statusReporter.AssertCalled(t, "UpdateStatus", status.Running, mock.AnythingOfType("string")), "stream degraded at iteration %d", i)
				}
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mock push reporter
			mpr := new(mockPushReporterV2)

			// Setup mock fetcher
			mrf := new(mockReportingFetcher)

			// Setup mock StatusReporter
			msr := new(mockStatusReporter)

			//Setup mock reporter (ensure proper handling of intermediate calls, no functional value here)
			mr := new(mockReporter)
			mr.On("StartFetchTimer").Return()
			mr.On("V2").Return(mpr)

			// assert mocks expectations
			t.Cleanup(func() {
				mock.AssertExpectationsForObjects(t, mrf, mr, mpr, msr)
			})

			// setup mocks before starting the test
			if tc.setup != nil {
				tc.setup(t, mrf, mpr, msr)
			}

			// add metricset in registry
			r := mb.NewRegister()
			err := r.AddMetricSet(mockModuleName, mockMetricSetName, func(base mb.BaseMetricSet) (mb.MetricSet, error) {
				mrf.BaseMetricSet = base
				return mrf, nil
			})
			require.NoError(t, err)

			aModule, metricSets, err := mb.NewModule(tc.config, r)
			require.NoError(t, err)

			// Set the mock status reporter
			aModule.SetStatusReporter(msr)

			moduleWrapper, err := NewWrapperForMetricSet(aModule, metricSets[0], WithMetricSetInfo())
			require.NoError(t, err)

			// run metricset synchronously
			wrappedMetricSet := moduleWrapper.MetricSets()[0]
			for i := 0; i < tc.iterations; i++ {
				wrappedMetricSet.fetch(context.TODO(), mr)
				if tc.assertIteration != nil {
					tc.assertIteration(t, i, wrappedMetricSet, mrf, mpr, msr)
				}
			}
		})
	}
}

func TestWrapperHandleFetchErrorAsync(t *testing.T) {

	t.Skip("This test runs a mock wrapped metricset asynchronously. Preferring the synchronous test for now")

	// Setup mock push reporter
	mpr := new(mockPushReporterV2)

	// Setup mock fetcher
	mrf := new(mockReportingFetcher)
	//mrf.On("Fetch", mpr).Return(nil)

	//Setup mock reporter
	mr := new(mockReporter)
	//mr.On("StartFetchTimer").Return()
	//mr.On("V2").Return(mpr)

	// assert mocks expectations
	t.Cleanup(func() {
		mock.AssertExpectationsForObjects(t, mrf, mr, mpr)
	})

	// add metricset in registry
	r := mb.NewRegister()
	err := r.AddMetricSet(mockModuleName, mockMetricSetName, func(base mb.BaseMetricSet) (mb.MetricSet, error) {
		mrf.BaseMetricSet = base
		return mrf, nil
	})
	require.NoError(t, err)

	hosts := []string{"testhost"}
	c := newConfig(t, map[string]interface{}{
		"module":     mockModuleName,
		"metricsets": []string{mockMetricSetName},
		"period":     "100ms",
		"hosts":      hosts,
		"health": map[string]interface{}{
			"enabled":          true,
			"failureThreshold": 2,
		},
	})

	aModule, metricSets, err := mb.NewModule(c, r)
	require.NoError(t, err)

	mWrapper, err := NewWrapperForMetricSet(aModule, metricSets[0], WithMetricSetInfo())
	require.NoError(t, err)

	require.Len(t, mWrapper.MetricSets(), 1)

	// run the metricset asynchronously
	done := make(chan struct{})
	output := mWrapper.Start(done)

	wg := new(sync.WaitGroup)
	outputConsumeLoop(t, wg, output, done, func(event beat.Event) {
		t.Logf("received event: %v", event)
	})
	time.Sleep(1 * time.Second)

	close(done)
	wg.Wait()
}

func outputConsumeLoop(t *testing.T, wg *sync.WaitGroup, output <-chan beat.Event, done chan struct{}, ehf eventHandlingTestFunc) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case e := <-output:
				ehf(e)
			case <-done:
				// finish consuming and return
				t.Log("test done, consuming remaining events")
				for e := range output {
					ehf(e)
				}
				t.Log("done consuming events")
				return
			}
		}
	}()
}

type eventHandlingTestFunc func(beat.Event)

func newConfig(t testing.TB, moduleConfig interface{}) *conf.C {
	config, err := conf.NewConfigFrom(moduleConfig)
	require.NoError(t, err)
	return config
}
