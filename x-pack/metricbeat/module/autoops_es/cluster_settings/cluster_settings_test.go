// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !integration
// +build !integration

package cluster_settings

import (
	"testing"

	"github.com/elastic/beats/v9/x-pack/metricbeat/module/autoops_es/auto_ops_testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v9/x-pack/metricbeat/module/autoops_es/metricset"
)

var (
	setupSuccessfulServer = auto_ops_testing.SetupSuccessfulServer(ClusterSettingsPath)
	useNamedMetricSet     = auto_ops_testing.UseNamedMetricSet(ClusterSettingsMetricSet)
)

func TestSuccessfulFetch(t *testing.T) {
	metricset.RunTestsForFetcherWithGlobFiles(t, "./_meta/test/cluster_settings.*.json", setupSuccessfulServer, useNamedMetricSet, func(t *testing.T, data metricset.FetcherData[map[string]interface{}]) {
		require.NoError(t, data.Error)

		require.Equal(t, 1, len(data.Reporter.GetEvents()))
	})
}
