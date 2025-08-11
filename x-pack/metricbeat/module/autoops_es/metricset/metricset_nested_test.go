// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !integration
// +build !integration

package metricset

import (
	"errors"
	"testing"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/events"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/module/elasticsearch"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/utils"
)

const (
	NESTED_NAME = "test_nested_metricset"
)

var (
	useNestedTestMetricSet = UseNamedMetricSet(NESTED_NAME)
)

// init registers the MetricSet with the central registry as soon as the program
// starts. The New function will be called later to instantiate an instance of
// the MetricSet for each host defined in the module's configuration. After the
// MetricSet has been created then Fetch will begin to be called periodically.
func init() {
	AddNestedAutoOpsMetricSet(NESTED_NAME, PATH, nestedEventsMapping)
}

func nestedEventsMapping(m *elasticsearch.MetricSet, r mb.ReporterV2, info *utils.ClusterInfo, data *testObjectType) error {
	if m == nil {
		return errors.New("missing metricset")
	}

	for _, value := range data.Values {
		parsed, err := schema.Apply(value)

		if err != nil {
			return err
		}

		r.Event(events.CreateEventWithRandomTransactionId(info, parsed))
	}

	return nil
}

func TestNestedSuccessfulFetch(t *testing.T) {
	RunTestsForFetcherWithGlobFiles(t, "./_meta/test/success.*.json", setupSuccessfulServer, useNestedTestMetricSet, func(t *testing.T, data FetcherData[testObjectType]) {
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
