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

	"github.com/elastic/beats/libbeat/common"
)

func TestLightMetricSetRegistration(t *testing.T) {
	cases := []struct {
		title     string
		module    string
		metricSet string
		isDefault bool
		fail      bool
	}{
		{
			title:     "metricset is registered",
			module:    "foo",
			metricSet: "bar",
			fail:      false,
		},
		{
			title:     "metricset is registered and is default",
			module:    "foo",
			metricSet: "bar",
			isDefault: true,
			fail:      false,
		},
		{
			title:     "module is not registered",
			module:    "notexists",
			metricSet: "notexists",
			fail:      true,
		},
		{
			title:     "metricset is not registered",
			module:    "foo",
			metricSet: "notexists",
			fail:      true,
		},
	}

	fakeMetricSetFactory = func(b BaseMetricSet) (MetricSet, error) { return &b, nil }

	moduleName := "foo"
	metricSetName := "bar"
	lightMetricSetName := "metricset"
	lightModuleName := "module"

	r := NewRegister()
	r.MustAddMetricSet(moduleName, metricSetName, fakeMetricSetFactory)

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			ms := LightMetricSet{
				Name:    lightMetricSetName,
				Module:  lightModuleName,
				Default: c.isDefault,
			}
			ms.Input.Module = c.module
			ms.Input.MetricSet = c.metricSet

			registration, err := ms.Registration(r)
			if c.fail {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)

			// Check that registration has the light metricset settings
			assert.Equal(t, c.metricSet, registration.Name)
			assert.Equal(t, c.isDefault, registration.IsDefault)

			// Check that calling the factory with a registered base module
			// does the proper overrides in the resulting metricset
			bm := baseModule(t, r, moduleName, metricSetName)
			metricSet, err := registration.Factory(bm)
			require.NoError(t, err)
			require.NotNil(t, metricSet)

			assert.Equal(t, lightModuleName, metricSet.Module().Name())
			assert.Equal(t, lightMetricSetName, metricSet.Name())
		})
	}
}

func baseModule(t *testing.T, r *Register, module, metricSet string) BaseMetricSet {
	origRegistration, err := r.metricSetRegistration(module, metricSet)
	require.NoError(t, err)

	c := DefaultModuleConfig()
	c.Module = module
	c.MetricSets = []string{metricSet}
	raw, err := common.NewConfigFrom(c)
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
