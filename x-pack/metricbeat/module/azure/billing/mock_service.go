// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package billing

import (
	"github.com/stretchr/testify/mock"

	"github.com/elastic/beats/v8/x-pack/metricbeat/module/azure"

	"github.com/elastic/beats/v8/libbeat/logp"

	prevConsumption "github.com/Azure/azure-sdk-for-go/services/consumption/mgmt/2019-01-01/consumption"
	"github.com/Azure/azure-sdk-for-go/services/consumption/mgmt/2019-10-01/consumption"
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

// GetForcast is a mock function for the billing service
func (service *MockService) GetForcast(filter string) (consumption.ForecastsListResult, error) {
	args := service.Called(filter)
	return args.Get(0).(consumption.ForecastsListResult), args.Error(1)
}

// GetUsageDetails is a mock function for the billing service
func (service *MockService) GetUsageDetails(scope string, expand string, filter string, skiptoken string, top *int32, apply string) (prevConsumption.UsageDetailsListResultPage, error) {
	args := service.Called(scope, expand, filter, skiptoken, top, apply)
	return args.Get(0).(prevConsumption.UsageDetailsListResultPage), args.Error(1)
}
