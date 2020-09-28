// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azure

import (
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/preview/monitor/mgmt/2019-06-01/insights"
)

// DefaultTimeGrain is set as default timegrain for the azure metrics
const DefaultTimeGrain = "PT5M"

var instanceIdRegex = regexp.MustCompile(`.*?(\d+)$`)

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
		if name == met.name &&
			metric.TimeStamp.Equal(met.timestamp) &&
			compareMetricValues(met.avg, metric.Average) &&
			compareMetricValues(met.total, metric.Total) &&
			compareMetricValues(met.max, metric.Maximum) &&
			compareMetricValues(met.min, metric.Minimum) &&
			compareMetricValues(met.count, metric.Count) {
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
	if prevMet.Namespace == met.Namespace && reflect.DeepEqual(prevMet.Names, met.Names) && prevMet.ResourceId == met.ResourceId &&
		prevMet.Aggregations == met.Aggregations && prevMet.TimeGrain == met.TimeGrain {
		return true
	}
	return false
}

// getResourceGroupFromId maps resource group from resource ID
func getResourceGroupFromId(path string) string {
	params := strings.Split(path, "/")
	for i, param := range params {
		if param == "resourceGroups" {
			return params[i+1]
		}
	}
	return ""
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

// compareMetricValues will compare 2 pointer values
func compareMetricValues(metVal *float64, metricVal *float64) bool {
	if metVal == nil && metricVal == nil {
		return true
	}
	if metVal == nil || metricVal == nil {
		return false
	}
	if *metVal == *metricVal {
		return true
	}
	return false
}

// convertTimegrainToDuration will convert azure timegrain options to actual duration values
func convertTimegrainToDuration(timegrain string) time.Duration {
	var duration time.Duration
	switch timegrain {
	case "PT1M":
		duration = time.Duration(time.Minute)
	default:
	case "PT5M":
		duration = time.Duration(5 * time.Minute)
	case "PT15M":
		duration = time.Duration(15 * time.Minute)
	case "PT30M":
		duration = time.Duration(30 * time.Minute)
	case "PT1H":
		duration = time.Duration(time.Hour)
	case "PT6H":
		duration = time.Duration(6 * time.Hour)
	case "PT12H":
		duration = time.Duration(12 * time.Hour)
	case "PT1D":
		duration = time.Duration(24 * time.Hour)
	}
	return duration
}

// groupMetricsByResource is used in order to group metrics by resource and return data faster
func groupMetricsByResource(metrics []Metric) map[string][]Metric {
	grouped := make(map[string][]Metric)
	for _, metric := range metrics {
		if _, ok := grouped[metric.ResourceId]; !ok {
			grouped[metric.ResourceId] = make([]Metric, 0)
		}
		grouped[metric.ResourceId] = append(grouped[metric.ResourceId], metric)
	}
	return grouped
}

// getDimension will check if the dimension value is found in the list
func getDimension(dimension string, dimensions []Dimension) (Dimension, bool) {
	for _, dim := range dimensions {
		if strings.ToLower(dim.Name) == strings.ToLower(dimension) {
			return dim, true
		}
	}
	return Dimension{}, false
}

func containsResource(resourceId string, resources []Resource) bool {
	for _, res := range resources {
		if res.Id == resourceId {
			return true
		}
	}
	return false
}

func getInstanceId(dimensionValue string) string {
	matches := instanceIdRegex.FindStringSubmatch(dimensionValue)
	if len(matches) == 2 {
		return matches[1]
	}
	return ""
}

func getVM(vmName string, vms []VmResource) (VmResource, bool) {
	if len(vms) == 0 {
		return VmResource{}, false
	}
	for _, vm := range vms {
		if vm.Name == vmName {
			return vm, true
		}
	}
	return VmResource{}, false
}
