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

package mb

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestLightMetricSetRegistration(t *testing.T) {
	cases := map[string]struct {
		module    string
		metricSet string
		isDefault bool
		fail      bool
	}{
		"metricset is registered": {
			module:    "foo",
			metricSet: "bar",
			fail:      false,
		},
		"metricset is registered and is default": {
			module:    "foo",
			metricSet: "bar",
			isDefault: true,
			fail:      false,
		},
		"module is not registered": {
			module:    "notexists",
			metricSet: "notexists",
			fail:      true,
		},
		"metricset is not registered": {
			module:    "foo",
			metricSet: "notexists",
			fail:      true,
		},
	}

	fakeMetricSetFactory := func(b BaseMetricSet) (MetricSet, error) { return &b, nil }

	moduleName := "foo"
	metricSetName := "bar"
	lightMetricSetName := "metricset"
	lightModuleName := "module"

	r := NewRegister()
	r.MustAddMetricSet(moduleName, metricSetName, fakeMetricSetFactory)

	for title, c := range cases {
		t.Run(title, func(t *testing.T) {
			ms := LightMetricSet{
				Name:    lightMetricSetName,
				Module:  lightModuleName,
				Default: c.isDefault,
			}
			ms.Input.Module = c.module
			ms.Input.MetricSet = c.metricSet
			ms.Input.Defaults = mapstr.M{
				"query": mapstr.M{
					"extra": "something",
				},
			}

			registration, err := ms.Registration(r)
			if c.fail {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)

			// Check that registration has the light metricset settings
			assert.Equal(t, c.metricSet, registration.Name)
			assert.Equal(t, c.isDefault, registration.IsDefault)

			// Check that calling the factory with a registered base module:
			// - Does not modify original base module
			// - Does the proper overrides in the resulting metricset
			bm := baseModule(t, r, moduleName, metricSetName)
			moduleConfigBefore := bm.Module().Config().String()
			metricSet, err := registration.Factory(bm)

			assert.Equal(t, moduleConfigBefore, bm.Module().Config().String(),
				"original base module config should not change")
			require.NoError(t, err)
			require.NotNil(t, metricSet)

			assert.Equal(t, lightModuleName, metricSet.Module().Name())
			assert.Equal(t, lightMetricSetName, metricSet.Name())

			expectedQuery := QueryParams{
				"default": "foo",
				"extra":   "something",
			}
			query := metricSet.Module().Config().Query
			assert.Equal(t, expectedQuery, query)
		})
	}
}

func baseModule(t *testing.T, r *Register, module, metricSet string) BaseMetricSet {
	origRegistration, err := r.metricSetRegistration(module, metricSet)
	require.NoError(t, err)

	c := DefaultModuleConfig()
	c.Module = module
	c.MetricSets = []string{metricSet}
	c.Query = QueryParams{"default": "foo"}
	raw, err := conf.NewConfigFrom(c)
	require.NoError(t, err)
	baseModule, err := newBaseModuleFromConfig(raw)
	require.NoError(t, err)

	bm := BaseMetricSet{
		name:         "bar",
		module:       &baseModule,
		registration: origRegistration,
	}
	return bm
}
