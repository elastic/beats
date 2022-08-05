// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package billing

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/costmanagement/mgmt/2019-11-01/costmanagement"

	"errors"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// EventsMapping maps the usage details to a slice of metricbeat events.
func EventsMapping(subscriptionId string, results Usage, opts TimeIntervalOptions) ([]mb.Event, error) {
	events := make([]mb.Event, 0, len(results.UsageDetails))

	//
	// Usage Details
	//

	if len(results.UsageDetails) > 0 {
		for _, ud := range results.UsageDetails {
			event := mb.Event{Timestamp: time.Now().UTC()}

			// shared fields
			event.RootFields = mapstr.M{
				"cloud.provider": "azure",
			}

			if legacy, isLegacy := ud.AsLegacyUsageDetail(); isLegacy {

				//
				// legacy data format
				//

				event.ModuleFields = mapstr.M{
					"subscription_id":   legacy.SubscriptionID,
					"subscription_name": legacy.SubscriptionName,
					"resource": mapstr.M{
						"name":  legacy.ResourceName,
						"type":  legacy.ConsumedService,
						"group": legacy.ResourceGroup,
					},
				}
				event.MetricSetFields = mapstr.M{
					// original fields
					"billing_period_id": legacy.ID,
					"product":           legacy.Product,
					"pretax_cost":       legacy.Cost,
					"currency":          legacy.BillingCurrency,
					"department_name":   legacy.InvoiceSection,
					"account_name":      legacy.BillingAccountName,
					"usage_start":       opts.usageStart,
					"usage_end":         opts.usageEnd,

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
					"usage_start":       opts.usageStart,
					"usage_end":         opts.usageEnd,

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

	forecastsEvents, err := getEventsFromQueryResult(results.Forecasts, subscriptionId)
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
func getEventsFromQueryResult(result costmanagement.QueryResult, subscriptionID string) ([]mb.Event, error) {
	// the number of columns expected in the QueryResult supported by this input.
	expectedNumberOfColumns := 4

	if result.Columns == nil || len(*result.Columns) != expectedNumberOfColumns {
		return []mb.Event{}, fmt.Errorf("unsupported forecasts QueryResult format: %d instead of %d", len(*result.Columns), expectedNumberOfColumns)
	}

	if result.Rows == nil {
		return []mb.Event{}, errors.New("unsupported forecasts QueryResult format: no rows")
	}

	events := make([]mb.Event, 0, len(*result.Rows))
	for _, row := range *result.Rows {
		var cost float64
		var currency string
		var costStatus string
		var usageDate time.Time

		if len(row) != expectedNumberOfColumns {
			return events, fmt.Errorf("unsupported forecasts QueryResult.Rows format: %d instead of %d", len(row), expectedNumberOfColumns)
		}

		// cost
		if value, ok := row[0].(float64); !ok {
			return events, errors.New("unsupported cost format: not float64")
		} else {
			cost = value
		}

		// usage date
		if value, ok := row[1].(float64); !ok {
			return events, errors.New("unsupported usage date format: not float64")
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
				return events, errors.New("unsupported usage date format: not valid date")
			}
		}

		// cost status (can be "Actual" or "Forecast")
		if value, ok := row[2].(string); !ok {
			return events, errors.New("unsupported cost status format: not string")
		} else {
			costStatus = value
		}

		// currency (can be "USD", "EUR", or other currency symbols)
		if value, ok := row[3].(string); !ok {
			return events, errors.New("unsupported currency format: not string")
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
			return events, errors.New("unsupported cost status: not 'Actual' or 'Forecast'")
		}

		event := mb.Event{
			RootFields: mapstr.M{
				"cloud.provider": "azure",
			},
			ModuleFields: mapstr.M{
				"subscription_id": subscriptionID,
			},
			MetricSetFields: mapstr.M{
				costFieldName: cost,
				"usage_date":  usageDate,
				"currency":    currency,
			},
			Timestamp: time.Now().UTC(),
		}

		events = append(events, event)
	}

	return events, nil
}
