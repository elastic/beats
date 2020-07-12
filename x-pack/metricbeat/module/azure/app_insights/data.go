// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package app_insights

import (
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/preview/appinsights/v1/insights"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/metricbeat/mb"
)

func EventsMapping(metricValues insights.ListMetricsResultsItem, applicationId string) []mb.Event {
	var events []mb.Event
	if metricValues.Value == nil {
		return events
	}
	for _, item := range *metricValues.Value {
		if item.Body != nil && item.Body.Value != nil {
			if item.Body.Value.AdditionalProperties != nil {
				events = append(events, createEvent(*item.Body.Value, insights.MetricsSegmentInfo{}, applicationId))
			} else if item.Body.Value.Segments != nil {
				for _, segment := range *item.Body.Value.Segments {
					events = append(events, createEvent(*item.Body.Value, segment, applicationId))
				}
			}
		}
	}
	return events
}

func createEvent(value insights.MetricsResultInfo, segment insights.MetricsSegmentInfo, applicationId string) mb.Event {
	metricList := common.MapStr{}
	if value.AdditionalProperties != nil {
		metrics := getMetric(value.AdditionalProperties)
		for key, metric := range metrics {
			metricList.Put(key, metric)
		}
	} else {
		metrics := getMetric(segment.AdditionalProperties)
		for key, metric := range metrics {
			metricList.Put(key, metric)
		}
	}
	event := mb.Event{
		MetricSetFields: common.MapStr{
			"start_date":     value.Start,
			"end_date":       value.End,
			"application_id": applicationId,
		},
		Timestamp: value.End.Time,
	}
	event.RootFields = common.MapStr{}
	event.RootFields.Put("cloud.provider", "azure")
	event.MetricSetFields.Put("metrics", metricList)
	return event
}

func getMetric(addProp map[string]interface{}) map[string]interface{} {
	metricNames := make(map[string]interface{})
	for key, val := range addProp {
		switch val.(type) {
		case map[string]interface{}:
			for subKey, subVal := range val.(map[string]interface{}) {
				metricNames[cleanMetricNames(fmt.Sprintf("%s.%s", key, subKey))] = subVal
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
