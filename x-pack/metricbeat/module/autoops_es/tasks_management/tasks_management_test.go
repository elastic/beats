// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !integration

package tasks_management

import (
	"strings"
	"testing"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/auto_ops_testing"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/metricset"

	"github.com/stretchr/testify/require"
)

var (
	setupSuccessfulServer = auto_ops_testing.SetupSuccessfulServer(TasksPath)
	useNamedMetricSet     = auto_ops_testing.UseNamedMetricSet(TasksMetricSet)
)

func TestSuccessfulFetch(t *testing.T) {
	metricset.RunTestsForFetcherWithGlobFiles(t, "./_meta/test/tasks.*.json", setupSuccessfulServer, useNamedMetricSet, func(t *testing.T, data metricset.FetcherData[GroupedTasks]) {
		require.NoError(t, data.Error)

		// 1 <= len(...)
		require.LessOrEqual(t, 1, len(data.Reporter.GetEvents()))
	})
}

// The filter must collect snapshot creation, but neither the nonexistent indices:admin/snapshot* nor anything broader that would also match get, status, delete and restore
func TestTasksPathSnapshotFilter(t *testing.T) {
	require.Contains(t, TasksPath, "cluster:admin/snapshot/create*", "filter should collect snapshot creation tasks")
	require.NotContains(t, TasksPath, "indices:admin/snapshot", "filter should not use the nonexistent indices:admin/snapshot action")
	require.False(t, strings.Contains(TasksPath, "cluster:admin/snapshot*"), "filter should not collect every snapshot action")
}
