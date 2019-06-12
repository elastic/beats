// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build !integration

package mb

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
)

// TestLightModulesAsModuleSource checks that registry correctly lists
// metricsets when used with light modules
func TestLightModulesAsModuleSource(t *testing.T) {
	logp.TestingSetup()

	type testMetricSet struct {
		name       string
		module     string
		isDefault  bool
		hostParser mb.HostParser
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

	fakeMetricSetFactory := func(base mb.BaseMetricSet) (mb.MetricSet, error) {
		return &base, nil
	}

	newRegistry := func(metricSets []testMetricSet) *mb.Register {
		r := mb.NewRegister()
		for _, m := range metricSets {
			opts := []mb.MetricSetOption{}
			if m.isDefault {
				opts = append(opts, mb.DefaultMetricSet())
			}
			if m.hostParser != nil {
				opts = append(opts, mb.WithHostParser(m.hostParser))
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
			for module, expected := range c.expectedDefaultMetricSets {
				t.Run("default metricsets for "+module, func(t *testing.T) {
					found, err := r.DefaultMetricSets(module)
					if len(expected) > 0 {
						assert.NoError(t, err)
						assert.ElementsMatch(t, expected, found)
					} else {
						assert.Error(t, err, "error expected when there are no default metricsets")

					}
				})
			}
		})
	}
}

func TestLoadModule(t *testing.T) {
	logp.TestingSetup()

	cases := []struct {
		name   string
		exists bool
		err    bool
	}{
		{
			name:   "service",
			exists: true,
			err:    false,
		},
		{
			name:   "broken",
			exists: true,
			err:    true,
		},
		{
			name:   "empty",
			exists: false,
			err:    false,
		},
		{
			name:   "notexists",
			exists: false,
			err:    false,
		},
	}

	for _, c := range cases {
		r := NewLightModulesSource("testdata/lightmodules")
		t.Run(c.name, func(t *testing.T) {
			_, err := r.loadModule(c.name)
			if c.err {
				assert.Error(t, err)
			}
			assert.Equal(t, c.exists, r.HasModule(c.name))
		})
	}
}

func TestNewModuleFromConfig(t *testing.T) {
	logp.TestingSetup()

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

	r := mb.NewRegister()
	r.MustAddMetricSet("foo", "bar", newMetricSetWithOption)
	r.SetSecondarySource(NewLightModulesSource("testdata/lightmodules"))

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			config, err := common.NewConfigFrom(c.config)
			require.NoError(t, err)

			module, metricSets, err := mb.NewModule(config, r)
			if c.err {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)

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
	logp.TestingSetup()

	r := mb.NewRegister()
	r.MustAddMetricSet("foo", "bar", newMetricSetWithOption)
	r.SetSecondarySource(NewLightModulesSource("testdata/lightmodules"))

	called := false
	r.AddModule("foo", func(base mb.BaseModule) (mb.Module, error) {
		called = true
		return mb.DefaultModuleFactory(base)
	})

	config, err := common.NewConfigFrom(common.MapStr{"module": "service"})
	require.NoError(t, err)

	_, _, err = mb.NewModule(config, r)
	assert.NoError(t, err)

	assert.True(t, called, "module factory must be called if registered")
}

type metricSetWithOption struct {
	mb.BaseMetricSet
	Option string
}

func newMetricSetWithOption(base mb.BaseMetricSet) (mb.MetricSet, error) {
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

func (*metricSetWithOption) Fetch(mb.ReporterV2) error { return nil }
