// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !integration
// +build !integration

package license

import (
	"testing"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/auto_ops_testing"
	autoopsevents "github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/events"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/metricset"
)

var (
	setupLicenseErrorServer = auto_ops_testing.SetupDataErrorServer(LicensePath)
	setupSuccessfulServer   = auto_ops_testing.SetupSuccessfulServer(LicensePath)
	useNamedMetricSet       = auto_ops_testing.UseNamedMetricSet(LicenseMetricsSet)
)

func TestSuccessfulFetch(t *testing.T) {
	metricset.RunTestsForFetcherWithGlobFiles(t, "./_meta/test/license.valid*.json", setupSuccessfulServer, useNamedMetricSet, func(t *testing.T, data metricset.FetcherData[map[string]interface{}]) {
		require.NoError(t, data.Error)

		require.Equal(t, 1, len(data.Reporter.GetEvents()))
	})
}

func TestFailedClusterInfoFetch(t *testing.T) {
	metricset.RunTestsForFetcherWithGlobFiles(t, "./_meta/test/license.valid*.json", auto_ops_testing.SetupClusterInfoErrorServer, useNamedMetricSet, func(t *testing.T, data metricset.FetcherData[map[string]interface{}]) {
		require.ErrorContains(t, data.Error, "failed to get cluster info from cluster, license metricset")
	})
}

func TestFailedLicenseFetch(t *testing.T) {
	metricset.RunTestsForFetcherWithGlobFiles(t, "./_meta/test/license.valid*.json", setupLicenseErrorServer, useNamedMetricSet, func(t *testing.T, data metricset.FetcherData[map[string]interface{}]) {
		require.ErrorContains(t, data.Error, "failed to get data, license metricset")
	})
}

func TestFailedLicenseFetchEventsMapping(t *testing.T) {
	metricset.RunTestsForFetcherWithGlobFiles(t, "./_meta/test/no_*.license*.json", setupSuccessfulServer, useNamedMetricSet, func(t *testing.T, data metricset.FetcherData[map[string]interface{}]) {
		require.Error(t, data.Error)

		// Check error event
		require.Equal(t, 1, len(data.Reporter.GetEvents()))
		event := data.Reporter.GetEvents()[0]
		_, ok := event.MetricSetFields["error"].(autoopsevents.ErrEvent)
		require.True(t, ok, "error field should be of type error.ErrEvent")
	})
}
