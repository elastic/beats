// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !requirefips

package billing

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/consumption/armconsumption"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/costmanagement/armcostmanagement"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/azure"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

var (
	config = azure.Config{}
)

func TestClient(t *testing.T) {
	usageStart, usageEnd := usageIntervalFrom(time.Now(), defaultUsageLookback)
	forecastStart, forecastEnd := forecastIntervalFrom(time.Now(), defaultForecastWindow)
	opts := TimeIntervalOptions{
		usageStart:    usageStart,
		usageEnd:      usageEnd,
		forecastStart: forecastStart,
		forecastEnd:   forecastEnd,
	}

	t.Run("return error not valid query", func(t *testing.T) {
		client := NewMockClient(logptest.NewTestingLogger(t, ""))
		client.Config = config
		m := &MockService{}
		m.On("GetForecast", mock.Anything, mock.Anything, mock.Anything).Return(armcostmanagement.QueryResult{}, errors.New("invalid query"))
		m.On("GetUsageDetails", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(armconsumption.UsageDetailsListResult{}, nil)
		client.BillingService = m
		_, err := client.GetMetrics(opts)
		assert.Error(t, err)
		//assert.NotNil(t, usage.Forecasts)
		//assert.True(t, usage.Forecasts.Rows == nil)
		//assert.Equal(t, len(*usage.Forecasts.Rows), 0)
		m.AssertExpectations(t)
	})
	t.Run("return results", func(t *testing.T) {
		client := NewMockClient(logptest.NewTestingLogger(t, ""))
		client.Config = config
		m := &MockService{}
		forecasts := armcostmanagement.QueryResult{}
		m.On("GetForecast", mock.Anything, mock.Anything, mock.Anything).Return(forecasts, nil)
		m.On("GetUsageDetails", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(armconsumption.UsageDetailsListResult{}, nil)
		client.BillingService = m
		_, err := client.GetMetrics(opts)
		assert.NoError(t, err)
		//assert.NotNil(t, usage.Forecasts.Rows)
		//assert.Equal(t, len(*usage.Forecasts.Rows), 2)
		// assert.Equal(t, len(results.ForecastCosts), 2)
		m.AssertExpectations(t)
	})
}

func TestClientUsesRangeFilterForUsageDetails(t *testing.T) {
	opts := TimeIntervalOptions{
		usageStart:    time.Date(2026, 7, 19, 0, 0, 0, 0, time.UTC),
		usageEnd:      time.Date(2026, 7, 21, 23, 59, 59, 0, time.UTC),
		forecastStart: time.Date(2026, 7, 19, 0, 0, 0, 0, time.UTC),
		forecastEnd:   time.Date(2026, 8, 17, 23, 59, 59, 0, time.UTC),
	}

	client := NewMockClient(logptest.NewTestingLogger(t, ""))
	client.Config = azure.Config{SubscriptionId: "sub"}
	m := &MockService{}

	expectedFilter := "properties/usageStart ge '2026-07-19T00:00:00Z' and properties/usageEnd le '2026-07-21T23:59:59Z'"
	m.On(
		"GetUsageDetails",
		"subscriptions/sub",
		"properties/meterDetails",
		expectedFilter,
		armconsumption.MetrictypeActualCostMetricType,
		"2026-07-19",
		"2026-07-21",
	).Return(armconsumption.UsageDetailsListResult{}, nil)
	m.On("GetForecast", "subscriptions/sub", opts.forecastStart, opts.forecastEnd).Return(armcostmanagement.QueryResult{}, nil)
	client.BillingService = m

	_, err := client.GetMetrics(opts)
	assert.NoError(t, err)
	m.AssertExpectations(t)
}
