// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package billing

import (
	"errors"
	"testing"
	"time"

	// consumption "github.com/Azure/azure-sdk-for-go/services/consumption/mgmt/2019-01-01/consumption"
	// "github.com/Azure/azure-sdk-for-go/services/consumption/mgmt/2019-10-01/consumption"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/azure"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/azure/billing/consumption/mgmt/2019-10-01/consumption"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	config = azure.Config{}
)

func TestClient(t *testing.T) {
	startTime := time.Now().UTC().Truncate(24 * time.Hour).Add((-48) * time.Hour)
	endTime := startTime.Add(time.Hour * 24).Add(time.Second * (-1))

	t.Run("return error not valid query", func(t *testing.T) {
		client := NewMockClient()
		client.Config = config
		m := &MockService{}
		m.On("GetForecast", mock.Anything).Return(consumption.ForecastsListResult{}, errors.New("invalid query"))
		m.On("GetUsageDetails", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(consumption.UsageDetailsListResultPage{}, nil)
		client.BillingService = m
		results, err := client.GetMetrics(startTime, endTime)
		assert.Error(t, err)
		assert.Equal(t, len(results.ActualCosts), 0)
		m.AssertExpectations(t)
	})
	t.Run("return results", func(t *testing.T) {
		client := NewMockClient()
		client.Config = config
		m := &MockService{}
		forecasts := []consumption.Forecast{{}, {}}
		m.On("GetForecast", mock.Anything).Return(consumption.ForecastsListResult{Value: &forecasts}, nil)
		m.On("GetUsageDetails", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(consumption.UsageDetailsListResultPage{}, nil)
		client.BillingService = m
		results, err := client.GetMetrics(startTime, endTime)
		assert.NoError(t, err)
		assert.Equal(t, len(results.ActualCosts), 2)
		assert.Equal(t, len(results.ForecastCosts), 2)
		m.AssertExpectations(t)
	})
}
