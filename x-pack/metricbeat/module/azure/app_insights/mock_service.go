// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package app_insights

import (
	"github.com/Azure/azure-sdk-for-go/services/preview/appinsights/v1/insights"
	"github.com/stretchr/testify/mock"

	"github.com/menderesk/beats/v7/libbeat/logp"
)

// Service interface for the azure monitor service and mock for testing
type Service interface {
	GetMetricValues(applicationId string, bodyMetrics []insights.MetricsPostBodySchema) (insights.ListMetricsResultsItem, error)
}

// MockService mock for the azure monitor services
type MockService struct {
	mock.Mock
}

// NewMockClient instantiates a new client with the mock billing service
func NewMockClient() *Client {
	return &Client{
		new(MockService),
		Config{},
		logp.NewLogger("test azure appinsights"),
	}
}

// GetMetricValues will return specified app insights metrics
func (service *MockService) GetMetricValues(applicationId string, bodyMetrics []insights.MetricsPostBodySchema) (insights.ListMetricsResultsItem, error) {
	args := service.Called(applicationId, bodyMetrics)
	return args.Get(0).(insights.ListMetricsResultsItem), args.Error(1)
}
