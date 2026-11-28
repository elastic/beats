// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration

package tag

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/tests/compose"
	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	"github.com/elastic/beats/v7/pkg/mapstr"
)

func TestFetch(t *testing.T) {
	service := compose.EnsureUp(t, "plc4x")

	ms := mbtest.NewReportingMetricSetV2WithContext(t, getConfig(service.Host()))
	metricSet, ok := ms.(*MetricSet)
	require.True(t, ok)
	t.Cleanup(func() { assert.NoError(t, metricSet.Close()) })

	events, errs := mbtest.ReportingFetchV2WithContext(ms)
	require.Empty(t, errs)
	require.Len(t, events, 5)

	byName := map[string]mapstr.M{}
	for _, e := range events {
		name, err := e.MetricSetFields.GetValue("name")
		require.NoError(t, err)
		nameString, ok := name.(string)
		require.True(t, ok)
		byName[nameString] = e.MetricSetFields
	}

	assertValue(t, byName["temperature"], "value.double", 21.5)
	assertValue(t, byName["temperature"], "data_type", "REAL")
	assertValue(t, byName["temperature"], "address", "holding-register:1:REAL")
	assertValue(t, byName["temperature"], "response_code", "OK")
	assertValue(t, byName["offset"], "value.long", int64(-42))
	assertValue(t, byName["counter"], "value.long", int64(1234))
	assertValue(t, byName["input"], "value.long", int64(777))

	invalid := byName["invalid"]
	_, err := invalid.GetValue("value")
	assert.Error(t, err, "a tag that can't be read must not have a value")
	code, err := invalid.GetValue("response_code")
	require.NoError(t, err)
	assert.NotEqual(t, "OK", code)

	// A second fetch reuses the connection.
	events, errs = mbtest.ReportingFetchV2WithContext(ms)
	require.Empty(t, errs)
	require.Len(t, events, 5)
}

func TestData(t *testing.T) {
	service := compose.EnsureUp(t, "plc4x")

	f := mbtest.NewFetcher(t, getConfig(service.Host()))
	f.WriteEventsCond(t, "", func(e mapstr.M) bool {
		name, _ := e.GetValue("plc4x.tag.name")
		return name == "temperature"
	})
}

func assertValue(t *testing.T, fields mapstr.M, key string, want any) {
	t.Helper()
	got, err := fields.GetValue(key)
	if assert.NoError(t, err, key) {
		assert.Equal(t, want, got, key)
	}
}

func getConfig(host string) map[string]any {
	return map[string]any{
		"module":     "plc4x",
		"metricsets": []string{"tag"},
		"hosts":      []string{"modbus-tcp://" + host},
		"period":     "10s",
		"plc_tags": []map[string]any{
			{"address": "holding-register:1:REAL", "name": "temperature"},
			{"address": "holding-register:11:INT", "name": "offset"},
			{"address": "holding-register:12:UINT", "name": "counter"},
			{"address": "input-register:1:UINT", "name": "input"},
			{"address": "holding-register:60000:INT", "name": "invalid"},
		},
	}
}
