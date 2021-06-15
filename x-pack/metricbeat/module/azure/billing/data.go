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

func EventsMapping(results Usage) []mb.Event {
	var events []mb.Event
	if len(results.UsageDetails) > 0 {
		for _, usageDetail := range results.UsageDetails {
			if usDetail, ok := usageDetail.AsLegacyUsageDetail(); ok {
				event := mb.Event{
					ModuleFields: common.MapStr{
						"resource": common.MapStr{
							"type":  usDetail.ConsumedService,
							"group": getResourceGroupFromId(*usDetail.ID),
							"name":  usDetail.Name,
						},
						"subscription_id": usDetail.SubscriptionID,
					},
					MetricSetFields: common.MapStr{
						//"pretax_cost":       usDetail.Cost,
						//"department_name":   usDetail.,
						"product": usDetail.Product,
						//"usage_start":       usDetail.UsageStart.ToTime(),
						//"usage_end":         usDetail.UsageEnd.ToTime(),
						//"currency":          usDetail.Currency,
						//"billing_period_id": usDetail.BillingPeriodID,
						"account_name": usDetail.AccountName,
					},
					Timestamp: time.Now().UTC(),
				}
				event.RootFields = common.MapStr{}
				event.RootFields.Put("cloud.provider", "azure")
				//event.RootFields.Put("cloud.region", usDetail.InstanceLocation)
				//event.RootFields.Put("cloud.instance.name", usDetail.InstanceName)
				//event.RootFields.Put("cloud.instance.id", usDetail.InstanceID)
				events = append(events, event)
			}

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
