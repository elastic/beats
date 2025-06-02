// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !integration
// +build !integration

package cat_template

import (
	"testing"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/auto_ops_testing"
	autoopsevents "github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/events"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/metricset"
)

var (
	setupCatTemplateErrorServer = auto_ops_testing.SetupDataErrorServer(CatTemplatePath)
	setupEmptySuccessfulServer  = auto_ops_testing.SetupSuccessfulServer(CatTemplatePath)
	useNamedMetricSet           = auto_ops_testing.UseNamedMetricSet(CatTemplateMetricSet)
)

func TestEmptySuccessfulFetch(t *testing.T) {
	metricset.RunTestsForFetcherWithGlobFiles(t, "./_meta/test/empty.*.json", setupEmptySuccessfulServer, useNamedMetricSet, func(t *testing.T, data metricset.FetcherData[[]CatTemplate]) {
		require.NoError(t, data.Error)
		require.Equal(t, 0, len(data.Reporter.GetEvents()))
	})
}

func TestSuccessfulFetch(t *testing.T) {
	metricset.RunTestsForFetcherWithGlobFiles(t, "./_meta/test/cat_template.*.json", auto_ops_testing.SetupSuccessfulTemplateServer(CatTemplatePath, templatePathPrefix, getTemplateResponse), useNamedMetricSet, func(t *testing.T, data metricset.FetcherData[[]CatTemplate]) {
		require.NoError(t, data.Error)

		// 1 <= len(...)
		require.LessOrEqual(t, 1, len(data.Reporter.GetEvents()))
	})
}

func TestFailedClusterInfoFetch(t *testing.T) {
	metricset.RunTestsForFetcherWithGlobFiles(t, "./_meta/test/cat_template.*.json", auto_ops_testing.SetupClusterInfoErrorServer, useNamedMetricSet, func(t *testing.T, data metricset.FetcherData[[]CatTemplate]) {
		require.ErrorContains(t, data.Error, "failed to get cluster info from cluster, cat_template metricset")
	})
}

func TestFailedTasksFetch(t *testing.T) {
	metricset.RunTestsForFetcherWithGlobFiles(t, "./_meta/test/cat_template.*.json", setupCatTemplateErrorServer, useNamedMetricSet, func(t *testing.T, data metricset.FetcherData[[]CatTemplate]) {
		require.ErrorContains(t, data.Error, "failed to get data, cat_template metricset")
	})
}

func TestFailedTasksFetchEventsMapping(t *testing.T) {
	// Note: it will fail due to an inner error looking up templates
	metricset.RunTestsForFetcherWithGlobFiles(t, "./_meta/test/cat_template.*.json", auto_ops_testing.SetupTemplateErrorsServer(CatTemplatePath, templatePathPrefix), useNamedMetricSet, func(t *testing.T, data metricset.FetcherData[[]CatTemplate]) {
		require.Error(t, data.Error)
		require.Equal(t, 1, len(data.Reporter.GetEvents()))

		// Check error event
		event := data.Reporter.GetEvents()[0]
		_, ok := event.MetricSetFields["error"].(autoopsevents.ErrEvent)
		require.True(t, ok, "error field should be of type error.ErrEvent")
	})
}
