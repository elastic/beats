// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package billing

import (
	"fmt"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/consumption/mgmt/2019-10-01/consumption"

	"github.com/shopspring/decimal"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/metricbeat/mb"
)

func EventsMapping(results Usage, startTime time.Time, endTime time.Time) []mb.Event {
	var events []mb.Event
	if len(results.UsageDetails) > 0 {
		for _, ud := range results.UsageDetails {
			event := mb.Event{Timestamp: time.Now().UTC()}
			if legacyUsageDetail, err := ud.AsLegacyUsageDetail(); err == true {
				event.ModuleFields = common.MapStr{
					"resource": common.MapStr{
						"type":  legacyUsageDetail.ConsumedService,
						"group": legacyUsageDetail.ResourceGroup,
						"name":  legacyUsageDetail.ResourceName,
					},
					"subscription_id": legacyUsageDetail.SubscriptionID,
				}
				event.MetricSetFields = common.MapStr{
					"pretax_cost": legacyUsageDetail.Cost,
					//"department_name":   legacyUsageDetail.,
					"product":              legacyUsageDetail.Product,
					"usage_start":          startTime,
					"usage_end":            endTime,
					"billing_period_start": legacyUsageDetail.BillingPeriodStartDate.ToTime(),
					"billing_period_end":   legacyUsageDetail.BillingPeriodEndDate.ToTime(),
					"currency":             legacyUsageDetail.BillingCurrency,
					//"billing_period_id": legacyUsageDetail.per,
					"effective_price":      legacyUsageDetail.EffectivePrice,
					"account_name":         legacyUsageDetail.AccountName,
					"account_id":           legacyUsageDetail.BillingAccountID,
					"billing_account_name": legacyUsageDetail.BillingAccountName,
					"subscription_name":    legacyUsageDetail.SubscriptionName,
					"unit_price":           legacyUsageDetail.UnitPrice,
				}
				event.RootFields = common.MapStr{}
				event.RootFields.Put("cloud.provider", "azure")
				event.RootFields.Put("cloud.region", legacyUsageDetail.ResourceLocation)
				event.RootFields.Put("cloud.instance.name", legacyUsageDetail.ResourceName)
				event.RootFields.Put("cloud.instance.id", legacyUsageDetail.ResourceID)
			}
			if modernUsageDetail, err := ud.AsModernUsageDetail(); err == true {
				event.ModuleFields = common.MapStr{
					"resource": common.MapStr{
						"type":  modernUsageDetail.ConsumedService,
						"group": modernUsageDetail.ResourceGroup,
						"name":  modernUsageDetail.InstanceName,
					},
					"subscription_id": modernUsageDetail.SubscriptionGUID,
				}
				event.MetricSetFields = common.MapStr{
					"product":              modernUsageDetail.Product,
					"usage_start":          startTime,
					"usage_end":            endTime,
					"billing_period_start": modernUsageDetail.BillingPeriodStartDate.ToTime(),
					"billing_period_end":   modernUsageDetail.BillingPeriodEndDate.ToTime(),
					"currency":             modernUsageDetail.BillingCurrencyCode,
					"account_id":           modernUsageDetail.BillingAccountID,
					"billing_account_name": modernUsageDetail.BillingAccountName,
					"subscription_name":    modernUsageDetail.SubscriptionName,
					"unit_price":           modernUsageDetail.UnitPrice,
				}
				event.RootFields = common.MapStr{}
				event.RootFields.Put("cloud.provider", "azure")
				event.RootFields.Put("cloud.region", modernUsageDetail.ResourceLocation)
			}
			if _, err := ud.AsUsageDetail(); err == true {
				continue
			}
			events = append(events, event)
		}
	}

	groupedCosts := make(map[*string][]consumption.Forecast)
	for _, forecast := range results.ForecastCosts {
		groupedCosts[forecast.UsageDate] = append(groupedCosts[forecast.UsageDate], forecast)
	}
	for _, forecast := range results.ActualCosts {
		groupedCosts[forecast.UsageDate] = append(groupedCosts[forecast.UsageDate], forecast)
	}
	for usageDate, items := range groupedCosts {
		var actualCost *decimal.Decimal
		var forecastCost *decimal.Decimal
		for _, item := range items {
			if item.ChargeType == consumption.ChargeTypeActual {
				actualCost = item.Charge
			} else {
				forecastCost = item.Charge
			}
		}
		parsedDate, err := time.Parse("2006-01-02", *usageDate)
		if err != nil {
			parsedDate = time.Now().UTC()
		}
		event := mb.Event{
			RootFields: common.MapStr{
				"cloud.provider": "azure",
			},
			MetricSetFields: common.MapStr{
				"actual_cost":   actualCost,
				"forecast_cost": forecastCost,
				"usage_date":    parsedDate,
				"currency":      items[0].Currency,
			},
			Timestamp: time.Now().UTC(),
		}
		//event.ID = generateEventID(parsedDate)
		events = append(events, event)
	}
	return events
}

// getResourceGroupFromId maps resource group from resource ID
func getResourceGroupFromId(path string) string {
	params := strings.Split(path, "/")
	for i, param := range params {
		if param == "resourceGroups" {
			return fmt.Sprintf("%s", params[i+1])
		}
	}
	return ""
}
