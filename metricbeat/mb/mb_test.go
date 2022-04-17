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

//go:build !integration
// +build !integration

package mb

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/menderesk/beats/v7/libbeat/common"
)

// Reporting V2 MetricSet

type testModule struct {
	BaseModule
	hostParser func(string) (HostData, error)
}

func (m testModule) ParseHost(host string) (HostData, error) {
	return m.hostParser(host)
}

type testMetricSet struct {
	BaseMetricSet
}

func (m *testMetricSet) Fetch(reporter ReporterV2) {}

// ReportingFetcher

type testMetricSetReportingFetcher struct {
	BaseMetricSet
}

func (m *testMetricSetReportingFetcher) Fetch(r Reporter) {}

// PushMetricSet

type testPushMetricSet struct {
	BaseMetricSet
}

func (m *testPushMetricSet) Run(r PushReporter) {}

func TestModuleConfig(t *testing.T) {
	tests := []struct {
		name string
		in   interface{}
		out  ModuleConfig
		err  string
	}{
		{
			name: "string value is not set on required field",
			in:   map[string]interface{}{},
			err:  "string value is not set accessing 'module'",
		},
		{
			name: "valid config",
			in: map[string]interface{}{
				"module":     "example",
				"metricsets": []string{"test"},
			},
			out: ModuleConfig{
				Module:     "example",
				MetricSets: []string{"test"},
				Enabled:    true,
				Period:     time.Second * 10,
				Timeout:    0,
				Query:      nil,
			},
		},
		{
			name: "missing period",
			in: map[string]interface{}{
				"module":     "example",
				"metricsets": []string{"test"},
				"period":     -1,
			},
			err: "negative value accessing 'period'",
		},
		{
			name: "negative timeout",
			in: map[string]interface{}{
				"module":     "example",
				"metricsets": []string{"test"},
				"timeout":    -1,
			},
			err: "negative value accessing 'timeout'",
		},
	}

	for i, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c, err := common.NewConfigFrom(test.in)
			if err != nil {
				t.Fatal(err)
			}

			unpackedConfig := DefaultModuleConfig()
			err = c.Unpack(&unpackedConfig)
			if err != nil && test.err == "" {
				t.Errorf("unexpected error while unpacking in testcase %d: %v", i, err)
				return
			}
			if test.err != "" {
				if err != nil {
					assert.Contains(t, err.Error(), test.err, "testcase %d", i)
				} else {
					t.Errorf("expected error '%v' in testcase %d", test.err, i)
				}
				return
			}

			assert.Equal(t, test.out, unpackedConfig)
		})
	}
}

// TestModuleConfigDefaults validates that the default values are not changed.
// Any changes to this test case are probably indicators of non-backwards
// compatible changes affect all modules (including community modules).
func TestModuleConfigDefaults(t *testing.T) {
	c, err := common.NewConfigFrom(map[string]interface{}{
		"module":     "mymodule",
		"metricsets": []string{"mymetricset"},
	})
	if err != nil {
		t.Fatal(err)
	}

	mc := DefaultModuleConfig()
	err = c.Unpack(&mc)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, true, mc.Enabled)
	assert.Equal(t, time.Second*10, mc.Period)
	assert.Equal(t, time.Second*0, mc.Timeout)
	assert.Empty(t, mc.Hosts)
}

// TestNewModulesDuplicateHosts verifies that an error is returned by
// NewModules if any module configuration contains duplicate hosts.
func TestNewModulesDuplicateHosts(t *testing.T) {
	r := newTestRegistry(t)

	c := newConfig(t, map[string]interface{}{
		"module":     moduleName,
		"metricsets": []string{metricSetName},
		"hosts":      []string{"a", "b", "a"},
	})

	_, _, err := NewModule(c, r)
	assert.Error(t, err)
}

// TestNewModulesWithDefaultMetricSet verifies that the default MetricSet is
// instantiated when no metricsets are specified in the config.
func TestNewModulesWithDefaultMetricSet(t *testing.T) {
	r := newTestRegistry(t, DefaultMetricSet())

	c := newConfig(t, map[string]interface{}{
		"module": moduleName,
	})

	_, metricSets, err := NewModule(c, r)
	if err != nil {
		t.Fatal(err)
	}
	if assert.Len(t, metricSets, 1) {
		assert.Equal(t, metricSetName, metricSets[0].Name())
	}
}

func TestNewModulesHostParser(t *testing.T) {
	const (
		name = "HostParser"
		host = "example.com"
		uri  = "http://" + host
	)

	r := newTestRegistry(t)

	factory := func(base BaseMetricSet) (MetricSet, error) {
		return &testMetricSet{BaseMetricSet: base}, nil
	}

	hostParser := func(m Module, rawHost string) (HostData, error) {
		return HostData{URI: uri, Host: host}, nil
	}

	if err := r.AddMetricSet(moduleName, name, factory, hostParser); err != nil {
		t.Fatal(err)
	}

	t.Run("MetricSet without HostParser", func(t *testing.T) {
		ms := newTestMetricSet(t, r, map[string]interface{}{
			"module":     moduleName,
			"metricsets": []string{metricSetName},
			"hosts":      []string{uri},
		})

		// The URI is passed through in the Host() and HostData().URI.
		assert.Equal(t, uri, ms.Host())
		assert.Equal(t, HostData{URI: uri}, ms.HostData())
	})

	t.Run("MetricSet with HostParser", func(t *testing.T) {
		ms := newTestMetricSet(t, r, map[string]interface{}{
			"module":     moduleName,
			"metricsets": []string{name},
			"hosts":      []string{uri},
		})

		// The URI is passed through in the Host() and HostData().URI.
		assert.Equal(t, host, ms.Host())
		assert.Equal(t, HostData{URI: uri, Host: host}, ms.HostData())
	})
}

func TestNewModulesMetricSetTypes(t *testing.T) {
	r := newTestRegistry(t)

	factory := func(base BaseMetricSet) (MetricSet, error) {
		return &testMetricSet{base}, nil
	}

	name := "ReportingMetricSetV2"
	if err := r.AddMetricSet(moduleName, name, factory); err != nil {
		t.Fatal(err)
	}

	t.Run(name+" MetricSet", func(t *testing.T) {
		ms := newTestMetricSet(t, r, map[string]interface{}{
			"module":     moduleName,
			"metricsets": []string{name},
		})
		_, ok := ms.(ReportingMetricSetV2)
		assert.True(t, ok, name+" not implemented")
	})

	factory = func(base BaseMetricSet) (MetricSet, error) {
		return &testMetricSetReportingFetcher{base}, nil
	}

	name = "ReportingFetcher"
	if err := r.AddMetricSet(moduleName, name, factory); err != nil {
		t.Fatal(err)
	}

	t.Run(name+" MetricSet", func(t *testing.T) {
		ms := newTestMetricSet(t, r, map[string]interface{}{
			"module":     moduleName,
			"metricsets": []string{name},
		})

		_, ok := ms.(ReportingMetricSet)
		assert.True(t, ok, name+" not implemented")
	})

	factory = func(base BaseMetricSet) (MetricSet, error) {
		return &testPushMetricSet{base}, nil
	}

	name = "Push"
	if err := r.AddMetricSet(moduleName, name, factory); err != nil {
		t.Fatal(err)
	}

	t.Run(name+" MetricSet", func(t *testing.T) {
		ms := newTestMetricSet(t, r, map[string]interface{}{
			"module":     moduleName,
			"metricsets": []string{name},
		})
		_, ok := ms.(PushMetricSet)
		assert.True(t, ok, name+" not implemented")
	})
}

// TestNewBaseModuleFromModuleConfigStruct tests the creation a new BaseModule.
func TestNewBaseModuleFromModuleConfigStruct(t *testing.T) {
	moduleConf := DefaultModuleConfig()
	moduleConf.Module = moduleName
	moduleConf.MetricSets = []string{metricSetName}

	c := newConfig(t, moduleConf)

	baseModule, err := newBaseModuleFromConfig(c)
	assert.NoError(t, err)

	assert.Equal(t, moduleName, baseModule.Name())
	assert.Equal(t, moduleName, baseModule.Config().Module)
	assert.Equal(t, true, baseModule.Config().Enabled)
	assert.Equal(t, time.Second*10, baseModule.Config().Period)
	assert.Equal(t, time.Second*10, baseModule.Config().Timeout)
	assert.Empty(t, baseModule.Config().Hosts)
}

func newTestRegistry(t testing.TB, metricSetOptions ...MetricSetOption) *Register {
	r := NewRegister()

	if err := r.AddModule(moduleName, DefaultModuleFactory); err != nil {
		t.Fatal(err)
	}

	factory := func(base BaseMetricSet) (MetricSet, error) {
		return &testMetricSet{base}, nil
	}

	if err := r.addMetricSet(moduleName, metricSetName, factory, metricSetOptions...); err != nil {
		t.Fatal(err)
	}

	return r
}

func newTestMetricSet(t testing.TB, r *Register, config map[string]interface{}) MetricSet {
	_, metricsets, err := NewModule(newConfig(t, config), r)
	if err != nil {
		t.Fatal(err)
	}
	if !assert.Len(t, metricsets, 1) {
		assert.FailNow(t, "invalid number of metricsets")
	}

	return metricsets[0]
}

func newConfig(t testing.TB, moduleConfig interface{}) *common.Config {
	config, err := common.NewConfigFrom(moduleConfig)
	if err != nil {
		t.Fatal(err)
	}
	return config
}

func TestModuleConfigQueryParams(t *testing.T) {
	qp := QueryParams{
		"stringKey": "value",
		"intKey":    10,
		"floatKey":  11.5,
		"boolKey":   true,
		"nullKey":   nil,
		"arKey":     []interface{}{1, 2},
	}

	res := qp.String()

	expectedValues := []string{"stringKey=value", "intKey=10", "floatKey=11.5", "boolKey=true", "nullKey=", "arKey=1", "arKey=2"}
	for _, expected := range expectedValues {
		assert.Contains(t, res, expected)
	}

	assert.NotContains(t, res, "?")
	assert.NotContains(t, res, "%")
	assert.NotEqual(t, "&", res[0])
	assert.NotEqual(t, "&", res[len(res)-1])
}

func TestBaseModuleWithConfig(t *testing.T) {
	mockRegistry := NewRegister()

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
		newConfig      metricSetConfig
		expectedConfig metricSetConfig
		expectedErrMsg string
	}{
		"metricsets": {
			metricSetConfig{
				MetricSets: []string{"qux", "baz", "bar"},
			},
			metricSetConfig{
				Module:     moduleName,
				MetricSets: []string{"qux", "baz", "bar"},
			},
			"",
		},
		"module_name": {
			metricSetConfig{
				Module: "new_test_module",
			},
			metricSetConfig{},
			fmt.Sprintf("cannot change module name from %v to %v", moduleName, "new_test_module"),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			initConfig := metricSetConfig{
				Module:     moduleName,
				MetricSets: []string{"foo", "bar"},
			}

			m, _, err := NewModule(common.MustNewConfigFrom(initConfig), mockRegistry)
			require.NoError(t, err)

			bm, ok := m.(*BaseModule)
			if !ok {
				require.Fail(t, "expecting module to be base module")
			}

			newBM, err := bm.WithConfig(*common.MustNewConfigFrom(test.newConfig))

			if err == nil {
				var actualNewConfig metricSetConfig
				err = newBM.UnpackConfig(&actualNewConfig)
				require.NoError(t, err)
				require.Equal(t, test.expectedConfig, actualNewConfig)
			} else {
				require.Equal(t, test.expectedErrMsg, err.Error())
				require.Nil(t, newBM)
			}
		})
	}
}

type mockMetricSet struct {
	BaseMetricSet
}

func (m *mockMetricSet) Fetch(r ReporterV2) error { return nil }

type metricSetConfig struct {
	Module     string   `config:"module"`
	MetricSets []string `config:"metricsets"`
}

func mockMetricSetFactory(base BaseMetricSet) (MetricSet, error) {
	return &mockMetricSet{base}, nil
}
