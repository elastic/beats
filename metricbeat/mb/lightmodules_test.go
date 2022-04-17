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
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/menderesk/beats/v7/libbeat/common"
	"github.com/menderesk/beats/v7/libbeat/logp"
	_ "github.com/menderesk/beats/v7/libbeat/processors/add_id"
)

// TestLightModulesAsModuleSource checks that registry correctly lists
// metricsets when used with light modules
func TestLightModulesAsModuleSource(t *testing.T) {
	logp.TestingSetup()

	type testMetricSet struct {
		name       string
		module     string
		isDefault  bool
		hostParser HostParser
	}

	cases := map[string]struct {
		registered                []testMetricSet
		expectedMetricSets        map[string][]string
		expectedDefaultMetricSets map[string][]string
	}{
		"no registered modules": {
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
		"same module registered (mixed modules case)": {
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
		"some metricsets registered": {
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

	fakeMetricSetFactory := func(base BaseMetricSet) (MetricSet, error) {
		return &base, nil
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

	for title, c := range cases {
		t.Run(title, func(t *testing.T) {
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
		register := NewRegister()
		r := NewLightModulesSource("testdata/lightmodules")
		t.Run(c.name, func(t *testing.T) {
			_, err := r.loadModule(register, c.name)
			if c.err {
				assert.Error(t, err)
			}
			assert.Equal(t, c.exists, r.HasModule(c.name))
		})
	}
}

func TestNewModuleFromConfig(t *testing.T) {
	logp.TestingSetup()

	cases := map[string]struct {
		config         common.MapStr
		err            bool
		expectedOption string
		expectedQuery  QueryParams
		expectedPeriod time.Duration
	}{
		"normal module": {
			config:         common.MapStr{"module": "foo", "metricsets": []string{"bar"}},
			expectedOption: "default",
			expectedQuery:  nil,
		},
		"light module": {
			config:         common.MapStr{"module": "service", "metricsets": []string{"metricset"}},
			expectedOption: "test",
			expectedQuery:  nil,
		},
		"light module default metricset": {
			config:         common.MapStr{"module": "service"},
			expectedOption: "test",
			expectedQuery:  nil,
		},
		"light module override option": {
			config:         common.MapStr{"module": "service", "option": "overriden"},
			expectedOption: "overriden",
			expectedQuery:  nil,
		},
		"light module with query": {
			config:         common.MapStr{"module": "service", "query": common.MapStr{"param": "foo"}},
			expectedOption: "test",
			expectedQuery:  QueryParams{"param": "foo"},
		},
		"light module with custom period": {
			config:         common.MapStr{"module": "service", "period": "42s"},
			expectedOption: "test",
			expectedPeriod: 42 * time.Second,
			expectedQuery:  nil,
		},
		"light module is broken": {
			config: common.MapStr{"module": "broken"},
			err:    true,
		},
		"light metric set doesn't exist": {
			config: common.MapStr{"module": "service", "metricsets": []string{"notexists"}},
			err:    true,
		},
		"disabled light module": {
			config: common.MapStr{"module": "service", "enabled": false},
			err:    true,
		},
		"mixed module with standard and light metricsets": {
			config:         common.MapStr{"module": "mixed", "metricsets": []string{"standard", "light"}},
			expectedOption: "default",
			expectedQuery:  nil,
		},
		"mixed module with unregistered and light metricsets": {
			config: common.MapStr{"module": "mixedbroken", "metricsets": []string{"unregistered", "light"}},
			err:    true,
		},
	}

	r := NewRegister()
	r.MustAddMetricSet("foo", "bar", newMetricSetWithOption)
	r.MustAddMetricSet("foo", "light", newMetricSetWithOption)
	r.MustAddMetricSet("mixed", "standard", newMetricSetWithOption)
	r.SetSecondarySource(NewLightModulesSource("testdata/lightmodules"))

	for title, c := range cases {
		t.Run(title, func(t *testing.T) {
			config, err := common.NewConfigFrom(c.config)
			require.NoError(t, err)

			module, metricSets, err := NewModule(config, r)
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
				t.Run(ms.Name(), func(t *testing.T) {
					ms, ok := ms.(*metricSetWithOption)
					require.True(t, ok)
					assert.Equal(t, c.expectedOption, ms.Option)
					assert.Equal(t, c.expectedQuery, ms.Module().Config().Query)
					expectedPeriod := c.expectedPeriod
					if expectedPeriod == 0 {
						expectedPeriod = DefaultModuleConfig().Period
					}
					assert.Equal(t, expectedPeriod, ms.Module().Config().Period)
				})
			}
		})
	}
}

func TestLightMetricSet_VerifyHostDataURI(t *testing.T) {
	const hostEndpoint = "ceph-restful:8003"
	const sampleHttpsEndpoint = "https://" + hostEndpoint

	r := NewRegister()
	r.MustAddMetricSet("http", "json", newMetricSetWithOption,
		WithHostParser(func(module Module, host string) (HostData, error) {
			u, err := url.Parse(host)
			if err != nil {
				return HostData{}, err
			}
			return HostData{
				Host: u.Host,
				URI:  host,
			}, nil
		}))
	r.SetSecondarySource(NewLightModulesSource("testdata/lightmodules"))

	config, err := common.NewConfigFrom(
		common.MapStr{
			"module":     "httpextended",
			"metricsets": []string{"extends"},
			"hosts":      []string{sampleHttpsEndpoint},
		})
	require.NoError(t, err)

	_, metricSets, err := NewModule(config, r)
	require.NoError(t, err)
	require.Len(t, metricSets, 1)

	assert.Equal(t, hostEndpoint, metricSets[0].Host())
	assert.Equal(t, sampleHttpsEndpoint, metricSets[0].HostData().URI)
}

func TestLightMetricSet_WithoutHostParser(t *testing.T) {
	const sampleHttpsEndpoint = "https://ceph-restful:8003"

	r := NewRegister()
	r.MustAddMetricSet("http", "json", newMetricSetWithOption)
	r.SetSecondarySource(NewLightModulesSource("testdata/lightmodules"))

	config, err := common.NewConfigFrom(
		common.MapStr{
			"module":     "httpextended",
			"metricsets": []string{"extends"},
			"hosts":      []string{sampleHttpsEndpoint},
		})
	require.NoError(t, err)

	_, metricSets, err := NewModule(config, r)
	require.NoError(t, err)
	require.Len(t, metricSets, 1)

	assert.Equal(t, sampleHttpsEndpoint, metricSets[0].Host())
	assert.Equal(t, sampleHttpsEndpoint, metricSets[0].HostData().URI)
}

func TestLightMetricSet_VerifyHostDataURI_NonParsableHost(t *testing.T) {
	const (
		postgresHost     = "host1:5432"
		postgresEndpoint = "postgres://user1:pass@host1:5432?connect_timeout=2"
		postgresParsed   = "connect_timeout=3 host=host1 password=pass port=5432 user=user1"
	)

	r := NewRegister()
	r.MustAddMetricSet("http", "json", newMetricSetWithOption,
		WithHostParser(func(module Module, host string) (HostData, error) {
			return HostData{
				Host: postgresHost,
				URI:  postgresParsed,
			}, nil
		}))
	r.SetSecondarySource(NewLightModulesSource("testdata/lightmodules"))

	config, err := common.NewConfigFrom(
		common.MapStr{
			"module":     "httpextended",
			"metricsets": []string{"extends"},
			"hosts":      []string{postgresEndpoint},
		})
	require.NoError(t, err)

	_, metricSets, err := NewModule(config, r)
	require.NoError(t, err)
	require.Len(t, metricSets, 1)

	assert.Equal(t, postgresHost, metricSets[0].Host())
	assert.Equal(t, postgresParsed, metricSets[0].HostData().URI)
}

func TestNewModulesCallModuleFactory(t *testing.T) {
	logp.TestingSetup()

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

func TestProcessorsForMetricSet_UnknownModule(t *testing.T) {
	r := NewRegister()
	source := NewLightModulesSource("testdata/lightmodules")
	procs, err := source.ProcessorsForMetricSet(r, "nonexisting", "fake")
	require.Error(t, err)
	require.Nil(t, procs)
}

func TestProcessorsForMetricSet_UnknownMetricSet(t *testing.T) {
	r := NewRegister()
	source := NewLightModulesSource("testdata/lightmodules")
	procs, err := source.ProcessorsForMetricSet(r, "unpack", "nonexisting")
	require.Error(t, err)
	require.Nil(t, procs)
}

func TestProcessorsForMetricSet_ProcessorsRead(t *testing.T) {
	r := NewRegister()
	source := NewLightModulesSource("testdata/lightmodules")
	procs, err := source.ProcessorsForMetricSet(r, "unpack", "withprocessors")
	require.NoError(t, err)
	require.NotNil(t, procs)
	require.Len(t, procs.List, 1)
}

func TestProcessorsForMetricSet_ListModules(t *testing.T) {
	source := NewLightModulesSource("testdata/lightmodules")
	modules, err := source.Modules()
	require.NoError(t, err)

	// Check that regular file in directory is not listed as module
	require.FileExists(t, "testdata/lightmodules/regular_file")
	assert.NotContains(t, modules, "regular_file")

	expectedModules := []string{
		"broken",
		"httpextended",
		"mixed",
		"mixedbroken",
		"service",
		"unpack",
	}
	assert.ElementsMatch(t, expectedModules, modules, "Modules found: %v", modules)
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
