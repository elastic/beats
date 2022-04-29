// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package elastic

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/metricbeat/mb"
	conf "github.com/elastic/elastic-agent-libs/config"
)

func TestMakeXPackMonitoringIndexName(t *testing.T) {
	tests := []struct {
		Name     string
		Product  Product
		Expected string
	}{
		{
			"Elasticsearch monitoring index",
			Elasticsearch,
			".monitoring-es-8-mb",
		},
		{
			"Kibana monitoring index",
			Kibana,
			".monitoring-kibana-8-mb",
		},
		{
			"Logstash monitoring index",
			Logstash,
			".monitoring-logstash-8-mb",
		},
		{
			"Beats monitoring index",
			Beats,
			".monitoring-beats-8-mb",
		},
	}

	for _, test := range tests {
		name := fmt.Sprintf("Test naming %v", test.Name)
		t.Run(name, func(t *testing.T) {
			indexName := MakeXPackMonitoringIndexName(test.Product)
			assert.Equal(t, test.Expected, indexName)
		})
	}
}

type MockReporterV2 struct {
	mb.ReporterV2
}

func (MockReporterV2) Event(event mb.Event) bool {
	return true
}

var currentErr error // This hack is necessary because the Error method below cannot receive the type *MockReporterV2

func (m MockReporterV2) Error(err error) bool {
	currentErr = err
	return true
}

func TestReportErrorForMissingField(t *testing.T) {
	field := "some.missing.field"
	r := MockReporterV2{}
	err := ReportErrorForMissingField(field, Elasticsearch, r)

	expectedError := fmt.Errorf("Could not find field '%v' in Elasticsearch API response", field)
	assert.Equal(t, expectedError, err)
	assert.Equal(t, expectedError, currentErr)
}

func TestFixTimestampField(t *testing.T) {
	tests := []struct {
		Name          string
		OriginalValue map[string]interface{}
		ExpectedValue map[string]interface{}
	}{
		{
			"converts float64s in scientific notation to ints",
			map[string]interface{}{
				"foo": 1.571284349e+09,
			},
			map[string]interface{}{
				"foo": 1571284349,
			},
		},
		{
			"converts regular notation float64s to ints",
			map[string]interface{}{
				"foo": float64(1234),
			},
			map[string]interface{}{
				"foo": 1234,
			},
		},
		{
			"ignores missing fields",
			map[string]interface{}{
				"bar": 12345,
			},
			map[string]interface{}{
				"bar": 12345,
			},
		},
		{
			"leaves strings untouched",
			map[string]interface{}{
				"foo": "bar",
			},
			map[string]interface{}{
				"foo": "bar",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			err := FixTimestampField(test.OriginalValue, "foo")
			assert.NoError(t, err)
			assert.Equal(t, test.ExpectedValue, test.OriginalValue)
		})
	}
}

func TestConfigureModule(t *testing.T) {
	mockRegistry := mb.NewRegister()

	const moduleName = "test_module"

	err := mockRegistry.AddMetricSet(moduleName, "foo", mockMetricSetFactory)
	require.NoError(t, err)
	err = mockRegistry.AddMetricSet(moduleName, "bar", mockMetricSetFactory)
	require.NoError(t, err)
	err = mockRegistry.AddMetricSet(moduleName, "qux", mockMetricSetFactory)
	require.NoError(t, err)
	err = mockRegistry.AddMetricSet(moduleName, "baz", mockMetricSetFactory)
	require.NoError(t, err)

	tests := map[string]struct {
		initConfig             metricSetConfig
		xpackEnabledMetricsets []string
		newConfig              metricSetConfig
	}{
		"no_xpack_enabled": {
			metricSetConfig{
				Module:     moduleName,
				MetricSets: []string{"foo", "bar"},
			},
			[]string{"baz", "qux", "foo"},
			metricSetConfig{
				Module:     moduleName,
				MetricSets: []string{"foo", "bar"},
			},
		},
		"xpack_enabled": {
			metricSetConfig{
				Module:       moduleName,
				XPackEnabled: true,
				MetricSets:   []string{"foo", "bar"},
			},
			[]string{"baz", "qux", "foo"},
			metricSetConfig{
				Module:       moduleName,
				XPackEnabled: true,
				MetricSets:   []string{"baz", "qux", "foo"},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			cfg := conf.MustNewConfigFrom(test.initConfig)
			m, _, err := mb.NewModule(cfg, mockRegistry)
			require.NoError(t, err)

			bm, ok := m.(*mb.BaseModule)
			if !ok {
				require.Fail(t, "expecting module to be base module")
			}

			newM, err := NewModule(bm, test.xpackEnabledMetricsets, logp.L())
			require.NoError(t, err)

			var newConfig metricSetConfig
			err = newM.UnpackConfig(&newConfig)
			require.NoError(t, err)
			require.Equal(t, test.newConfig, newConfig)
		})
	}
}

type mockMetricSet struct {
	mb.BaseMetricSet
}

func (m *mockMetricSet) Fetch(r mb.ReporterV2) error { return nil }

type metricSetConfig struct {
	Module       string   `config:"module"`
	MetricSets   []string `config:"metricsets"`
	XPackEnabled bool     `config:"xpack.enabled"`
}

func mockMetricSetFactory(base mb.BaseMetricSet) (mb.MetricSet, error) {
	return &mockMetricSet{base}, nil
}
