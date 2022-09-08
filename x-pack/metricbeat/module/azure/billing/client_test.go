// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package billing

import (
	"errors"
	"testing"

	prevConsumption "github.com/Azure/azure-sdk-for-go/services/consumption/mgmt/2019-01-01/consumption"
	"github.com/Azure/azure-sdk-for-go/services/consumption/mgmt/2019-10-01/consumption"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

<<<<<<< HEAD
=======
	"github.com/Azure/azure-sdk-for-go/services/consumption/mgmt/2019-10-01/consumption"
	"github.com/Azure/azure-sdk-for-go/services/costmanagement/mgmt/2019-11-01/costmanagement"

>>>>>>> 86b111d594 ([Azure Billing] Switch to Cost Management API for forecast data (#32589))
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/azure"
)

var (
	config = azure.Config{}
)

func TestClient(t *testing.T) {
<<<<<<< HEAD
=======
	usageStart, usageEnd := usageIntervalFrom(time.Now())
	forecastStart, forecastEnd := forecastIntervalFrom(time.Now())
	opts := TimeIntervalOptions{
		usageStart:    usageStart,
		usageEnd:      usageEnd,
		forecastStart: forecastStart,
		forecastEnd:   forecastEnd,
	}

>>>>>>> 86b111d594 ([Azure Billing] Switch to Cost Management API for forecast data (#32589))
	t.Run("return error not valid query", func(t *testing.T) {
		client := NewMockClient()
		client.Config = config
		m := &MockService{}
<<<<<<< HEAD
		m.On("GetForcast", mock.Anything).Return(consumption.ForecastsListResult{}, errors.New("invalid query"))
		m.On("GetUsageDetails", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(prevConsumption.UsageDetailsListResultPage{}, nil)
		client.BillingService = m
		results, err := client.GetMetrics()
=======
		m.On("GetForecast", mock.Anything, mock.Anything, mock.Anything).Return(costmanagement.QueryResult{}, errors.New("invalid query"))
		m.On("GetUsageDetails", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(consumption.UsageDetailsListResultPage{}, nil)
		client.BillingService = m
		_, err := client.GetMetrics(opts)
>>>>>>> 86b111d594 ([Azure Billing] Switch to Cost Management API for forecast data (#32589))
		assert.Error(t, err)
		//assert.NotNil(t, usage.Forecasts)
		//assert.True(t, usage.Forecasts.Rows == nil)
		//assert.Equal(t, len(*usage.Forecasts.Rows), 0)
		m.AssertExpectations(t)
	})
	t.Run("return results", func(t *testing.T) {
		client := NewMockClient()
		client.Config = config
		m := &MockService{}
<<<<<<< HEAD
		forecasts := []consumption.Forecast{{}, {}}
		m.On("GetForcast", mock.Anything).Return(consumption.ForecastsListResult{Value: &forecasts}, nil)
		m.On("GetUsageDetails", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(prevConsumption.UsageDetailsListResultPage{}, nil)
		client.BillingService = m
		results, err := client.GetMetrics()
=======
		forecasts := costmanagement.QueryResult{}
		m.On("GetForecast", mock.Anything, mock.Anything, mock.Anything).Return(forecasts, nil)
		m.On("GetUsageDetails", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(consumption.UsageDetailsListResultPage{}, nil)
		client.BillingService = m
		_, err := client.GetMetrics(opts)
>>>>>>> 86b111d594 ([Azure Billing] Switch to Cost Management API for forecast data (#32589))
		assert.NoError(t, err)
		//assert.NotNil(t, usage.Forecasts.Rows)
		//assert.Equal(t, len(*usage.Forecasts.Rows), 2)
		// assert.Equal(t, len(results.ForecastCosts), 2)
		m.AssertExpectations(t)
	})
}
