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

// EventsMapping will map metric values to beats events
func EventsMapping(report mb.ReporterV2, metrics []Metric) error {
	for _, metric := range metrics {
		// check if any valid values are returned
		if len(metric.Values) == 0 {
			continue
		}
		// group events by timestamp (api will return multiple timestamps)
		groupByTimeMetrics := make(map[time.Time][]MetricValue)
		for _, m := range metric.Values {
			groupByTimeMetrics[m.timestamp] = append(groupByTimeMetrics[m.timestamp], m)
		}
		for timestamp, groupTimeValues := range groupByTimeMetrics {
			// group events by dimension values
			exists, validDimensions := returnAllDimensions(metric.Dimensions)
			if exists {
				for _, selectedDimension := range validDimensions {
					groupByDimensions := make(map[string][]MetricValue)
					for _, dimGroupValue := range groupTimeValues {
						dimKey := fmt.Sprintf("%s,%s", selectedDimension.Name, getDimensionValue(selectedDimension.Name, dimGroupValue.dimensions))
						groupByDimensions[dimKey] = append(groupByDimensions[dimKey], dimGroupValue)
					}
					for _, groupDimValues := range groupByDimensions {
						report.Event(initEvent(timestamp, metric, groupDimValues))
					}
				}
			} else {
				report.Event(initEvent(timestamp, metric, groupTimeValues))
			}
		}
	}

	return nil
}

// managePropertyName function will handle metric names
func managePropertyName(metric string) string {
	resultMetricName := strings.Replace(metric, " ", "_", -1)
	resultMetricName = strings.Replace(resultMetricName, "/", "_per_", -1)
	resultMetricName = strings.ToLower(resultMetricName)
	return resultMetricName
}

// initEvent will create a new base event
func initEvent(timestamp time.Time, metric Metric, metricValues []MetricValue) mb.Event {
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
		Timestamp: timestamp,
		MetricSetFields: common.MapStr{
			"resource": common.MapStr{
				"name":  metric.Resource.Name,
				"type":  metric.Resource.Type,
				"group": metric.Resource.Group,
			},
			"namespace":       metric.Namespace,
			"subscription_id": metric.Resource.Subscription,
			"metrics":         metricList,
		},
	}
	if len(metric.Resource.Tags) > 0 {
		event.MetricSetFields.Put("resource.tags", metric.Resource.Tags)
	}
	if len(metric.Dimensions) > 0 {
		for _, dimension := range metric.Dimensions {
			if dimension.Value == "*" {
				event.MetricSetFields.Put(fmt.Sprintf("dimensions.%s", managePropertyName(dimension.Name)), getDimensionValue(dimension.Name, metricValues[0].dimensions))
			} else {
				event.MetricSetFields.Put(fmt.Sprintf("dimensions.%s", managePropertyName(dimension.Name)), dimension.Value)
			}

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
