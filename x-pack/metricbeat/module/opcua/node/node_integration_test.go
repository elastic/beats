// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration

package node

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/tests/compose"
	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	"github.com/elastic/beats/v7/pkg/mapstr"
)

func TestFetch(t *testing.T) {
	service := compose.EnsureUp(t, "opcua")

	ms := mbtest.NewReportingMetricSetV2WithContext(t, getConfig(service.Host()))
	metricSet, ok := ms.(*MetricSet)
	require.True(t, ok)
	t.Cleanup(func() { assert.NoError(t, metricSet.Close()) })

	events, errs := mbtest.ReportingFetchV2WithContext(ms)
	require.Empty(t, errs)
	require.Len(t, events, 5)

	byID := map[string]mapstr.M{}
	for _, e := range events {
		id, err := e.MetricSetFields.GetValue("id")
		require.NoError(t, err)
		idString, ok := id.(string)
		require.True(t, ok)
		byID[idString] = e.MetricSetFields
	}

	step := byID["ns=3;s=StepUp"]
	assertValue(t, step, "name", "StepUp")
	assertValue(t, step, "label", "step")
	assertValue(t, step, "data_type", "Uint32")
	assertValue(t, step, "status.severity", "good")
	assertHas(t, step, "value.long")
	assertHas(t, step, "source_timestamp")

	assertHas(t, byID["ns=3;s=AlternatingBoolean"], "value.boolean")
	assertHas(t, byID["ns=3;s=RandomSignedInt32"], "value.long")
	assertHas(t, byID["ns=3;s=PositiveTrendData"], "value.double")

	unknown := byID["ns=3;s=DoesNotExist"]
	assertValue(t, unknown, "status.severity", "bad")
	assertValue(t, unknown, "status.name", "StatusBadNodeIDUnknown")
	_, err := unknown.GetValue("value")
	assert.Error(t, err, "a node with a bad status must not have a value")

	// A second fetch reuses the session.
	events, errs = mbtest.ReportingFetchV2WithContext(ms)
	require.Empty(t, errs)
	require.Len(t, events, 5)
}

func TestData(t *testing.T) {
	service := compose.EnsureUp(t, "opcua")

	f := mbtest.NewFetcher(t, getConfig(service.Host()))
	f.WriteEventsCond(t, "", func(e mapstr.M) bool {
		id, _ := e.GetValue("opcua.node.id")
		return id == "ns=3;s=StepUp"
	})
}

func assertValue(t *testing.T, fields mapstr.M, key string, want any) {
	t.Helper()
	got, err := fields.GetValue(key)
	if assert.NoError(t, err, key) {
		assert.Equal(t, want, got, key)
	}
}

func assertHas(t *testing.T, fields mapstr.M, key string) {
	t.Helper()
	_, err := fields.GetValue(key)
	assert.NoError(t, err, key)
}

func getConfig(host string) map[string]any {
	return map[string]any{
		"module":     "opcua",
		"metricsets": []string{"node"},
		"hosts":      []string{"opc.tcp://" + host},
		"period":     "10s",
		"nodes": []map[string]any{
			{"id": "ns=3;s=StepUp", "label": "step"},
			{"id": "ns=3;s=AlternatingBoolean"},
			{"id": "ns=3;s=RandomSignedInt32"},
			{"id": "ns=3;s=PositiveTrendData"},
			{"id": "ns=3;s=DoesNotExist"},
		},
	}
}
