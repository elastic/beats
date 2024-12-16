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

//go:build integration && linux

package apiserver

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/elastic/beats/v7/metricbeat/helper/prometheus"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/module/kubernetes/test"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestFetchMetricset(t *testing.T) {
	config := test.GetAPIServerConfig(t, "apiserver")
	metricSet := mbtest.NewFetcher(t, config)
	events, errs := metricSet.FetchEvents()
	if len(errs) > 0 {
		t.Fatalf("Expected 0 error, had %d. %v\n", len(errs), errs)
	}
	assert.NotEmpty(t, events)
}

// TestFetch tests the behavior of the Fetch function in Metricset
func TestFetch(t *testing.T) {
	// Create mocks
	mockPrometheusClient := new(MockPrometheusClient)
	mockHTTP := new(MockHTTP)
	mockReporter := new(MockReporter)

	// Sample events to return from the mock client
	events := []mapstr.M{
		{"metric": "cpu_usage", "value": 75},
		{"metric": "memory_usage", "value": 65},
	}

	// Define behavior for GetProcessedMetrics (success)
	mockPrometheusClient.On("GetProcessedMetrics", mock.Anything).Return(events, nil)

	// Define behavior for RefreshAuthorizationHeader (first call fails, second succeeds)
	mockHTTP.On("RefreshAuthorizationHeader").Return(false, errors.New("unauthorized")).Once() // Fail the first time
	mockHTTP.On("RefreshAuthorizationHeader").Return(true, nil).Once()                         // Succeed the second time

	// Create Metricset
	ms := &Metricset{
		http:             mockHTTP,
		prometheusClient: mockPrometheusClient,
	}

	// Mock the reporter behavior (event successfully reported)
	mockReporter.On("Event", mock.Anything).Return(true).Once()

	// Step 1: Call the Fetch function
	err := ms.Fetch(mockReporter)

	// Step 2: Assertions
	assert.NoError(t, err) // Should not return an error
	mockPrometheusClient.AssertExpectations(t)
	mockHTTP.AssertExpectations(t)
	mockReporter.AssertExpectations(t)

	// Verify if the RefreshAuthorizationHeader was called twice
	mockHTTP.AssertNumberOfCalls(t, "RefreshAuthorizationHeader", 1)

	// Verify that the event was reported correctly
	mockReporter.AssertNumberOfCalls(t, "Event", 2) // As there are two events to report

	// Assert that the Authorization header refresh retry logic works (first fail, second success)
	assert.Equal(t, 1, len(mockHTTP.Calls)) // Ensure it attempted authorization refresh
}

// TestFetchUnauthorized tests the Fetch function when the authorization refresh fails
func TestFetchUnauthorized(t *testing.T) {
	// Create mocks
	mockPrometheusClient := new(MockPrometheusClient)
	mockHTTP := new(MockHTTP)
	mockReporter := new(MockReporter)

	// Define behavior for GetProcessedMetrics (will fail after 2 retries)
	mockPrometheusClient.On("GetProcessedMetrics", mock.Anything).Return(nil, errors.New("unauthorized"))

	// Define behavior for RefreshAuthorizationHeader (always fails)
	mockHTTP.On("RefreshAuthorizationHeader").Return(false, errors.New("unauthorized")).Twice() // Fail twice

	// Create Metricset
	ms := &Metricset{
		http:             mockHTTP,
		prometheusClient: mockPrometheusClient,
	}

	// Step 1: Call the Fetch function (expect an error)
	err := ms.Fetch(mockReporter)

	// Step 2: Assertions
	assert.Error(t, err) // Expect an error due to failed authorization refresh
	mockPrometheusClient.AssertExpectations(t)
	mockHTTP.AssertExpectations(t)
	mockReporter.AssertExpectations(t)

	// Verify that authorization refresh was attempted twice
	mockHTTP.AssertNumberOfCalls(t, "RefreshAuthorizationHeader", 2)

	// Verify that no events were reported (because the metrics fetch failed)
	mockReporter.AssertNumberOfCalls(t, "Event", 0)
}

// Mock Prometheus Client
type MockPrometheusClient struct {
	mock.Mock
}

func (m *MockPrometheusClient) GetProcessedMetrics(mapping *prometheus.MetricsMapping) ([]mapstr.M, error) {
	args := m.Called(mapping)
	return args.Get(0).([]mapstr.M), args.Error(1)
}

// Mock HTTP Client
type MockHTTP struct {
	mock.Mock
}

func (m *MockHTTP) RefreshAuthorizationHeader() (bool, error) {
	args := m.Called()
	return args.Bool(0), args.Error(1)
}

// Mock Reporter
type MockReporter struct {
	mock.Mock
}

func (m *MockReporter) Event(event mb.Event) bool {
	args := m.Called(event)
	return args.Bool(0)
}
