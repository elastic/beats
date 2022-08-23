// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package billing

import (
	"strings"
	"time"

	"errors"

	"github.com/Azure/azure-sdk-for-go/services/consumption/mgmt/2019-10-01/consumption"
	"github.com/shopspring/decimal"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/metricbeat/mb"
)

// EventsMapping maps the usage details to a slice of metricbeat events.
func EventsMapping(subscriptionId string, results Usage, startTime time.Time, endTime time.Time) ([]mb.Event, error) {
	var events []mb.Event
	if len(results.UsageDetails) > 0 {
		// <<<<<<< HEAD
		// 		for _, usageDetail := range results.UsageDetails {
		// 			event := mb.Event{
		// 				ModuleFields: common.MapStr{
		// 					"resource": common.MapStr{
		// 						"type":  usageDetail.ConsumedService,
		// 						"group": getResourceGroupFromId(*usageDetail.InstanceID),
		// 						"name":  usageDetail.InstanceName,
		// 					},
		// 					"subscription_id": usageDetail.SubscriptionGUID,
		// 				},
		// 				MetricSetFields: common.MapStr{
		// 					"pretax_cost":       usageDetail.PretaxCost,
		// 					"department_name":   usageDetail.DepartmentName,
		// 					"product":           usageDetail.Product,
		// 					"usage_start":       usageDetail.UsageStart.ToTime(),
		// 					"usage_end":         usageDetail.UsageEnd.ToTime(),
		// 					"currency":          usageDetail.Currency,
		// 					"billing_period_id": usageDetail.BillingPeriodID,
		// 					"account_name":      usageDetail.AccountName,
		// 				},
		// 				Timestamp: time.Now().UTC(),
		// 			}
		// 			event.RootFields = common.MapStr{}
		// 			event.RootFields.Put("cloud.provider", "azure")
		// 			event.RootFields.Put("cloud.region", usageDetail.InstanceLocation)
		// 			event.RootFields.Put("cloud.instance.name", usageDetail.InstanceName)
		// 			event.RootFields.Put("cloud.instance.id", usageDetail.InstanceID)
		// =======
		for _, ud := range results.UsageDetails {
			event := mb.Event{Timestamp: time.Now().UTC()}

			// shared fields
			event.RootFields = common.MapStr{
				"cloud.provider": "azure",
			}

			if legacy, isLegacy := ud.AsLegacyUsageDetail(); isLegacy {

				//
				// legacy data format
				//

				event.ModuleFields = common.MapStr{
					"subscription_id":   legacy.SubscriptionID,
					"subscription_name": legacy.SubscriptionName,
					"resource": common.MapStr{
						"name":  legacy.ResourceName,
						"type":  legacy.ConsumedService,
						"group": legacy.ResourceGroup,
					},
				}
				event.MetricSetFields = common.MapStr{
					// original fields
					"billing_period_id": legacy.ID,
					"product":           legacy.Product,
					"pretax_cost":       legacy.Cost,
					"currency":          legacy.BillingCurrency,
					"department_name":   legacy.InvoiceSection,
					"account_name":      legacy.BillingAccountName,
					"usage_start":       startTime,
					"usage_end":         endTime,

					// additional fields
					"usage_date": legacy.Date, // Date for the usage record.
					"account_id": legacy.BillingAccountID,
					"unit_price": legacy.UnitPrice,
					"quantity":   legacy.Quantity,
				}
				_, _ = event.RootFields.Put("cloud.region", legacy.ResourceLocation)
				_, _ = event.RootFields.Put("cloud.instance.name", legacy.ResourceName)
				_, _ = event.RootFields.Put("cloud.instance.id", legacy.ResourceID)

			} else if modern, isModern := ud.AsModernUsageDetail(); isModern {

				//
				// modern data format
				//

				event.ModuleFields = common.MapStr{
					"subscription_id":   modern.SubscriptionGUID,
					"subscription_name": modern.SubscriptionName,
					"resource": common.MapStr{
						"name":  getResourceNameFromPath(*modern.InstanceName),
						"type":  modern.ConsumedService,
						"group": modern.ResourceGroup,
					},
				}
				event.MetricSetFields = common.MapStr{
					// original fields
					"billing_period_id": modern.ID,
					"product":           modern.Product,
					"pretax_cost":       modern.CostInBillingCurrency,
					"currency":          modern.BillingCurrencyCode,
					"department_name":   modern.InvoiceSectionName,
					"account_name":      modern.BillingAccountName,
					"usage_start":       startTime,
					"usage_end":         endTime,

					// additional fields
					"usage_date": modern.Date, // Date for the usage record.
					"account_id": modern.BillingAccountID,
					"unit_price": modern.UnitPrice,
					"quantity":   modern.Quantity,
				}
				_, _ = event.RootFields.Put("cloud.region", modern.ResourceLocation)

			} else {

				//
				// Unsupported data format
				//
				return events, errors.New("unsupported usage details format: not legacy nor modern")
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
			RootFields: common.MapStr{
				"cloud.provider": "azure",
			},
			ModuleFields: common.MapStr{
				"subscription_id": subscriptionId,
			},
			MetricSetFields: common.MapStr{
				"actual_cost":   actualCost,
				"forecast_cost": forecastCost,
				"usage_date":    parsedDate,
				"currency":      items[0].Currency,
			},
			Timestamp: time.Now().UTC(),
		}

		events = append(events, event)
	}

	return events, nil
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
