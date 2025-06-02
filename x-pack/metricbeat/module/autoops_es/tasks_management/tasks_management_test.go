// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !integration
// +build !integration

package tasks_management

import (
	"testing"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/auto_ops_testing"
	auto_ops_events "github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/events"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/metricset"
)

var (
	setupClusterHealthErrorServer = auto_ops_testing.SetupDataErrorServer(TasksPath)
	setupSuccessfulServer         = auto_ops_testing.SetupSuccessfulServer(TasksPath)
	useNamedMetricSet             = auto_ops_testing.UseNamedMetricSet(TasksMetricSet)
)

func TestSuccessfulFetch(t *testing.T) {
	metricset.RunTestsForFetcherWithGlobFiles(t, "./_meta/test/tasks.*.json", setupSuccessfulServer, useNamedMetricSet, func(t *testing.T, data metricset.FetcherData[GroupedTasks]) {
		require.NoError(t, data.Error)

		// 1 <= len(...)
		require.LessOrEqual(t, 1, len(data.Reporter.GetEvents()))
	})
}

func TestFailedClusterInfoFetch(t *testing.T) {
	metricset.RunTestsForFetcherWithGlobFiles(t, "./_meta/test/tasks.*.json", auto_ops_testing.SetupClusterInfoErrorServer, useNamedMetricSet, func(t *testing.T, data metricset.FetcherData[GroupedTasks]) {
		require.ErrorContains(t, data.Error, "failed to get cluster info from cluster, tasks_management metricset")
	})
}

func TestFailedTasksFetch(t *testing.T) {
	metricset.RunTestsForFetcherWithGlobFiles(t, "./_meta/test/tasks.*.json", setupClusterHealthErrorServer, useNamedMetricSet, func(t *testing.T, data metricset.FetcherData[GroupedTasks]) {
		require.ErrorContains(t, data.Error, "failed to get data, tasks_management metricset")
	})
}

func TestFailedTasksFetchEventsMapping(t *testing.T) {
	metricset.RunTestsForFetcherWithGlobFiles(t, "./_meta/test/no_*.tasks.*.json", setupSuccessfulServer, useNamedMetricSet, func(t *testing.T, data metricset.FetcherData[GroupedTasks]) {
		require.Error(t, data.Error)

		// Check error event
		event := data.Reporter.GetEvents()[0]
		_, ok := event.MetricSetFields["error"].(auto_ops_events.ErrEvent)
		require.True(t, ok, "expected error event to be of type auto_ops_events.ErrEvent")
	})
}
