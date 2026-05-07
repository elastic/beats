// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !requirefips

package app_insights

import (
	"github.com/stretchr/testify/mock"

	"github.com/elastic/elastic-agent-libs/logp"
)

// Service is the abstraction over the Application Insights metrics API used
// by Client. A mock is provided for unit tests.
type Service interface {
	GetMetricValues(applicationId string, bodyMetrics []MetricsBatchRequestItem) (ListMetricsResultsItem, error)
}

// MockService mocks the Application Insights metrics service for unit tests.
type MockService struct {
	mock.Mock
}

// NewMockClient instantiates a Client backed by a MockService.
func NewMockClient(logger *logp.Logger) *Client {
	return &Client{
		new(MockService),
		Config{},
		logger.Named("test azure appinsights"),
	}
}

// GetMetricValues records the call and returns the configured response.
func (service *MockService) GetMetricValues(applicationId string, bodyMetrics []MetricsBatchRequestItem) (ListMetricsResultsItem, error) {
	args := service.Called(applicationId, bodyMetrics)
	// Tests always stub this with a ListMetricsResultsItem; the comma-ok
	// form is used purely to satisfy errcheck.check-type-assertions.
	res, _ := args.Get(0).(ListMetricsResultsItem)
	return res, args.Error(1)
}
