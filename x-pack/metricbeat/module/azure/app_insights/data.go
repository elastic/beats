// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package app_insights

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/preview/appinsights/v1/insights"
	"github.com/Azure/go-autorest/autorest/date"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/azure"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/elastic/beats/v7/metricbeat/mb"
)

const aggsRegex = "_(?:sum|count|unique|avg|min|max)$"

// segmentNames list is used to filter out the dimension from the api response. Based on the body format it is not possible to detect what was the segment selected
var segmentNames = []string{
	"request_source", "request_name", "request_url_host", "request_url_path", "request_success", "request_result_code", "request_performance_bucket", "operation_name", "operation_synthetic", "operation_synthetic_source", "user_authenticated", "application_version", "client_type", "client_model",
	"client_os", "client_city", "client_state_or_province", "client_country_or_region", "client_browser", "cloud_role_name", "cloud_role_instance", "custom_dimensions__ms_processedb_by_metric_extractors", "custom_dimensions_developer_mode",
	"page_view_name", "page_view_url_path", "page_view_url_host", "page_view_performance_bucket", "custom_dimensions_ibiza_session_id", "custom_dimensions_part_instance", "browser_timing_name", "browser_timing_url_host", "browser_timing_url_path", "browser_timing_performance_bucket",
	"trace_severity_level", "type", "custom_dimensions_agent_session", "custom_dimensions_agent_version", "custom_dimensions_machine_name", "custom_dimensions_running_mode", "custom_dimensions_source", "custom_dimensions_agent_assembly_version", "custom_dimensions_agent_process_session",
	"custom_dimensions_hashed_machine_name",
	"custom_dimensions_data_cube", "dependency_target", "dependency_type", "dependency_name", "dependency_success", "dependency_result_code", "dependency_performance_bucket", "custom_dimensions_container", "custom_dimensions_blob", "custom_dimensions_error_message",
	"custom_event_name", "custom_dimensions_event_name", "custom_dimensions_page_title", "custom_dimensions_service_profiler_content", "custom_dimensions_executing_assembly_file_version", "custom_dimensions_service_profiler_version", "custom_dimensions_process_id", "custom_dimensions_request_id",
	"custom_dimensions_running_session", "custom_dimensions_problem_id", "custom_dimensions_snapshot_context", "custom_dimensions_snapshot_version", "custom_dimensions_duration", "custom_dimensions_snapshot_id", "custom_dimensions_stamp_id", "custom_dimensions_de_optimization_id",
	"custom_dimensions_method", "custom_dimensions_parent_process_id", "custom_dimensions_section", "custom_dimensions_configuration", "custom_dimensions_dump_folder", "custom_dimensions_reason", "custom_dimensions_extension_version", "custom_dimensions_site_name",
	"availability_result_name", "availability_result_location", "availability_result_success", "custom_dimensions_full_test_result_available", "exception_problem_id", "exception_handled_at", "exception_type", "exception_assembly", "exception_method", "custom_dimensions_custom_perf_counter",
	"exception_severity_level", "custom_dimensions_url", "custom_dimensions_ai.snapshot_stampid", "custom_dimensions_ai.snapshot_id", "custom_dimensions_ai.snapshot_version", "custom_dimensions_ai.snapshot_planid", "custom_dimensions__ms_example", "custom_dimensions_sa_origin_app_id",
	"custom_dimensions_base_sdk_target_framework", "custom_dimensions_runtime_framework", "custom_dimensions__ms_aggregation_interval_ms", "custom_dimensions_problem_id", "custom_dimensions_operation_name", "custom_dimensions_request_success", "custom_dimensions__ms_metric_id",
	"custom_dimensions_dependency_success", "custom_dimensions__ms_is_autocollected", "custom_dimensions_dependency_type", "performance_counter_name", "performance_counter_category", "performance_counter_counter", "performance_counter_instance", "custom_dimensions_counter_instance_name",
}

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
			Start:       item.Body.Value.Start,
			End:         item.Body.Value.End,
			Value:       map[string]interface{}{},
			SegmentName: map[string]string{},
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

type metricTimeKey struct {
	Start time.Time
	End   time.Time
}

func newMetricTimeKey(start, end time.Time) metricTimeKey {
	return metricTimeKey{Start: start, End: end}
}

func EventsMapping(metricValues insights.ListMetricsResultsItem, applicationId string, namespace string) []mb.Event {
	var events []mb.Event
	if metricValues.Value == nil {
		return events
	}

	mValues := mapMetricValues(metricValues)

	groupedByDimensions := groupMetricsByDimension(mValues)

	for _, group := range groupedByDimensions {
		event := createGroupEvent(group, newMetricTimeKey(group[0].Start.Time, group[0].End.Time), applicationId, namespace)

		// Only add events that have metric values.
		if len(event.MetricSetFields) > 0 {
			events = append(events, event)
		}
	}
	return events
}

// groupMetricsByDimension groups the given metrics by their dimension keys.
func groupMetricsByDimension(metrics []MetricValue) map[string][]MetricValue {
	keys := make(map[string][]MetricValue)

	var stack []MetricValue
	stack = append(stack, metrics...)

	// Initialize default start and end times using the first metric's times
	// The reason we need to use first metric's start and end times is because
	// the start and end times of the child segments are not always set.
	firstStart := metrics[0].Start
	firstEnd := metrics[0].End

	// Iterate until all metrics are processed
	for len(stack) > 0 {
		// Retrieve and remove the last metric from the stack
		metric := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		// Update default times if the current metric has valid start and end times
		if metric.End != nil && !metric.End.IsZero() {
			firstEnd = metric.End
		}
		if metric.Start != nil && !metric.Start.IsZero() {
			firstStart = metric.Start
		}

		// Generate a sorted key from the segment names to ensure consistent dimension keys
		sortedSegmentsKey := getSortedKeys(metric.SegmentName)

		// Construct a dimension key using the default times and sorted segment names
		dimensionKey := createDimensionKey(firstStart.Unix(), firstEnd.Unix(), sortedSegmentsKey)

		// If the metric has child segments, process them
		// This is usually the case for segments that don't have actual metric values
		if len(metric.Segments) > 0 {
			for _, segment := range metric.Segments {
				// Generate a sorted key from the segment names
				segmentKey := getSortedKeys(segment.SegmentName)
				if segmentKey != "" {
					// Combine the dimension key with the segment key
					combinedKey := dimensionKey + segmentKey

					// Create a new metric with the combined key and add it to the map
					newMetric := MetricValue{
						SegmentName: segment.SegmentName,
						Value:       segment.Value,
						Segments:    segment.Segments,
						Interval:    segment.Interval,
						Start:       firstStart,
						End:         firstEnd,
					}

					keys[combinedKey] = append(keys[combinedKey], newMetric)
				}
				// Add the child segments to the stack for processing
				stack = append(stack, segment.Segments...)
			}
		} else {
			// If the metric has no child segments, add it to the map using the dimension key
			// This is usually the case for segments that have actual metric values
			if dimensionKey != "" {
				metric.Start, metric.End = firstStart, firstEnd
				keys[dimensionKey] = append(keys[dimensionKey], metric)
			}
		}
	}

	return keys
}

// getSortedKeys is a function that returns a string of sorted keys.
// The keys are sorted in alphabetical order.
//
// By sorting the keys, we ensure that we always get the same string for the same map,
// regardless of the order in which the keys were originally added.
//
// For example, consider the following two maps:
// map1: map[string]string{"request_url_host": "", "request_url_path": "/home"}
// map2: map[string]string{"request_url_path": "/home", "request_url_host": ""}
// Even though they represent the same data, if we were to join their keys without sorting,
// we would get different results: "request_url_hostrequest_url_path" for map1 and
// "request_url_pathrequest_url_host" for map2.
//
// By sorting the keys, we ensure that we always get "request_url_hostrequest_url_path",
// regardless of the order in which the keys were added to the map.
func getSortedKeys(m map[string]string) string {
	keys := make([]string, 0, len(m))
	for k, v := range m {
		keys = append(keys, k+v)
	}
	sort.Strings(keys)

	return strings.Join(keys, "")
}

// createDimensionKey is used to generate a unique key for a specific dimension.
// The dimension key is a combination of the start time, end time, and sorted segments.
//
// startTime: The start time of the metric in Unix timestamp format.
// endTime: The end time of the metric in Unix timestamp format.
// sortedSegments: A string representing sorted segments (metric names).
//
// For example: 1617225600_1617232800_request_url_hostlocalhost
func createDimensionKey(startTime, endTime int64, sortedSegments string) string {
	return fmt.Sprintf("%d_%d_%s", startTime, endTime, sortedSegments)
}

func createGroupEvent(metricValue []MetricValue, metricTime metricTimeKey, applicationId, namespace string) mb.Event {
	// If the metric time is zero then we don't have a valid event.
	// This should never happen, it's a safety check.
	if metricTime.Start.IsZero() || metricTime.End.IsZero() {
		return mb.Event{}
	}

	metricList := mapstr.M{}

	for _, v := range metricValue {
		for key, metric := range v.Value {
			_, _ = metricList.Put(key, metric)
		}
	}

	// If we don't have any metrics then we don't have a valid event.
	if len(metricList) == 0 {
		return mb.Event{}
	}

	event := mb.Event{
		ModuleFields: mapstr.M{"application_id": applicationId},
		MetricSetFields: mapstr.M{
			"start_date": metricTime.Start,
			"end_date":   metricTime.End,
		},
		Timestamp: metricTime.End,
	}

	event.RootFields = mapstr.M{}
	_, _ = event.RootFields.Put("cloud.provider", "azure")

	segments := make(map[string]string)

	for _, v := range metricValue {
		for sn, sv := range v.SegmentName {
			segments[sn] = sv
		}
	}

	if len(segments) > 0 {
		_, _ = event.ModuleFields.Put("dimensions", segments)
	}

	if namespace == "" {
		_, _ = event.ModuleFields.Put("metrics", metricList)
	} else {
		for key, metric := range metricList {
			_, _ = event.MetricSetFields.Put(key, metric)
		}
	}

	return event
}

func getAdditionalPropMetric(addProp map[string]interface{}) map[string]interface{} {
	metricNames := make(map[string]interface{})
	for key, val := range addProp {
		switch v := val.(type) {
		case map[string]interface{}:
			for subKey, subVal := range v {
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
	metric = strings.Replace(metric, "/", "_", -1)
	metric = strings.Replace(metric, " ", "_", -1)
	metric = azure.ReplaceUpperCase(metric)
	obj := strings.Split(metric, ".")
	for index := range obj {
		// in some cases a trailing "_" is found
		obj[index] = strings.TrimPrefix(obj[index], "_")
		obj[index] = strings.TrimSuffix(obj[index], "_")
	}
	metric = strings.ToLower(strings.Join(obj, "_"))
	aggsRegex := regexp.MustCompile(aggsRegex)
	metric = aggsRegex.ReplaceAllStringFunc(metric, func(str string) string {
		return strings.Replace(str, "_", ".", -1)
	})
	return metric
}
