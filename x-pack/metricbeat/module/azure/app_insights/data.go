// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package app_insights

import (
	"fmt"
	"strings"

	"github.com/Azure/go-autorest/autorest/date"

	"github.com/Azure/azure-sdk-for-go/services/preview/appinsights/v1/insights"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/metricbeat/mb"
)

var segmentNames = []string{"request_name", "request_urlHost", "operation_name"}

type MetricValue struct {
	SegmentName map[string]string
	Value       map[string]interface{}
	Segments    []MetricValue
	Interval    string
	Start       *date.Time
	End         *date.Time
}

func mapMetricValues(metricValues insights.ListMetricsResultsItem) []MetricValue {
	var mapped []MetricValue
	for _, item := range *metricValues.Value {
		metricValue := MetricValue{
			Start: item.Body.Value.Start,
			End:   item.Body.Value.End,
		}
		metricValue.Interval = fmt.Sprintf("%sTO%s", item.Body.Value.Start, item.Body.Value.End)
		if item.Body != nil && item.Body.Value != nil {
			if item.Body.Value.AdditionalProperties != nil {
				metrics := getAdditionalPropMetric(item.Body.Value.AdditionalProperties)
				for key, metric := range metrics {
					if isSegment(key) {
						metricValue.SegmentName[key] = metric.(string)
					} else {
						metricValue.Value[key] = metric
					}
				}
			}
			if item.Body.Value.Segments != nil {
				for _, segment := range *item.Body.Value.Segments {
					metVal := mapSegment(segment, metricValue.SegmentName)
					metricValue.Segments = append(metricValue.Segments, metVal)
				}
			}
			mapped = append(mapped, metricValue)
		}

	}
	return mapped
}

func mapSegment(segment insights.MetricsSegmentInfo, parentSeg map[string]string) MetricValue {
	metricValue := MetricValue{Value: map[string]interface{}{}, SegmentName: map[string]string{}}
	if segment.AdditionalProperties != nil {
		metrics := getAdditionalPropMetric(segment.AdditionalProperties)
		for key, metric := range metrics {
			if isSegment(key) {
				metricValue.SegmentName[key] = metric.(string)
			} else {
				metricValue.Value[key] = metric
			}
		}
	}
	if len(parentSeg) > 0 {
		for key, val := range parentSeg {
			metricValue.SegmentName[key] = val
		}
	}
	if segment.Segments != nil {
		for _, segment := range *segment.Segments {
			metVal := mapSegment(segment, metricValue.SegmentName)
			metricValue.Segments = append(metricValue.Segments, metVal)
		}
	}

	return metricValue
}

func isSegment(metric string) bool {
	for _, seg := range segmentNames {
		if metric == seg {
			return true
		}
	}
	return false
}

func EventsMapping(metricValues insights.ListMetricsResultsItem, applicationId string) []mb.Event {
	var events []mb.Event
	if metricValues.Value == nil {
		return events
	}
	groupedAddProp := make(map[string][]MetricValue)
	mValues := mapMetricValues(metricValues)

	var segValues []MetricValue
	for _, mv := range mValues {
		if len(mv.Segments) == 0 {
			groupedAddProp[mv.Interval] = append(groupedAddProp[mv.Interval], mv)
		} else {
			segValues = append(segValues, mv)
		}
	}

	for _, val := range groupedAddProp {
		event := createNoSegEvent(val, applicationId)
		if len(event.MetricSetFields) > 0 {
			events = append(events, event)
		}
	}
	for _, val := range segValues {
		for _, seg := range val.Segments {
			lastSeg := getValue(seg)
			for _, ls := range lastSeg {
				events = append(events, createSegEvent(val, ls, applicationId))
			}
		}
	}
	return events
}

func getValue(metric MetricValue) []MetricValue {
	var values []MetricValue
	if metric.Segments == nil {
		return []MetricValue{metric}
	}
	for _, met := range metric.Segments {
		values = append(values, getValue(met)...)
	}
	return values
}

func createSegEvent(parentMetricValue MetricValue, metricValue MetricValue, applicationId string) mb.Event {
	metricList := common.MapStr{}
	for key, metric := range metricValue.Value {
		metricList.Put(key, metric)
	}
	if len(metricList) == 0 {
		return mb.Event{}
	}
	event := mb.Event{
		ModuleFields: common.MapStr{},
		MetricSetFields: common.MapStr{
			"start_date":     parentMetricValue.Start,
			"end_date":       parentMetricValue.End,
			"application_id": applicationId,
		},
		Timestamp: parentMetricValue.End.Time,
	}
	event.RootFields = common.MapStr{}
	event.RootFields.Put("cloud.provider", "azure")
	event.MetricSetFields.Put("metrics", metricList)
	if len(parentMetricValue.SegmentName) > 0 {
		event.ModuleFields.Put("dimensions", parentMetricValue.SegmentName)
	}
	if len(metricValue.SegmentName) > 0 {
		event.ModuleFields.Put("dimensions", metricValue.SegmentName)
	}
	return event
}

func createNoSegEvent(values []MetricValue, applicationId string) mb.Event {
	metricList := common.MapStr{}
	for _, value := range values {
		for key, metric := range value.Value {
			metricList.Put(key, metric)
		}
	}
	if len(metricList) == 0 {
		return mb.Event{}
	}
	event := mb.Event{
		MetricSetFields: common.MapStr{
			"start_date":     values[0].Start,
			"end_date":       values[0].End,
			"application_id": applicationId,
		},
		Timestamp: values[0].End.Time,
	}
	event.RootFields = common.MapStr{}
	event.RootFields.Put("cloud.provider", "azure")
	event.MetricSetFields.Put("metrics", metricList)
	return event
}

func getAdditionalPropMetric(addProp map[string]interface{}) map[string]interface{} {
	metricNames := make(map[string]interface{})
	for key, val := range addProp {
		switch val.(type) {
		case map[string]interface{}:
			for subKey, subVal := range val.(map[string]interface{}) {
				if subVal != nil {
					metricNames[cleanMetricNames(fmt.Sprintf("%s.%s", key, subKey))] = subVal
				}
			}
		default:
			metricNames[cleanMetricNames(key)] = val
		}
	}
	return metricNames
}

func cleanMetricNames(metric string) string {
	return strings.Replace(metric, "/", "_", -1)
}
