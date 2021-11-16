// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package billing

import (
	"testing"
	"time"

	prevConsumption "github.com/Azure/azure-sdk-for-go/services/consumption/mgmt/2019-01-01/consumption"
	consumption "github.com/Azure/azure-sdk-for-go/services/consumption/mgmt/2019-10-01/consumption"
	"github.com/Azure/go-autorest/autorest/date"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestEventMapping(t *testing.T) {
	usageDate := "2020-08-08"
	name := "test"
	startDate := date.Time{}

	var charge decimal.Decimal = decimal.NewFromFloat(8.123456)
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
	var prop1 = prevConsumption.UsageDetailProperties{
		InstanceName:     &name,
		SubscriptionName: &name,
		AccountName:      &name,
		DepartmentName:   &name,
		Product:          &name,
		InstanceID:       &name,
		UsageStart:       &startDate,
		UsageEnd:         &startDate,
	}
	usage := Usage{
		UsageDetails: []prevConsumption.UsageDetail{
			{
				UsageDetailProperties: &prop1,
				ID:                    nil,
				Name:                  nil,
				Type:                  nil,
				Tags:                  nil,
			},
		},
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
	events := EventsMapping("sub", usage)
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
