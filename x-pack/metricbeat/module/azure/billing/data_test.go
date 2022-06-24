// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package billing

import (
	"testing"
	"time"

	"github.com/Azure/go-autorest/autorest/date"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"

	"github.com/Azure/azure-sdk-for-go/services/consumption/mgmt/2019-10-01/consumption"
)

func TestEventMapping(t *testing.T) {
	ID := "ID"
	kind := "legacy"
	usageDate := "2020-08-08"
	name := "test"
	billingAccountId := "123"
	startDate := date.Time{}

	var charge = decimal.NewFromFloat(8.123456)
	var prop = consumption.ForecastProperties{
		UsageDate:        &usageDate,
		Grain:            "",
		Charge:           &charge,
		Currency:         &name,
		ChargeType:       "Forecast",
		ConfidenceLevels: nil,
	}
	var prop2 = consumption.ForecastProperties{
		UsageDate:        &usageDate,
		Grain:            "",
		Charge:           &charge,
		Currency:         &name,
		ChargeType:       "Actual",
		ConfidenceLevels: nil,
	}
	var pros = consumption.LegacyUsageDetailProperties{
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
		LegacyUsageDetailProperties: &pros,
	}
	var usage = Usage{UsageDetails: []consumption.BasicUsageDetail{legacy},
		ActualCosts: []consumption.Forecast{
			{
				ForecastProperties: &prop2,
				ID:                 nil,
				Name:               nil,
				Type:               nil,
				Tags:               nil,
			}},
		ForecastCosts: []consumption.Forecast{
			{
				ForecastProperties: &prop,
				ID:                 nil,
				Name:               nil,
				Type:               nil,
				Tags:               nil,
			}},
	}

	startTime := time.Now().UTC().Truncate(24 * time.Hour).Add((-48) * time.Hour)
	endTime := startTime.Add(time.Hour * 24).Add(time.Second * (-1))

	events, err := EventsMapping("sub", usage, startTime, endTime)
	assert.NoError(t, err)
	assert.Equal(t, len(events), 2)

	for _, event := range events {

		if ok, _ := event.MetricSetFields.HasKey("department_name"); ok {
			val1, _ := event.MetricSetFields.GetValue("account_name")
			assert.Equal(t, val1, &name)
			val2, _ := event.MetricSetFields.GetValue("product")
			assert.Equal(t, val2, &name)
			val3, _ := event.MetricSetFields.GetValue("department_name")
			assert.Equal(t, val3, &name)
		} else {
			dt, _ := time.Parse("2006-01-02", usageDate)
			val1, _ := event.MetricSetFields.GetValue("usage_date")
			assert.Equal(t, val1, dt)
			val2, _ := event.MetricSetFields.GetValue("forecast_cost")
			assert.Equal(t, val2, &charge)
			val3, _ := event.MetricSetFields.GetValue("actual_cost")
			assert.Equal(t, val3, &charge)

		}
	}
}
