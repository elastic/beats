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

func EventsMapping(metricValues insights.ListMetricsResultsItem, applicationId string) []mb.Event {
	var events []mb.Event
	if metricValues.Value == nil {
		return events
	}
	groupedAddProp := make(map[string][]insights.MetricsResultInfo)
	for _, item := range *metricValues.Value {
		if item.Body != nil && item.Body.Value != nil {
			if item.Body.Value.AdditionalProperties != nil {
				groupedAddProp[fmt.Sprintf("%sTO%s", item.Body.Value.Start, item.Body.Value.End)] =
					append(groupedAddProp[fmt.Sprintf("%sTO%s", item.Body.Value.Start, item.Body.Value.End)], *item.Body.Value)
			} else if item.Body.Value.Segments != nil {
				for _, segment := range *item.Body.Value.Segments {
					event, ok := createSegmentEvent(*item.Body.Value.Start, *item.Body.Value.End, segment, applicationId)
					if ok {
						events = append(events, event)
					}
				}
			}
		}
	}
	if len(groupedAddProp) > 0 {
		for _, val := range groupedAddProp {
			event, ok := createEvent(val, applicationId)
			if ok {
				events = append(events, event)
			}
		}
	}
	return events
}

func createSegmentEvent(start date.Time, end date.Time, segment insights.MetricsSegmentInfo, applicationId string) (mb.Event, bool) {
	metricList := common.MapStr{}
	metrics := getMetric(segment.AdditionalProperties)
	if len(metrics) == 0 {
		return mb.Event{}, false
	}
	for key, metric := range metrics {
		metricList.Put(key, metric)
	}
	event := mb.Event{
		MetricSetFields: common.MapStr{
			"start_date":     start,
			"end_date":       end,
			"application_id": applicationId,
		},
		Timestamp: end.Time,
	}
	event.RootFields = common.MapStr{}
	event.RootFields.Put("cloud.provider", "azure")
	event.MetricSetFields.Put("metrics", metricList)
	return event, true
}

func createEvent(values []insights.MetricsResultInfo, applicationId string) (mb.Event, bool) {
	metricList := common.MapStr{}
	for _, value := range values {
		metrics := getMetric(value.AdditionalProperties)
		for key, metric := range metrics {
			metricList.Put(key, metric)
		}
	}
	if len(metricList) == 0 {
		return mb.Event{}, false
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
	return event, true
}

func getMetric(addProp map[string]interface{}) map[string]interface{} {
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
