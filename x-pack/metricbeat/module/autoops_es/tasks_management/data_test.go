// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !integration
// +build !integration

package tasks_management

import (
	"testing"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/auto_ops_testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/metricset"
)

// Tests that Cluster Info is consistently reported and the Task is properly reported
func expectValidParsedData(t *testing.T, data metricset.FetcherData[GroupedTasks]) {
	require.NoError(t, data.Error)
	require.Equal(t, 0, len(data.Reporter.GetErrors()))
	require.LessOrEqual(t, 1, len(data.Reporter.GetEvents()))

	events := data.Reporter.GetEvents()

	auto_ops_testing.CheckAllEventsUseSameTransactionId(t, events)

	event := events[0]

	auto_ops_testing.CheckEventWithRandomTransactionId(t, event, data.ClusterInfo)

	// metrics exist
	require.True(t, len(*event.MetricSetFields.FlattenKeys()) > 2)

	// mapper is expected to drop this field if it appears
	require.Nil(t, auto_ops_testing.GetObjectValue(event.MetricSetFields, "children"))
}

// Tests that Cluster Info is consistently reported and the Tasks are properly reported
func expectValidParsedMultiTasks(t *testing.T, data metricset.FetcherData[GroupedTasks]) {
	expectValidParsedData(t, data)

	events := data.Reporter.GetEvents()

	require.LessOrEqual(t, 2, len(events))

	event1 := auto_ops_testing.GetEventByName(t, events, "task.taskId", "node1:45")
	event2 := auto_ops_testing.GetEventByName(t, events, "task.taskId", "node2:501")

	auto_ops_testing.CheckEventWithRandomTransactionId(t, event2, data.ClusterInfo)

	// metrics exist

	// event 1 (search)
	require.Equal(t, "node1:45", auto_ops_testing.GetObjectValue(event1.MetricSetFields, "task.taskId"))
	require.ElementsMatch(t, []string{"node1", "node2"}, auto_ops_testing.GetObjectValue(event1.MetricSetFields, "task.node"))
	require.EqualValues(t, 45, auto_ops_testing.GetObjectValue(event1.MetricSetFields, "task.id"))
	require.Equal(t, "transport", auto_ops_testing.GetObjectValue(event1.MetricSetFields, "task.taskType"))
	require.Equal(t, "indices:data/read/search", auto_ops_testing.GetObjectValue(event1.MetricSetFields, "task.action"))
	require.Equal(t, "async_search{indices[my-fake-search], search_type[QUERY_THEN_FETCH], source[{\"size\":1,\"query\":{\"match_all\":{}}}], preference[123]}", auto_ops_testing.GetObjectValue(event1.MetricSetFields, "task.description"))
	require.EqualValues(t, 1513823752749, auto_ops_testing.GetObjectValue(event1.MetricSetFields, "task.startTimeInMillis"))
	require.EqualValues(t, 60000293139, auto_ops_testing.GetObjectValue(event1.MetricSetFields, "task.runningTimeInNanos"))
	require.Equal(t, true, auto_ops_testing.GetObjectValue(event1.MetricSetFields, "task.cancellable"))
	require.Equal(t, "123456", auto_ops_testing.GetObjectValue(event1.MetricSetFields, "task.headers.X-Opaque-Id"))

	// event 2 (index)
	require.Equal(t, "node2:501", auto_ops_testing.GetObjectValue(event2.MetricSetFields, "task.taskId"))
	require.ElementsMatch(t, []string{"node2", "node3"}, auto_ops_testing.GetObjectValue(event2.MetricSetFields, "task.node"))
	require.EqualValues(t, 501, auto_ops_testing.GetObjectValue(event2.MetricSetFields, "task.id"))
	require.Equal(t, "transport", auto_ops_testing.GetObjectValue(event2.MetricSetFields, "task.taskType"))
	require.Equal(t, "indices:data/write/bulk", auto_ops_testing.GetObjectValue(event2.MetricSetFields, "task.action"))
	require.Equal(t, "requests[1], indices[my-fake-index]", auto_ops_testing.GetObjectValue(event2.MetricSetFields, "task.description"))
	require.EqualValues(t, 1513823752456, auto_ops_testing.GetObjectValue(event2.MetricSetFields, "task.startTimeInMillis"))
	require.EqualValues(t, 60000293456, auto_ops_testing.GetObjectValue(event2.MetricSetFields, "task.runningTimeInNanos"))
	require.Equal(t, false, auto_ops_testing.GetObjectValue(event2.MetricSetFields, "task.cancellable"))
	require.Equal(t, "456", auto_ops_testing.GetObjectValue(event2.MetricSetFields, "task.headers.X-Opaque-Id"))

	// mapper is expected to drop this field if it appears
	require.Nil(t, auto_ops_testing.GetObjectValue(event1.MetricSetFields, "children"))
	require.Nil(t, auto_ops_testing.GetObjectValue(event2.MetricSetFields, "children"))

	// schema is expected to drop this field
	require.Nil(t, auto_ops_testing.GetObjectValue(event1.MetricSetFields, "ignored_field"))
	require.Nil(t, auto_ops_testing.GetObjectValue(event2.MetricSetFields, "ignored_field"))
}

// Tests that Cluster Info is consistently reported and the Task is properly reported
func expectMixedValidParsedData(t *testing.T, data metricset.FetcherData[GroupedTasks]) {
	require.ErrorContains(t, data.Error, "failed applying task schema")

	require.Equal(t, 0, len(data.Reporter.GetErrors()))
	require.LessOrEqual(t, 1, len(data.Reporter.GetEvents()))

	events := data.Reporter.GetEvents()

	auto_ops_testing.CheckAllEventsUseSameTransactionId(t, events)

	event := auto_ops_testing.GetEventByName(t, events, "task.taskId", "node1:45")

	auto_ops_testing.CheckEventWithRandomTransactionId(t, event, data.ClusterInfo)

	// metrics exist
	require.True(t, len(*event.MetricSetFields.FlattenKeys()) > 2)
	require.Equal(t, "node1:45", auto_ops_testing.GetObjectValue(event.MetricSetFields, "task.taskId"))

	// mapper is expected to drop this field if it appears
	require.Nil(t, auto_ops_testing.GetObjectValue(event.MetricSetFields, "task.children"))
}

// Tests that the schema rejects the data
func expectError(t *testing.T, data metricset.FetcherData[GroupedTasks]) {
	require.ErrorContains(t, data.Error, "failed applying task schema")
}

// Expect a valid response from Elasticsearch to create N events
func TestProperlyHandlesResponse(t *testing.T) {
	metricset.RunTestsForServerlessMetricSetWithGlobFiles(t, "./_meta/test/tasks.*.json", TasksMetricSet, eventsMapping, expectValidParsedData)
}

// Expect a valid response from Elasticsearch to create zero events if nothing is old enough.
func TestProperlyIgnoresValuesFromResponse(t *testing.T) {
	t.Setenv(TASK_RUNTIME_THRESHOLD_IN_SECONDS_NAME, "120") // automatically unsets/resets after test

	metricset.RunTestsForServerlessMetricSetWithGlobFiles(t, "./_meta/test/tasks.*.json", TasksMetricSet, eventsMapping, func(t *testing.T, data metricset.FetcherData[GroupedTasks]) {
		require.NoError(t, data.Error)
		require.Equal(t, 0, len(data.Reporter.GetErrors()))
		require.Equal(t, 0, len(data.Reporter.GetEvents()))
	})
}

// Expect a valid response from Elasticsearch to create N events
func TestProperlyHandlesMultiResponse(t *testing.T) {
	metricset.RunTestsForServerlessMetricSetWithGlobFiles(t, "./_meta/test/tasks.multi.*.json", TasksMetricSet, eventsMapping, expectValidParsedMultiTasks)
}

// Expect a valid response from Elasticsearch to create N events
func TestProperlyHandlesInnerErrorsInResponse(t *testing.T) {
	metricset.RunTestsForServerlessMetricSetWithGlobFiles(t, "./_meta/test/mixed.tasks.*.json", TasksMetricSet, eventsMapping, expectMixedValidParsedData)
}

// Expect a corrupt response from Elasticsearch to trigger an error while applying the schema
func TestProperlyFailsOnBadResponse(t *testing.T) {
	metricset.RunTestsForServerlessMetricSetWithGlobFiles(t, "./_meta/test/no_*.tasks.*.json", TasksMetricSet, eventsMapping, expectError)
}
