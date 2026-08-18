// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !integration
// +build !integration

package license

import (
	"testing"

	"github.com/elastic/beats/v9/x-pack/metricbeat/module/autoops_es/auto_ops_testing"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v9/x-pack/metricbeat/module/autoops_es/metricset"
)

// Expect a valid response from Elasticsearch to create a single event
func TestProperlyHandlesResponse(t *testing.T) {
	metricset.RunTestsForServerlessMetricSetWithGlobFiles(t, "./_meta/test/license.valid_complete*.json", LicenseMetricsSet, eventsMapping, expectValidParsedData)
}

func TestProperlyHandlesResponseWhenNoOptionalFields(t *testing.T) {
	metricset.RunTestsForServerlessMetricSetWithGlobFiles(t, "./_meta/test/license.valid_but_no_optionals*.json", LicenseMetricsSet, eventsMapping, expectValidParsedDataWithoutOptionalFields)
}

// Tests that License reported with no errors
func expectValidParsedData(t *testing.T, data metricset.FetcherData[map[string]interface{}]) {
	require.NoError(t, data.Error)
	require.Equal(t, 0, len(data.Reporter.GetErrors()))
	require.Equal(t, 1, len(data.Reporter.GetEvents()))

	event := data.Reporter.GetEvents()[0]

	expected := mapstr.M{
		"license": mapstr.M{
			"status":                "active",
			"uid":                   "cbff45e7-c553-41f7-ae4f-9205eabd80xx",
			"type":                  "trial",
			"issue_date":            "2018-10-20T22:05:12.332Z",
			"issue_date_in_millis":  int64(1540073112332),
			"expiry_date":           "2018-11-19T22:05:12.332Z",
			"expiry_date_in_millis": int64(1542665112332),
			"max_nodes":             float64(1000),
			"max_resource_units":    nil,
			"issued_to":             "test",
			"issuer":                "elasticsearch",
			"start_date_in_millis":  int64(-1),
		},
	}

	for key, expectedValue := range expected {
		actualValue := auto_ops_testing.GetObjectValue(event.MetricSetFields, key)
		require.EqualValues(t, expectedValue, actualValue, "Field %s does not match", key)
	}
}

// Tests that LicenseMetricsSet reported with no errors (without optional fields)
func expectValidParsedDataWithoutOptionalFields(t *testing.T, data metricset.FetcherData[map[string]interface{}]) {
	require.NoError(t, data.Error)
	require.Equal(t, 0, len(data.Reporter.GetErrors()))
	require.Equal(t, 1, len(data.Reporter.GetEvents()))

	event := data.Reporter.GetEvents()[0]

	expected := mapstr.M{
		"license": mapstr.M{
			"status":                "active",
			"uid":                   "cbff45e7-c553-41f7-ae4f-9205eabd80xx",
			"type":                  "trial",
			"issue_date":            "2018-10-20T22:05:12.332Z",
			"expiry_date":           "2018-11-19T22:05:12.332Z",
			"expiry_date_in_millis": int64(1542665112332),
			"max_resource_units":    nil,
			"issued_to":             "test",
			"issuer":                "elasticsearch",
		},
	}

	for key, expectedValue := range expected {
		actualValue := auto_ops_testing.GetObjectValue(event.MetricSetFields, key)
		require.EqualValues(t, expectedValue, actualValue, "Field %s does not match", key)
	}
}
