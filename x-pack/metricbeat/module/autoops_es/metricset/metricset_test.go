// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !integration
// +build !integration

package metricset

import (
	"testing"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/auto_ops_testing"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/events"

	"github.com/stretchr/testify/require"

	s "github.com/elastic/beats/v7/libbeat/common/schema"
	c "github.com/elastic/beats/v7/libbeat/common/schema/mapstriface"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/utils"
)

type testObjectType struct {
	Values []map[string]interface{} `json:"items"`
}

const (
	NAME = "test_metricset"
	PATH = "/_fake/path"
)

var (
	schema = s.Schema{
		"name":  c.Str("name", s.Required),
		"value": c.Int("value", s.Required),
	}
	setupClusterSettingsErrorServer = auto_ops_testing.SetupDataErrorServer(PATH)
	setupSuccessfulServer           = auto_ops_testing.SetupSuccessfulServer(PATH)
	useTestMetricSet                = UseNamedMetricSet(NAME)
)

// init registers the MetricSet with the central registry as soon as the program
// starts. The New function will be called later to instantiate an instance of
// the MetricSet for each host defined in the module's configuration. After the
// MetricSet has been created then Fetch will begin to be called periodically.
func init() {
	AddAutoOpsMetricSet(NAME, PATH, eventsMapping)
}

func eventsMapping(r mb.ReporterV2, info *utils.ClusterInfo, data *testObjectType) error {
	for _, value := range data.Values {
		parsed, err := schema.Apply(value)

		if err != nil {
			return err
		}

		r.Event(events.CreateEventWithRandomTransactionId(info, parsed))
	}

	return nil
}

func TestSuccessfulFetch(t *testing.T) {
	RunTestsForFetcherWithGlobFiles(t, "./_meta/test/success.*.json", setupSuccessfulServer, useTestMetricSet, func(t *testing.T, data FetcherData[testObjectType]) {
		require.NoError(t, data.Error)

		require.Equal(t, 2, len(data.Reporter.GetEvents()))

		event1 := data.Reporter.GetEvents()[0]

		require.Equal(t, "obj1", GetObjectValue(event1.MetricSetFields, "name"))
		require.EqualValues(t, 1, GetObjectValue(event1.MetricSetFields, "value"))

		event2 := data.Reporter.GetEvents()[1]

		require.Equal(t, "obj2", GetObjectValue(event2.MetricSetFields, "name"))
		require.EqualValues(t, 2, GetObjectValue(event2.MetricSetFields, "value"))
	})
}

func TestFailedClusterInfoFetch(t *testing.T) {
	RunTestsForFetcherWithGlobFiles(t, "./_meta/test/success.*.json", auto_ops_testing.SetupClusterInfoErrorServer, useTestMetricSet, func(t *testing.T, data FetcherData[testObjectType]) {
		require.ErrorContains(t, data.Error, "failed to get cluster info from cluster, "+NAME+" metricset")
	})
}

func TestFailedClusterDataFetch(t *testing.T) {
	RunTestsForFetcherWithGlobFiles(t, "./_meta/test/success.*.json", setupClusterSettingsErrorServer, useTestMetricSet, func(t *testing.T, data FetcherData[testObjectType]) {
		require.ErrorContains(t, data.Error, "failed to get data, "+NAME+" metricset")
	})
}

func TestFailedClusterDataFetchEventsMapping(t *testing.T) {
	RunTestsForFetcherWithGlobFiles(t, "./_meta/test/no_*.error.*.json", setupSuccessfulServer, useTestMetricSet, func(t *testing.T, data FetcherData[testObjectType]) {
		require.Error(t, data.Error)
	})
}
