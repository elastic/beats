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

// +build !integration

package mb

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/libbeat/common"
)

// TestLightModulesAsRegistryChild checks that registry correctly lists
// metricsets when used with light modules
func TestLightModulesAsRegistryChild(t *testing.T) {
	type testMetricSet struct {
		name       string
		module     string
		isDefault  bool
		hostParser HostParser
	}

	cases := []struct {
		title                     string
		registered                []testMetricSet
		expectedMetricSets        map[string][]string
		expectedDefaultMetricSets map[string][]string
	}{
		{
			title: "no registered modules",
			expectedMetricSets: map[string][]string{
				"service": []string{"metricset", "nondefault"},
				"broken":  []string{},
				"empty":   []string{},
			},
			expectedDefaultMetricSets: map[string][]string{
				"service": []string{"metricset"},
				"broken":  []string{},
				"empty":   []string{},
			},
		},
		{
			title: "same module registered (mixed modules case)",
			registered: []testMetricSet{
				{name: "other", module: "service"},
			},
			expectedMetricSets: map[string][]string{
				"service": []string{"metricset", "nondefault", "other"},
			},
			expectedDefaultMetricSets: map[string][]string{
				"service": []string{"metricset"},
			},
		},
		{
			title: "some metricsets registered",
			registered: []testMetricSet{
				{name: "other", module: "service"},
				{name: "metricset", module: "something", isDefault: true},
				{name: "metricset", module: "someotherthing"},
			},
			expectedMetricSets: map[string][]string{
				"service":        []string{"metricset", "nondefault", "other"},
				"something":      []string{"metricset"},
				"someotherthing": []string{"metricset"},
			},
			expectedDefaultMetricSets: map[string][]string{
				"service":        []string{"metricset"},
				"something":      []string{"metricset"},
				"someotherthing": []string{},
			},
		},
	}

	newRegistry := func(metricSets []testMetricSet) *Register {
		r := NewRegister()
		for _, m := range metricSets {
			opts := []MetricSetOption{}
			if m.isDefault {
				opts = append(opts, DefaultMetricSet())
			}
			if m.hostParser != nil {
				opts = append(opts, WithHostParser(m.hostParser))
			}
			r.MustAddMetricSet(m.module, m.name, fakeMetricSetFactory, opts...)
		}
		r.SetSecondarySource(NewLightModulesSource("testdata/lightmodules"))
		return r
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			r := newRegistry(c.registered)

			// Check metricsets
			for module, metricSets := range c.expectedMetricSets {
				t.Run("metricsets for "+module, func(t *testing.T) {
					assert.ElementsMatch(t, metricSets, r.MetricSets(module))
				})
			}

			// Check default metricsets
			for module, metricSets := range c.expectedDefaultMetricSets {
				t.Run("default metricsets for "+module, func(t *testing.T) {
					found, err := r.DefaultMetricSets(module)
					assert.ElementsMatch(t, metricSets, found)
					if len(metricSets) == 0 {
						assert.Error(t, err, "error expected when there are no default metricsets")
					}
				})
			}
		})
	}
}

func TestReadModule(t *testing.T) {
	cases := []struct {
		name  string
		found bool
		err   bool
	}{
		{
			name:  "service",
			found: true,
			err:   false,
		},
		{
			name:  "broken",
			found: true,
			err:   true,
		},
		{
			name:  "empty",
			found: false,
			err:   false,
		},
		{
			name:  "notexists",
			found: false,
			err:   false,
		},
	}

	for _, c := range cases {
		r := NewLightModulesSource("testdata/lightmodules")
		t.Run(c.name, func(t *testing.T) {
			_, found, err := r.loadModule(c.name)
			if c.err {
				assert.Error(t, err)
			}
			assert.Equal(t, c.found, found)
		})
	}
}

func TestNewModuleFromConfig(t *testing.T) {
	cases := []struct {
		title          string
		config         common.MapStr
		err            bool
		expectedOption string
	}{
		{
			title:          "normal module",
			config:         common.MapStr{"module": "foo", "metricsets": []string{"bar"}},
			expectedOption: "default",
		},
		{
			title:          "light module",
			config:         common.MapStr{"module": "service", "metricsets": []string{"metricset"}},
			expectedOption: "test",
		},
		{
			title:          "light module default metricset",
			config:         common.MapStr{"module": "service"},
			expectedOption: "test",
		},
		{
			title:          "light module override option",
			config:         common.MapStr{"module": "service", "option": "overriden"},
			expectedOption: "overriden",
		},
		{
			title:  "light module is broken",
			config: common.MapStr{"module": "broken"},
			err:    true,
		},
		{
			title:  "light metric set doesn't exist",
			config: common.MapStr{"module": "service", "metricsets": []string{"notexists"}},
			err:    true,
		},
		{
			title:  "disabled light module",
			config: common.MapStr{"module": "service", "enabled": false},
			err:    true,
		},
	}

	r := NewRegister()
	r.MustAddMetricSet("foo", "bar", newMetricSetWithOption)
	r.SetSecondarySource(NewLightModulesSource("testdata/lightmodules"))

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			config, err := common.NewConfigFrom(c.config)
			require.NoError(t, err)

			module, metricSets, err := NewModule(config, r)
			if c.err {
				assert.Error(t, err)
				return
			}

			assert.Equal(t, c.config["module"].(string), module.Config().Module)
			if metricSetNames, ok := c.config["metricsets"].([]string); ok {
				assert.ElementsMatch(t, metricSetNames, module.Config().MetricSets)
			}

			assert.NotEmpty(t, metricSets)
			assert.NoError(t, err)
			for _, ms := range metricSets {
				ms, ok := ms.(*metricSetWithOption)
				if assert.True(t, ok) {
					assert.Equal(t, c.expectedOption, ms.Option)
				}
			}
		})
	}
}

func TestNewModulesCallModuleFactory(t *testing.T) {
	r := NewRegister()
	r.MustAddMetricSet("foo", "bar", newMetricSetWithOption)
	r.SetSecondarySource(NewLightModulesSource("testdata/lightmodules"))

	called := false
	r.AddModule("foo", func(base BaseModule) (Module, error) {
		called = true
		return DefaultModuleFactory(base)
	})

	config, err := common.NewConfigFrom(common.MapStr{"module": "service"})
	require.NoError(t, err)

	_, _, err = NewModule(config, r)
	assert.NoError(t, err)

	assert.True(t, called, "module factory must be called if registered")
}

type metricSetWithOption struct {
	BaseMetricSet
	Option string
}

func newMetricSetWithOption(base BaseMetricSet) (MetricSet, error) {
	config := struct {
		Option string `config:"option"`
	}{
		Option: "default",
	}
	err := base.Module().UnpackConfig(&config)
	if err != nil {
		return nil, err
	}

	return &metricSetWithOption{
		BaseMetricSet: base,
		Option:        config.Option,
	}, nil
}

func (*metricSetWithOption) Fetch(ReporterV2) error { return nil }
