// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package billing

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/consumption/armconsumption"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/costmanagement/armcostmanagement"

	"errors"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// EventsMapping maps the usage details and forecast data to a list of metricbeat events to
// send to Elasticsearch.
func EventsMapping(
	subscriptionId string,
	results Usage,
	timeOpts TimeIntervalOptions,
	logger *logp.Logger,
) ([]mb.Event, error) {
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

			switch usageDetails := ud.(type) {
			case *armconsumption.LegacyUsageDetail:
				//
				// legacy data format
				//

				legacy := usageDetails

				event.ModuleFields = mapstr.M{
					"subscription_id":   legacy.Properties.SubscriptionID,
					"subscription_name": legacy.Properties.SubscriptionName,
					"resource": mapstr.M{
						"name":  legacy.Properties.ResourceName,
						"type":  legacy.Properties.ConsumedService,
						"group": legacy.Properties.ResourceGroup,
					},
				}
				if len(legacy.Tags) > 0 {
					_, _ = event.ModuleFields.Put("resource.tags", legacy.Tags)
				}

				event.MetricSetFields = mapstr.M{
					// original fields
					"billing_period_id": legacy.ID,
					"product":           legacy.Properties.Product,
					"pretax_cost":       legacy.Properties.Cost,
					"currency":          legacy.Properties.BillingCurrency,
					"department_name":   legacy.Properties.InvoiceSection,
					"account_name":      legacy.Properties.BillingAccountName,
					"usage_start":       timeOpts.usageStart,
					"usage_end":         timeOpts.usageEnd,

					// additional fields
					"usage_date": legacy.Properties.Date, // Date for the usage record.
					"account_id": legacy.Properties.BillingAccountID,
					"unit_price": legacy.Properties.UnitPrice,
					"quantity":   legacy.Properties.Quantity,
				}
				_, _ = event.RootFields.Put("cloud.region", legacy.Properties.ResourceLocation)
				_, _ = event.RootFields.Put("cloud.instance.name", legacy.Properties.ResourceName)
				_, _ = event.RootFields.Put("cloud.instance.id", legacy.Properties.ResourceID)
			case *armconsumption.ModernUsageDetail:
				//
				// modern data format
				//

				modern := usageDetails

				event.ModuleFields = mapstr.M{
					"subscription_id":   modern.Properties.SubscriptionGUID,
					"subscription_name": modern.Properties.SubscriptionName,
					"resource": mapstr.M{
						"name":  getResourceNameFromPath(*modern.Properties.InstanceName),
						"type":  modern.Properties.ConsumedService,
						"group": strings.ToLower(*modern.Properties.ResourceGroup),
					},
				}
				if len(modern.Tags) > 0 {
					_, _ = event.ModuleFields.Put("resource.tags", modern.Tags)
				}

				event.MetricSetFields = mapstr.M{
					// original fields
					"billing_period_id": modern.ID,
					"product":           modern.Properties.Product,
					"pretax_cost":       modern.Properties.CostInBillingCurrency,
					"currency":          modern.Properties.BillingCurrencyCode,
					"department_name":   modern.Properties.InvoiceSectionName,
					"account_name":      modern.Properties.BillingAccountName,
					"usage_start":       timeOpts.usageStart,
					"usage_end":         timeOpts.usageEnd,

					// additional fields
					"usage_date": modern.Properties.Date, // Date for the usage record.
					"account_id": modern.Properties.BillingAccountID,
					"unit_price": modern.Properties.UnitPrice,
					"quantity":   modern.Properties.Quantity,
				}
				_, _ = event.RootFields.Put("cloud.region", modern.Properties.ResourceLocation)
			default:
				return events, errors.New("unsupported usage details format: not legacy nor modern")
			}

			events = append(events, event)
		}
	}

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
func getEventsFromQueryResult(result armcostmanagement.QueryResult, subscriptionID string, logger *logp.Logger) ([]mb.Event, error) {
	// The number of columns expected in the QueryResult supported by this input.
	// The structure of the QueryResult is determined by the value we set in
	// the `costmanagement.ForecastDefinition` struct at query time.
	const expectedNumberOfColumns = 4

	if result.Properties == nil || result.Properties.Columns == nil {
		return []mb.Event{}, errors.New("unsupported forecasts QueryResult format: no columns")
	}

	if len(result.Properties.Columns) != expectedNumberOfColumns {
		return []mb.Event{}, fmt.Errorf("unsupported forecasts QueryResult format: got %d columns instead of %d", len(result.Properties.Columns), expectedNumberOfColumns)
	}

	if result.Properties.Rows == nil {
		logger.Warn("no rows in forecasts QueryResult")
		return []mb.Event{}, nil
	}

	events := make([]mb.Event, 0, len(result.Properties.Rows))
	for _, row := range result.Properties.Rows {
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
		}

		// test: trying to make the linter happy
		_ = costFieldName

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
