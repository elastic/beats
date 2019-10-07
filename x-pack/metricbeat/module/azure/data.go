// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azure

import (
	"fmt"
	"strings"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"
)

const (
	noDimension     = "none"
	nativeMetricset = "monitor"
)

// EventsMapping will map metric values to beats events
func EventsMapping(report mb.ReporterV2, metrics []Metric, metricset string) error {
	// metrics and metric values are currently grouped relevant to the azure REST API calls (metrics with the same aggregations per call)
	// multiple metrics can be mapped in one event depending on the resource, namespace, dimensions and timestamp

	// grouping metrics by resource and namespace
	groupByResourceNamespace := make(map[string][]Metric)
	for _, metric := range metrics {
		// check if any values are returned
		if len(metric.Values) == 0 {
			continue
		}
		// build a resource key with unique resource namespace combination
		resNamkey := fmt.Sprintf("%s,%s", metric.Resource.ID, metric.Namespace)
		groupByResourceNamespace[resNamkey] = append(groupByResourceNamespace[resNamkey], metric)
	}
	// grouping metrics by the dimensions configured
	groupByDimensions := make(map[string][]Metric)
	for resNamKey, resourceMetrics := range groupByResourceNamespace {
		for _, resourceMetric := range resourceMetrics {
			if len(resourceMetric.Dimensions) == 0 {
				groupByDimensions[resNamKey+noDimension] = append(groupByDimensions[resNamKey+noDimension], resourceMetric)
			} else {
				var dimKey string
				for _, dim := range resourceMetric.Dimensions {
					dimKey += dim.Name + dim.Value
				}
				groupByDimensions[resNamKey+dimKey] = append(groupByDimensions[resNamKey+dimKey], resourceMetric)
			}

		}
	}

	// grouping metric values by timestamp and creating events (for each metric the REST api can retrieve multiple metric values for same aggregation  but different timeframes)
	for _, grouped := range groupByDimensions {
		defaultMetric := grouped[0]
		groupByTimeMetrics := make(map[time.Time][]MetricValue)
		for _, metric := range grouped {
			for _, m := range metric.Values {
				groupByTimeMetrics[m.timestamp] = append(groupByTimeMetrics[m.timestamp], m)
			}
		}
		for timestamp, groupTimeValues := range groupByTimeMetrics {
			// group events by dimension values
			exists, validDimensions := returnAllDimensions(defaultMetric.Dimensions)
			if exists {
				for _, selectedDimension := range validDimensions {
					groupByDimensions := make(map[string][]MetricValue)
					for _, dimGroupValue := range groupTimeValues {
						dimKey := fmt.Sprintf("%s,%s", selectedDimension.Name, getDimensionValue(selectedDimension.Name, dimGroupValue.dimensions))
						groupByDimensions[dimKey] = append(groupByDimensions[dimKey], dimGroupValue)
					}
					for _, groupDimValues := range groupByDimensions {
						report.Event(initEvent(timestamp, defaultMetric, groupDimValues, metricset))
					}
				}
			} else {
				report.Event(initEvent(timestamp, defaultMetric, groupTimeValues, metricset))
			}
		}

	}

	return nil
}

// managePropertyName function will handle metric names, there are several formats the metric names are written
func managePropertyName(metric string) string {
	// replace spaces with underscores
	resultMetricName := strings.Replace(metric, " ", "_", -1)
	// replace backslashes with "per"
	resultMetricName = strings.Replace(resultMetricName, "/", "_per_", -1)
	resultMetricName = strings.Replace(resultMetricName, "\\", "_", -1)
	// replace actual percentage symbol with the smbol "pct"
	resultMetricName = strings.Replace(resultMetricName, "_%_", "_pct_", -1)
	// create an object in case of ":"
	resultMetricName = strings.Replace(resultMetricName, ":", ".", -1)
	// create an object in case of ":"
	resultMetricName = strings.Replace(resultMetricName, "_-_", "_", -1)
	//  avoid cases as this "logicaldisk_avg._disk_sec_per_transfer"
	obj := strings.Split(resultMetricName, ".")
	for index := range obj {
		// in some cases a trailing "_" is found
		obj[index] = strings.TrimPrefix(obj[index], "_")
		obj[index] = strings.TrimSuffix(obj[index], "_")
	}
	resultMetricName = strings.ToLower(strings.Join(obj, "."))

	return resultMetricName
}

// initEvent will create a new base event
func initEvent(timestamp time.Time, metric Metric, metricValues []MetricValue, metricset string) mb.Event {
	metricList := common.MapStr{}
	for _, value := range metricValues {
		metricNameString := fmt.Sprintf("%s", managePropertyName(value.name))
		if value.min != nil {
			metricList.Put(fmt.Sprintf("%s.%s", metricNameString, "min"), *value.min)
		}
		if value.max != nil {
			metricList.Put(fmt.Sprintf("%s.%s", metricNameString, "max"), *value.max)
		}
		if value.avg != nil {
			metricList.Put(fmt.Sprintf("%s.%s", metricNameString, "avg"), *value.avg)
		}
		if value.total != nil {
			metricList.Put(fmt.Sprintf("%s.%s", metricNameString, "total"), *value.total)
		}
		if value.count != nil {
			metricList.Put(fmt.Sprintf("%s.%s", metricNameString, "count"), *value.count)
		}
	}
	event := mb.Event{
		ModuleFields: common.MapStr{
			"resource": common.MapStr{
				"name":  metric.Resource.Name,
				"type":  metric.Resource.Type,
				"group": metric.Resource.Group,
			},
			"subscription_id": metric.Resource.Subscription,
			"namespace":       metric.Namespace,
		},
		MetricSetFields: common.MapStr{},
		Timestamp:       timestamp,
	}
	if len(metric.Resource.Tags) > 0 {
		event.ModuleFields.Put("resource.tags", metric.Resource.Tags)
	}
	if len(metric.Dimensions) > 0 {
		for _, dimension := range metric.Dimensions {
			if dimension.Value == "*" {
				event.ModuleFields.Put(fmt.Sprintf("dimensions.%s", managePropertyName(dimension.Name)), getDimensionValue(dimension.Name, metricValues[0].dimensions))
			} else {
				event.ModuleFields.Put(fmt.Sprintf("dimensions.%s", managePropertyName(dimension.Name)), dimension.Value)
			}

		}
	}
	if metricset == nativeMetricset {
		event.ModuleFields.Put("metrics", metricList)
	} else {
		for key, metric := range metricList {
			event.MetricSetFields.Put(key, metric)
		}
	}

	event.RootFields = common.MapStr{}
	event.RootFields.Put("cloud.provider", "azure")
	event.RootFields.Put("cloud.region", metric.Resource.Location)
	return event
}

// getDimensionValue will return dimension value for the key provided
func getDimensionValue(dimension string, dimensions []Dimension) string {
	for _, dim := range dimensions {
		if strings.ToLower(dim.Name) == strings.ToLower(dimension) {
			return dim.Value
		}
	}
	return ""
}

// returnAllDimensions will check if users has entered a filter for all dimension values (*)
func returnAllDimensions(dimensions []Dimension) (bool, []Dimension) {
	if len(dimensions) == 0 {
		return false, nil
	}
	var dims []Dimension
	for _, dim := range dimensions {
		if dim.Value == "*" {
			dims = append(dims, dim)
		}
	}
	if len(dims) == 0 {
		return false, nil
	}
	return true, dims
}
