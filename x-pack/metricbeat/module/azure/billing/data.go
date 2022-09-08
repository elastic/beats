// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package billing

import (
	"fmt"
<<<<<<< HEAD
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/consumption/mgmt/2019-10-01/consumption"

	"github.com/shopspring/decimal"
=======
	"strconv"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/costmanagement/mgmt/2019-11-01/costmanagement"

	"errors"
>>>>>>> 86b111d594 ([Azure Billing] Switch to Cost Management API for forecast data (#32589))

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/metricbeat/mb"
<<<<<<< HEAD
)

func EventsMapping(subscriptionId string, results Usage) []mb.Event {
	var events []mb.Event
=======
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// EventsMapping maps the usage details and forecast data to a list of metricbeat events to
// send to Elasticsearch.
func EventsMapping(subscriptionId string, results Usage, timeOpts TimeIntervalOptions, logger *logp.Logger) ([]mb.Event, error) {
	events := make([]mb.Event, 0, len(results.UsageDetails))

	//
	// Usage Details
	//

>>>>>>> 86b111d594 ([Azure Billing] Switch to Cost Management API for forecast data (#32589))
	if len(results.UsageDetails) > 0 {
		for _, usageDetail := range results.UsageDetails {
			event := mb.Event{
				ModuleFields: common.MapStr{
					"resource": common.MapStr{
						"type":  usageDetail.ConsumedService,
						"group": getResourceGroupFromId(*usageDetail.InstanceID),
						"name":  usageDetail.InstanceName,
					},
<<<<<<< HEAD
					"subscription_id": usageDetail.SubscriptionGUID,
				},
				MetricSetFields: common.MapStr{
					"pretax_cost":       usageDetail.PretaxCost,
					"department_name":   usageDetail.DepartmentName,
					"product":           usageDetail.Product,
					"usage_start":       usageDetail.UsageStart.ToTime(),
					"usage_end":         usageDetail.UsageEnd.ToTime(),
					"currency":          usageDetail.Currency,
					"billing_period_id": usageDetail.BillingPeriodID,
					"account_name":      usageDetail.AccountName,
				},
				Timestamp: time.Now().UTC(),
=======
				}
				event.MetricSetFields = mapstr.M{
					// original fields
					"billing_period_id": legacy.ID,
					"product":           legacy.Product,
					"pretax_cost":       legacy.Cost,
					"currency":          legacy.BillingCurrency,
					"department_name":   legacy.InvoiceSection,
					"account_name":      legacy.BillingAccountName,
					"usage_start":       timeOpts.usageStart,
					"usage_end":         timeOpts.usageEnd,

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

				event.ModuleFields = mapstr.M{
					"subscription_id":   modern.SubscriptionGUID,
					"subscription_name": modern.SubscriptionName,
					"resource": mapstr.M{
						"name":  getResourceNameFromPath(*modern.InstanceName),
						"type":  modern.ConsumedService,
						"group": strings.ToLower(*modern.ResourceGroup),
					},
				}
				event.MetricSetFields = mapstr.M{
					// original fields
					"billing_period_id": modern.ID,
					"product":           modern.Product,
					"pretax_cost":       modern.CostInBillingCurrency,
					"currency":          modern.BillingCurrencyCode,
					"department_name":   modern.InvoiceSectionName,
					"account_name":      modern.BillingAccountName,
					"usage_start":       timeOpts.usageStart,
					"usage_end":         timeOpts.usageEnd,

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
>>>>>>> 86b111d594 ([Azure Billing] Switch to Cost Management API for forecast data (#32589))
			}
			event.RootFields = common.MapStr{}
			event.RootFields.Put("cloud.provider", "azure")
			event.RootFields.Put("cloud.region", usageDetail.InstanceLocation)
			event.RootFields.Put("cloud.instance.name", usageDetail.InstanceName)
			event.RootFields.Put("cloud.instance.id", usageDetail.InstanceID)
			events = append(events, event)
		}
	}

<<<<<<< HEAD
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
=======
	//
	// Forecasts
	//

	forecastsEvents, err := getEventsFromQueryResult(results.Forecasts, subscriptionId, logger)
	if err != nil {
		return events, err
	}

	events = append(events, forecastsEvents...)

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

// getEventsFromQueryResult returns the events from the QueryResult obtained Cost Management API.
//
// Here's what you'll find in the QueryResult:
//
// .Columns:
// 0: Cost: Number
// 1: UsageDate: Number
// 2: CostStatus: String
// 3: Currency: String
//
// .Rows:
// 0: []interface {}{0.11, 2.0200807e+07, "Actual", "USD"}
// 1: []interface {}{0.11, 2.0200808e+07, "Forecast", "USD"}
//
func getEventsFromQueryResult(result costmanagement.QueryResult, subscriptionID string, logger *logp.Logger) ([]mb.Event, error) {
	// The number of columns expected in the QueryResult supported by this input.
	// The structure of the QueryResult is determined by the value we set in
	// the `costmanagement.ForecastDefinition` struct at query time.
	const expectedNumberOfColumns = 4

	if result.QueryProperties == nil || result.Columns == nil {
		return []mb.Event{}, errors.New("unsupported forecasts QueryResult format: no columns")
	}

	if len(*result.Columns) != expectedNumberOfColumns {
		return []mb.Event{}, fmt.Errorf("unsupported forecasts QueryResult format: got %d columns instead of %d", len(*result.Columns), expectedNumberOfColumns)
	}

	if result.Rows == nil {
		logger.Warn("no rows in forecasts QueryResult")
		return []mb.Event{}, nil
	}

	events := make([]mb.Event, 0, len(*result.Rows))
	for _, row := range *result.Rows {
		var cost float64
		var currency string
		var costStatus string
		var usageDate time.Time

		if len(row) != expectedNumberOfColumns {
			logger.Errorf("unsupported forecasts QueryResult.Rows format: %d instead of %d", len(row), expectedNumberOfColumns)
			continue
		}

		// Cost
		if value, ok := row[0].(float64); !ok {
			logger.Errorf("unsupported cost format: not float64")
			continue
		} else {
			cost = value
		}

		// Usage date
		if value, ok := row[1].(float64); !ok {
			logger.Errorf("unsupported usage date format: not float64")
			continue
		} else {
			var err error
			// The API returns the usage date as a float64 number representing the "YYYYMMDD" value. For example,
			// the value `float64(20170401)` represents the date "2017-04-01".
			//
			// If you print the row using the following statement:
			//
			// fmt.Printf("Row: %#v\n", row)
			//
			// You will see the following output:
			//
			// Row: []interface {}{0.11, 2.0200807e+07, "Actual", "USD"}
			//
			// 20170401 (float64) --> "2017-04-01T00:00:00Z" (time.Time)
			usageDate, err = time.Parse("20060102", strconv.FormatInt(int64(value), 10))
			if err != nil {
				logger.Errorf("unsupported usage date format: not valid date: %w", err)
				continue
			}
		}

		// Cost status (can be "Actual" or "Forecast")
		if value, ok := row[2].(string); !ok {
			logger.Errorf("unsupported cost status format: not string")
			continue
		} else {
			costStatus = value
		}

		// Currency code (can be "USD", "EUR", or other currency codes)
		if value, ok := row[3].(string); !ok {
			logger.Errorf("unsupported currency code format: not string")
			continue
		} else {
			currency = value
		}

		var costFieldName string
		switch costStatus {
		case "Actual":
			costFieldName = "actual_cost"
		case "Forecast":
			costFieldName = "forecast_cost"
		default:
			logger.Errorf("unsupported cost status: not 'Actual' or 'Forecast'")
			continue
>>>>>>> 86b111d594 ([Azure Billing] Switch to Cost Management API for forecast data (#32589))
		}
		event := mb.Event{
			RootFields: common.MapStr{
				"cloud.provider": "azure",
			},
<<<<<<< HEAD
			ModuleFields: common.MapStr{
				"subscription_id": subscriptionId,
			},
			MetricSetFields: common.MapStr{
				"actual_cost":   actualCost,
				"forecast_cost": forecastCost,
				"usage_date":    parsedDate,
				"currency":      items[0].Currency,
=======
			ModuleFields: mapstr.M{
				"subscription_id": subscriptionID,
			},
			MetricSetFields: mapstr.M{
				costFieldName: cost,
				"usage_date":  usageDate,
				"currency":    currency,
>>>>>>> 86b111d594 ([Azure Billing] Switch to Cost Management API for forecast data (#32589))
			},
			Timestamp: time.Now().UTC(),
		}
		//event.ID = generateEventID(parsedDate)
		events = append(events, event)
	}
	return events
}
<<<<<<< HEAD

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
=======
>>>>>>> 86b111d594 ([Azure Billing] Switch to Cost Management API for forecast data (#32589))
