// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package billing

import (
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/costmanagement/mgmt/2019-11-01/costmanagement"

	"github.com/Azure/go-autorest/autorest/date"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"

	"github.com/Azure/azure-sdk-for-go/services/consumption/mgmt/2019-10-01/consumption"
)

func TestEventMapping(t *testing.T) {
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

	events, err := EventsMapping("sub", usage, opts)
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

func column(name, type_ string) costmanagement.QueryColumn {
	return costmanagement.QueryColumn{Name: &name, Type: &type_}
}
