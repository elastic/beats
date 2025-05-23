// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package billing

import (
	"time"

	"github.com/stretchr/testify/mock"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/consumption/armconsumption"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/costmanagement/armcostmanagement"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/azure"
	"github.com/elastic/elastic-agent-libs/logp"
)

// MockService mock for the azure monitor services
type MockService struct {
	mock.Mock
}

// NewMockClient instantiates a new client with the mock billing service
func NewMockClient() *Client {
	return &Client{
		new(MockService),
		azure.Config{},
		logp.NewLogger("test azure monitor"),
	}
}

// GetForecast is a mock function for the billing service
func (service *MockService) GetForecast(
	scope string,
	startTime,
	endTime time.Time,
) (armcostmanagement.QueryResult, error) {
	args := service.Called(scope, startTime, endTime)
	return args.Get(0).(armcostmanagement.QueryResult), args.Error(1)
}

// GetUsageDetails is a mock function for the billing service
func (service *MockService) GetUsageDetails(
	scope string,
	expand string,
	filter string,
	metricType armconsumption.Metrictype,
	startDate string,
	endDate string,
) (armconsumption.UsageDetailsListResult, error) {
	args := service.Called(scope, expand, filter, metricType, startDate, endDate)
	return args.Get(0).(armconsumption.UsageDetailsListResult), args.Error(1)
}
