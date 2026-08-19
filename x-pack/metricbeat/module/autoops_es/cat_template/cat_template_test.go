// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !integration
// +build !integration

package cat_template

import (
	"testing"

	"github.com/elastic/beats/v9/x-pack/metricbeat/module/autoops_es/auto_ops_testing"
	"github.com/elastic/beats/v9/x-pack/metricbeat/module/autoops_es/metricset"

	"github.com/stretchr/testify/require"
)

var (
	setupEmptySuccessfulServer = auto_ops_testing.SetupSuccessfulServer(CatTemplatePath)
	useNamedMetricSet          = auto_ops_testing.UseNamedMetricSet(CatTemplateMetricSet)
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
