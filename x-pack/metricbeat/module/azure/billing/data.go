// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package billing

import (
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/consumption/mgmt/2019-10-01/consumption"
	"github.com/shopspring/decimal"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func EventsMapping(subscriptionId string, results Usage, startTime time.Time, endTime time.Time) []mb.Event {
	var events []mb.Event
	if len(results.UsageDetails) > 0 {
		for _, ud := range results.UsageDetails {
			event := mb.Event{Timestamp: time.Now().UTC()}

			// shared fields
			event.RootFields = mapstr.M{
				"cloud.provider": "azure",
			}

			//
			// legacy data format
			//
			if legacy, isLegacy := ud.AsLegacyUsageDetail(); isLegacy {
				event.ModuleFields = mapstr.M{
					"subscription_id": legacy.SubscriptionID,
					"resource": mapstr.M{
						"name":  legacy.ResourceName,
						"type":  legacy.ConsumedService,
						"group": legacy.ResourceGroup,
					},
				}
				event.MetricSetFields = mapstr.M{
					// original fields
					"product":         legacy.Product,
					"pretax_cost":     legacy.Cost,
					"currency":        legacy.BillingCurrency,
					"department_name": legacy.InvoiceSection,
					"account_name":    legacy.BillingAccountName,
					"usage_start":     startTime, // not sure the value is correct
					"usage_end":       endTime,   // not sure the value is correct
					// "billing_period_id":   "?", // missing

					"billing_period_start": legacy.BillingPeriodStartDate.ToTime(),
					"billing_period_end":   legacy.BillingPeriodEndDate.ToTime(),

					// additional fields
					"usage_date":        legacy.Date, // Date for the usage record.
					"account_id":        legacy.BillingAccountID,
					"subscription_name": legacy.SubscriptionName,
					"unit_price":        legacy.UnitPrice,
					"quantity":          legacy.Quantity,

					// legacy-only fields
					"effective_price": legacy.EffectivePrice,
				}
				_, _ = event.RootFields.Put("cloud.region", legacy.ResourceLocation)
				_, _ = event.RootFields.Put("cloud.instance.name", legacy.ResourceName)
				_, _ = event.RootFields.Put("cloud.instance.id", legacy.ResourceID)
			}

			//
			// modern data format
			//
			if modern, isModern := ud.AsModernUsageDetail(); isModern {
				event.ModuleFields = mapstr.M{
					"subscription_id": modern.SubscriptionGUID,
					"resource": mapstr.M{
						"name":  getResourceNameFromPath(*modern.InstanceName),
						"type":  modern.ConsumedService,
						"group": modern.ResourceGroup,
					},
				}
				event.MetricSetFields = mapstr.M{
					// original fields
					"product":         modern.Product,
					"pretax_cost":     modern.CostInBillingCurrency,
					"currency":        modern.BillingCurrencyCode,
					"department_name": modern.InvoiceSectionName,
					"account_name":    modern.BillingAccountName,
					"usage_start":     startTime, // not sure the value is correct
					"usage_end":       endTime,   // not sure the value is correct
					// "billing_period_id":   "?", // missing

					// additional fields
					"usage_date":        modern.Date, // Date for the usage record.
					"account_id":        modern.BillingAccountID,
					"subscription_name": modern.SubscriptionName,
					"unit_price":        modern.UnitPrice,
					"quantity":          modern.Quantity,

					"billing_period_start": modern.BillingPeriodStartDate.ToTime(),
					"billing_period_end":   modern.BillingPeriodEndDate.ToTime(),
				}
				_, _ = event.RootFields.Put("cloud.region", modern.ResourceLocation)
			}

			events = append(events, event)
		}
	}

	//
	// Forecasts
	//
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
			RootFields: mapstr.M{
				"cloud.provider": "azure",
			},
			ModuleFields: mapstr.M{
				"subscription_id": subscriptionId,
			},
			MetricSetFields: mapstr.M{
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

// getResourceNameFromPath returns the resource name by picking the last part from a `/` separated resource path.
//
// For example, given a path like the following:
// `/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Compute/virtualMachines/{vmName}`
//
// It would return the value `{vmName}`.
func getResourceNameFromPath(path string) string {
	parts := strings.Split(path, "/")
	// According to the documentation, `string.Split()` always returns a non-empty slice when the separator is not empty,
	// so it should be safe to use `len(parts) - 1` to get the last element.
	return parts[len(parts)-1]
}
