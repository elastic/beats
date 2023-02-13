// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package app_insights

import (
	"fmt"
	"regexp"
	"strings"

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

func EventsMapping(metricValues insights.ListMetricsResultsItem, applicationId string, namespace string) []mb.Event {
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
		event := createNoSegEvent(val, applicationId, namespace)
		if len(event.MetricSetFields) > 0 {
			events = append(events, event)
		}
	}
	for _, val := range segValues {
		for _, seg := range val.Segments {
			lastSeg := getValue(seg)
			for _, ls := range lastSeg {
				events = append(events, createSegEvent(val, ls, applicationId, namespace))
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

func createSegEvent(parentMetricValue MetricValue, metricValue MetricValue, applicationId string, namespace string) mb.Event {
	metricList := mapstr.M{}
	for key, metric := range metricValue.Value {
		metricList.Put(key, metric)
	}
	if len(metricList) == 0 {
		return mb.Event{}
	}
	event := createEvent(parentMetricValue.Start, parentMetricValue.End, applicationId, namespace, metricList)
	if len(parentMetricValue.SegmentName) > 0 {
		event.ModuleFields.Put("dimensions", parentMetricValue.SegmentName)
	}
	if len(metricValue.SegmentName) > 0 {
		event.ModuleFields.Put("dimensions", metricValue.SegmentName)
	}
	return event
}

func createEvent(start *date.Time, end *date.Time, applicationId string, namespace string, metricList mapstr.M) mb.Event {
	event := mb.Event{
		ModuleFields: mapstr.M{
			"application_id": applicationId,
		},
		MetricSetFields: mapstr.M{
			"start_date": start,
			"end_date":   end,
		},
		Timestamp: end.Time,
	}
	event.RootFields = mapstr.M{}
	event.RootFields.Put("cloud.provider", "azure")
	if namespace == "" {
		event.ModuleFields.Put("metrics", metricList)
	} else {
		for key, metric := range metricList {
			event.MetricSetFields.Put(key, metric)
		}
	}
	return event
}

func createNoSegEvent(values []MetricValue, applicationId string, namespace string) mb.Event {
	metricList := mapstr.M{}
	for _, value := range values {
		for key, metric := range value.Value {
			metricList.Put(key, metric)
		}
	}
	if len(metricList) == 0 {
		return mb.Event{}
	}
	return createEvent(values[0].Start, values[0].End, applicationId, namespace, metricList)

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
