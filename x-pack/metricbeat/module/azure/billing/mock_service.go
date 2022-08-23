// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package billing

import (
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/azure"
	"github.com/stretchr/testify/mock"

	"github.com/Azure/azure-sdk-for-go/services/consumption/mgmt/2019-10-01/consumption"
	// <<<<<<< HEAD
	// 	"github.com/elastic/beats/v7/libbeat/logp"
	// 	prevConsumption "github.com/Azure/azure-sdk-for-go/services/consumption/mgmt/2019-01-01/consumption"
	// 	"github.com/Azure/azure-sdk-for-go/services/consumption/mgmt/2019-10-01/consumption"
	// =======
	// 	"github.com/elastic/beats/v7/x-pack/metricbeat/module/azure"
	// 	"github.com/elastic/elastic-agent-libs/logp"
	// >>>>>>> 1f232dc343 ([Azure Billing] Upgrade Usage Details API to version 2019-10-01 (#31970))
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
func (service *MockService) GetForecast(filter string) ([]consumption.Forecast, error) {
	args := service.Called(filter)
	return args.Get(0).([]consumption.Forecast), args.Error(1)
}

// GetUsageDetails is a mock function for the billing service
func (service *MockService) GetUsageDetails(scope string, expand string, filter string, skiptoken string, top *int32, metricType consumption.Metrictype, startDate string, endDate string) (consumption.UsageDetailsListResultPage, error) {
	args := service.Called(scope, expand, filter, skiptoken, top, metricType, startDate, endDate)
	return args.Get(0).(consumption.UsageDetailsListResultPage), args.Error(1)
}
