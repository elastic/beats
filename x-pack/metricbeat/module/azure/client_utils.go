// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azure

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/preview/monitor/mgmt/2019-06-01/insights"
)

// DefaultTimeGrain is set as default timegrain for the azure metrics
const DefaultTimeGrain = "PT5M"

// mapMetricValues should map the metric values
func mapMetricValues(metrics []insights.Metric, previousMetrics []MetricValue, startTime time.Time, endTime time.Time) []MetricValue {
	var currentMetrics []MetricValue
	// compare with the previously returned values and filter out any double records
	for _, v := range metrics {
		for _, t := range *v.Timeseries {
			for _, mv := range *t.Data {
				if metricExists(*v.Name.Value, mv, previousMetrics) || metricIsEmpty(mv) {
					continue
				}
				// remove metric values that are not part of the timeline selected
				if mv.TimeStamp.Time.After(startTime) && mv.TimeStamp.Time.Before(endTime) {
					continue
				}
				// define the new metric value and match aggregations values
				var val MetricValue
				val.name = *v.Name.Value
				val.timestamp = mv.TimeStamp.Time
				if mv.Minimum != nil {
					val.min = mv.Minimum
				}
				if mv.Maximum != nil {
					val.max = mv.Maximum
				}
				if mv.Average != nil {
					val.avg = mv.Average
				}
				if mv.Total != nil {
					val.total = mv.Total
				}
				if mv.Count != nil {
					val.count = mv.Count
				}
				if t.Metadatavalues != nil {
					for _, dim := range *t.Metadatavalues {
						val.dimensions = append(val.dimensions, Dimension{Name: *dim.Name.Value, Value: *dim.Value})
					}
				}
				currentMetrics = append(currentMetrics, val)
			}
		}
	}
	return currentMetrics
}

// metricExists will check if the metric value has been retrieved in the past
func metricExists(name string, metric insights.MetricValue, metrics []MetricValue) bool {
	for _, met := range metrics {
		if name == met.name && metric.TimeStamp.Time == met.timestamp && metric.Average == met.avg && metric.Total == met.total && metric.Minimum == met.min &&
			metric.Maximum == met.max && metric.Count == met.count {
			return true
		}
	}
	return false
}

// metricIsEmpty will check if the metric value is empty, this seems to be an issue with the azure sdk
func metricIsEmpty(metric insights.MetricValue) bool {
	if metric.Average == nil && metric.Total == nil && metric.Minimum == nil && metric.Maximum == nil && metric.Count == nil {
		return true
	}
	return false
}

// matchMetrics will compare current metrics
func matchMetrics(prevMet Metric, met Metric) bool {
	if prevMet.Namespace == met.Namespace && reflect.DeepEqual(prevMet.Names, met.Names) && prevMet.Resource.ID == met.Resource.ID &&
		prevMet.Aggregations == met.Aggregations && prevMet.TimeGrain == met.TimeGrain {
		return true
	}
	return false
}

// getResourceGroupFormID maps resource group from resource ID
func getResourceGroupFromID(path string) string {
	params := strings.Split(path, "/")
	for i, param := range params {
		if param == "resourceGroups" {
			return params[i+1]
		}
	}
	return ""
}

// getResourceNameFormID maps resource group from resource ID
func getResourceTypeFromID(path string) string {
	params := strings.Split(path, "/")
	for i, param := range params {
		if param == "providers" {
			return fmt.Sprintf("%s/%s", params[i+1], params[i+2])
		}
	}
	return ""
}

// getResourceNameFormID maps resource group from resource ID
func getResourceNameFromID(path string) string {
	params := strings.Split(path, "/")
	if strings.HasSuffix(path, "/") {
		return params[len(params)-2]
	}
	return params[len(params)-1]

}

// mapTags maps resource tags
func mapTags(azureTags map[string]*string) map[string]string {
	if len(azureTags) == 0 {
		return nil
	}
	var tags = map[string]string{}
	for key, value := range azureTags {
		tags[key] = *value
	}
	return tags
}
