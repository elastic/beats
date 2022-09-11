// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package billing

import (
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/consumption/mgmt/2019-10-01/consumption"
	"github.com/Azure/azure-sdk-for-go/services/costmanagement/mgmt/2019-11-01/costmanagement"
	"github.com/Azure/go-autorest/autorest/date"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/metricbeat/mb"
)

// TestEventMapping tests that mapping a QueryResult into a list of events is accurate.
func TestEventMapping(t *testing.T) {
	logger := logp.NewLogger("TestEventMapping")

	ID := "ID"
	kind := "legacy"
	name := "test"
	billingAccountId := "123"
	startDate := date.Time{}

	//
	// Usage Details
	//
	var charge = decimal.NewFromFloat(8.123456)
	var props = consumption.LegacyUsageDetailProperties{
		BillingAccountID:       &billingAccountId,
		BillingAccountName:     &name,
		BillingPeriodStartDate: &startDate,
		BillingPeriodEndDate:   &startDate,
		Cost:                   &charge,
		InvoiceSection:         &name,
		Product:                &name,
	}
	var legacy = consumption.LegacyUsageDetail{
		ID:                          &ID,
		Kind:                        consumption.Kind(kind),
		LegacyUsageDetailProperties: &props,
	}

	//
	// Forecast
	//
	actualCost := float64(0.11)
	forecastCost := float64(0.11)
	// I know, it's weird, but the API returns the usage date as a number using
	// this unusual format.
	actualUsageDate := float64(20200807)
	forecastUsageDate := float64(20200808)
	rows := [][]interface{}{
		{actualCost, actualUsageDate, "Actual", "USD"},
		{forecastCost, forecastUsageDate, "Forecast", "USD"},
	}

	var forecastQueryResult = costmanagement.QueryResult{
		QueryProperties: &costmanagement.QueryProperties{
			Columns: &[]costmanagement.QueryColumn{
				column("Cost", "Number"),
				column("UsageDate", "Number"),
				column("CostStatus", "String"),
				column("Currency", "String"),
			},
			Rows: &rows,
		},
	}

	var usage = Usage{
		UsageDetails: []consumption.BasicUsageDetail{legacy},
		Forecasts:    forecastQueryResult,
	}

	//
	// Run the tests
	//
	usageStart, usageEnd := usageIntervalFrom(time.Now())
	forecastStart, forecastEnd := forecastIntervalFrom(time.Now())
	opts := TimeIntervalOptions{
		usageStart:    usageStart,
		usageEnd:      usageEnd,
		forecastStart: forecastStart,
		forecastEnd:   forecastEnd,
	}

	events, err := EventsMapping("sub", usage, opts, logger)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(events))

	//
	// Check the results
	//
	for _, event := range events {
		// if is an usage event
		if ok, _ := event.MetricSetFields.HasKey("department_name"); ok {
			val1, _ := event.MetricSetFields.GetValue("account_name")
			assert.Equal(t, val1, &name)
			val2, _ := event.MetricSetFields.GetValue("product")
			assert.Equal(t, val2, &name)
			val3, _ := event.MetricSetFields.GetValue("department_name")
			assert.Equal(t, val3, &name)
		} else {

			// Check the actual cost
			isActual, _ := event.MetricSetFields.HasKey("actual_cost")
			if isActual {
				cost, _ := event.MetricSetFields.GetValue("actual_cost")
				assert.Equal(t, actualCost, cost)
				dt, _ := time.Parse("2006-01-02", "2020-08-07")
				usageDate, _ := event.MetricSetFields.GetValue("usage_date")
				assert.Equal(t, usageDate, dt)
			}

			// Check the forecast cost
			isForecast, _ := event.MetricSetFields.HasKey("forecast_cost")
			if isForecast {
				cost, _ := event.MetricSetFields.GetValue("forecast_cost")
				assert.Equal(t, forecastCost, cost)
				dt, _ := time.Parse("2006-01-02", "2020-08-08")
				usageDate, _ := event.MetricSetFields.GetValue("usage_date")
				assert.Equal(t, usageDate, dt)
			}

			if !isActual && !isForecast {
				assert.Fail(t, "Event is neither an actual nor a forecast")
			}
		}
	}
}

func TestGetEventsFromQueryResult(t *testing.T) {
	logger := logp.NewLogger("TestGetEventsFromQueryResult")
	subscriptionID := "sub"

	columns := []costmanagement.QueryColumn{
		column("Cost", "Number"),
		column("UsageDate", "Number"),
		column("CostStatus", "String"),
		column("Currency", "String"),
	}

	t.Run("no columns", func(t *testing.T) {
		queryResult := costmanagement.QueryResult{}

		events, err := getEventsFromQueryResult(queryResult, subscriptionID, logger)
		assert.Equal(t, []mb.Event{}, events)
		assert.Error(t, err)
	})

	t.Run("wrong number of column", func(t *testing.T) {
		badColumns := []costmanagement.QueryColumn{
			column("Cost", "Number"),
			column("UsageDate", "Number"),
			column("CostStatus", "String"),
			column("Currency", "String"),
			column("UnexpectedColumn", "String"),
		}
		queryResult := costmanagement.QueryResult{
			QueryProperties: &costmanagement.QueryProperties{
				Columns: &badColumns,
				Rows:    nil,
			},
		}

		events, err := getEventsFromQueryResult(queryResult, subscriptionID, logger)
		assert.Equal(t, []mb.Event{}, events)
		assert.EqualError(t, err, "unsupported forecasts QueryResult format: got 5 columns instead of 4")
	})

	t.Run("no rows", func(t *testing.T) {
		queryResult := costmanagement.QueryResult{
			QueryProperties: &costmanagement.QueryProperties{
				Columns: &columns,
				Rows:    nil,
			},
		}

		events, err := getEventsFromQueryResult(queryResult, subscriptionID, logger)
		assert.Equal(t, []mb.Event{}, events)
		assert.NoError(t, err)
	})

	t.Run("wrong number of elements in a row", func(t *testing.T) {
		rows := [][]interface{}{
			{float64(1), float64(2), "Actual", "USD", "UnexpectedValue"},
		}
		queryResult := costmanagement.QueryResult{
			QueryProperties: &costmanagement.QueryProperties{
				Columns: &columns,
				Rows:    &rows,
			},
		}

		events, err := getEventsFromQueryResult(queryResult, subscriptionID, logger)
		assert.Equal(t, []mb.Event{}, events)
		assert.NoError(t, err)
	})

	t.Run("drop rows with a wrong type", func(t *testing.T) {
		rows := [][]interface{}{
			{float64(1), float64(20220818), "Actual", "USD"}, // good row, this will be mapped as event
			{42, float64(20220818), "Actual", "USD"},         // wrong cost type
			{float64(1), 20220818, "Actual", "USD"},          // wrong usage date type
			{float64(1), float64(20220818), 42, "USD"},       // wrong cost status type
			{float64(1), float64(20220818), "Actual", 42},    // wrong currency type
		}
		queryResult := costmanagement.QueryResult{
			QueryProperties: &costmanagement.QueryProperties{
				Columns: &columns,
				Rows:    &rows,
			},
		}

		events, err := getEventsFromQueryResult(queryResult, subscriptionID, logger)
		assert.Equal(t, 1, len(events))
		assert.NoError(t, err)
	})
}

func column(name, type_ string) costmanagement.QueryColumn {
	return costmanagement.QueryColumn{Name: &name, Type: &type_}
}
