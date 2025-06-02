// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !integration
// +build !integration

package index_template

import (
	"net/url"
	"strings"
	"testing"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/auto_ops_testing"
	autoopsevents "github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/events"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/metricset"
)

var (
	setupIndexTemplateErrorServer = auto_ops_testing.SetupDataErrorServer(IndexTemplatePath)
	setupEmptySuccessfulServer    = auto_ops_testing.SetupSuccessfulServer(IndexTemplatePath)
	useNamedMetricSet             = auto_ops_testing.UseNamedMetricSet(IndexTemplateMetricSet)
)

func TestIndexTemplatePath(t *testing.T) {
	parsedURL, err := url.Parse(IndexTemplatePath)
	require.NoError(t, err)

	require.True(t, strings.HasPrefix(parsedURL.Path, "/_index_template"), "path %s does not start with /_index_template", parsedURL.Path)

	params := parsedURL.Query()
	filterPath := params.Get("filter_path")
	actualFields := strings.Split(filterPath, ",")

	expectedFields := map[string]string{
		"name":          "index_templates.name",
		"managed":       "index_templates.index_template._meta.managed",
		"index_pattern": "index_templates.index_template.index_patterns",
	}

	for field, fullPath := range expectedFields {
		require.Containsf(t, actualFields, fullPath, "expected filter for '%s' with path '%s' not found in '%v'", field, fullPath, actualFields)
	}
}

func TestEmptySuccessfulFetch(t *testing.T) {
	metricset.RunTestsForFetcherWithGlobFiles(t, "./_meta/test/empty.*.json", setupEmptySuccessfulServer, useNamedMetricSet, func(t *testing.T, data metricset.FetcherData[IndexTemplates]) {
		require.NoError(t, data.Error)
		require.Equal(t, 0, len(data.Reporter.GetEvents()))
	})
}

func TestSuccessfulFetch(t *testing.T) {
	metricset.RunTestsForFetcherWithGlobFiles(t, "./_meta/test/index_template.*.json", auto_ops_testing.SetupSuccessfulTemplateServer(IndexTemplatePath, templatePathPrefix, getTemplateResponse), useNamedMetricSet, func(t *testing.T, data metricset.FetcherData[IndexTemplates]) {
		require.NoError(t, data.Error)

		// 1 <= len(...)
		require.LessOrEqual(t, 1, len(data.Reporter.GetEvents()))
	})
}

func TestFailedClusterInfoFetch(t *testing.T) {
	metricset.RunTestsForFetcherWithGlobFiles(t, "./_meta/test/index_template.*.json", auto_ops_testing.SetupClusterInfoErrorServer, useNamedMetricSet, func(t *testing.T, data metricset.FetcherData[IndexTemplates]) {
		require.ErrorContains(t, data.Error, "failed to get cluster info from cluster, index_template metricset")
	})
}

func TestFailedTasksFetch(t *testing.T) {
	metricset.RunTestsForFetcherWithGlobFiles(t, "./_meta/test/index_template.*.json", setupIndexTemplateErrorServer, useNamedMetricSet, func(t *testing.T, data metricset.FetcherData[IndexTemplates]) {
		require.ErrorContains(t, data.Error, "failed to get data, index_template metricset")
	})
}

func TestFailedTasksFetchEventsMapping(t *testing.T) {
	// Note: it will fail due to an inner error looking up templates
	metricset.RunTestsForFetcherWithGlobFiles(t, "./_meta/test/index_template.*.json", auto_ops_testing.SetupTemplateErrorsServer(IndexTemplatePath, templatePathPrefix), useNamedMetricSet, func(t *testing.T, data metricset.FetcherData[IndexTemplates]) {
		require.Error(t, data.Error)
		require.Equal(t, 1, len(data.Reporter.GetEvents()))

		// Check error event
		event := data.Reporter.GetEvents()[0]
		_, ok := event.MetricSetFields["error"].(autoopsevents.ErrEvent)
		require.True(t, ok, "error field should be of type error.ErrEvent")
	})
}
