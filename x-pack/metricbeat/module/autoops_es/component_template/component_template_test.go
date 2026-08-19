// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !integration
// +build !integration

package component_template

import (
	"net/url"
	"strings"
	"testing"

	"github.com/elastic/beats/v9/x-pack/metricbeat/module/autoops_es/auto_ops_testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v9/x-pack/metricbeat/module/autoops_es/metricset"
)

var (
	setupEmptySuccessfulServer = auto_ops_testing.SetupSuccessfulServer(ComponentTemplatePath)
	useNamedMetricSet          = auto_ops_testing.UseNamedMetricSet(ComponentTemplateMetricSet)
)

func TestComponentTemplatePath(t *testing.T) {

	parsedURL, err := url.Parse(ComponentTemplatePath)
	require.NoError(t, err)

	require.Truef(t, strings.HasPrefix(parsedURL.Path, "/_component_template"), "path %s does not start with /_component_template", parsedURL.Path)

	params := parsedURL.Query()
	filterPath := params.Get("filter_path")
	actualFields := strings.Split(filterPath, ",")

	expectedFields := map[string]string{
		"name":    "component_templates.name",
		"managed": "component_templates.component_template._meta.managed",
	}

	for field, fullPath := range expectedFields {
		require.Containsf(t, actualFields, fullPath, "expected filter for '%s' as '%s' not found in '%v'", field, fullPath, actualFields)
	}
}

func TestEmptySuccessfulFetch(t *testing.T) {
	metricset.RunTestsForFetcherWithGlobFiles(t, "./_meta/test/empty.*.json", setupEmptySuccessfulServer, useNamedMetricSet, func(t *testing.T, data metricset.FetcherData[ComponentTemplates]) {
		require.NoError(t, data.Error)
		require.Equal(t, 0, len(data.Reporter.GetEvents()))
	})
}

func TestSuccessfulFetch(t *testing.T) {
	metricset.RunTestsForFetcherWithGlobFiles(t, "./_meta/test/component_template.*.json", auto_ops_testing.SetupSuccessfulTemplateServer(ComponentTemplatePath, templatePathPrefix, getTemplateResponse), useNamedMetricSet, func(t *testing.T, data metricset.FetcherData[ComponentTemplates]) {
		require.NoError(t, data.Error)

		// 1 <= len(...)
		require.LessOrEqual(t, 1, len(data.Reporter.GetEvents()))
	})
}
