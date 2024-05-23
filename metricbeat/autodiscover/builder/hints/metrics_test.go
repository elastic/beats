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

package hints

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/docker/docker/pkg/ioutils"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-autodiscover/bus"
	"github.com/elastic/elastic-agent-libs/keystore"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestMain(m *testing.M) {
	InitializeModule()
}

func TestGenerateHints(t *testing.T) {
	tests := []struct {
		message string
		event   bus.Event
		len     int
		result  []mapstr.M
	}{
		{
			message: "Empty event hints should return empty config",
			event: bus.Event{
				"host": "1.2.3.4",
				"kubernetes": mapstr.M{
					"container": mapstr.M{
						"name": "foobar",
						"id":   "abc",
					},
				},
				"docker": mapstr.M{
					"container": mapstr.M{
						"name": "foobar",
						"id":   "abc",
					},
				},
			},
			len:    0,
			result: []mapstr.M{},
		},
		{
			message: "Hints without host should return nothing",
			event: bus.Event{
				"hints": mapstr.M{
					"metrics": mapstr.M{
						"module": "mockmodule",
					},
				},
			},
			len:    0,
			result: []mapstr.M{},
		},
		{
			message: "Hints without matching port should return nothing",
			event: bus.Event{
				"host": "1.2.3.4",
				"port": 9090,
				"hints": mapstr.M{
					"metrics": mapstr.M{
						"module": "mockmoduledefaults",
						"hosts":  "${data.host}:8888",
					},
				},
			},
			len:    0,
			result: []mapstr.M{},
		},
		{
			message: "Hints with multiple hosts return only the matching one",
			event: bus.Event{
				"host": "1.2.3.4",
				"port": 9090,
				"hints": mapstr.M{
					"metrics": mapstr.M{
						"module": "mockmoduledefaults",
						"hosts":  "${data.host}:8888,${data.host}:9090",
					},
				},
			},
			len: 1,
			result: []mapstr.M{
				{
					"module":     "mockmoduledefaults",
					"metricsets": []string{"default"},
					"timeout":    "3s",
					"period":     "1m",
					"enabled":    true,
					"hosts":      []interface{}{"1.2.3.4:9090"},
					"processors": []interface{}{},
				},
			},
		},
		{
			message: "Hints with multiple hosts return only the one with the template",
			event: bus.Event{
				"host": "1.2.3.4",
				"port": 9090,
				"hints": mapstr.M{
					"metrics": mapstr.M{
						"module": "mockmoduledefaults",
						"hosts":  "${data.host}:8888,${data.host}:${data.port}",
					},
				},
			},
			len: 1,
			result: []mapstr.M{
				{
					"module":     "mockmoduledefaults",
					"metricsets": []string{"default"},
					"timeout":    "3s",
					"period":     "1m",
					"enabled":    true,
					"hosts":      []interface{}{"1.2.3.4:9090"},
					"processors": []interface{}{},
				},
			},
		},
		{
			message: "Only module hint should return all metricsets",
			event: bus.Event{
				"host": "1.2.3.4",
				"hints": mapstr.M{
					"metrics": mapstr.M{
						"module": "mockmodule",
					},
				},
			},
			len: 1,
			result: []mapstr.M{
				{
					"module":     "mockmodule",
					"metricsets": []string{"one", "two"},
					"timeout":    "3s",
					"period":     "1m",
					"enabled":    true,
					"processors": []interface{}{},
				},
			},
		},
		{
			message: "Metricsets hint works",
			event: bus.Event{
				"host": "1.2.3.4",
				"hints": mapstr.M{
					"metrics": mapstr.M{
						"module":     "mockmodule",
						"metricsets": "one",
					},
				},
			},
			len: 1,
			result: []mapstr.M{
				{
					"module":     "mockmodule",
					"metricsets": []string{"one"},
					"timeout":    "3s",
					"period":     "1m",
					"enabled":    true,
					"processors": []interface{}{},
				},
			},
		},
		{
			message: "Only module, it should return defaults",
			event: bus.Event{
				"host": "1.2.3.4",
				"hints": mapstr.M{
					"metrics": mapstr.M{
						"module": "mockmoduledefaults",
					},
				},
			},
			len: 1,
			result: []mapstr.M{
				{
					"module":     "mockmoduledefaults",
					"metricsets": []string{"default"},
					"timeout":    "3s",
					"period":     "1m",
					"enabled":    true,
					"processors": []interface{}{},
				},
			},
		},
		{
			message: "Module defined in modules as a JSON string should return a config",
			event: bus.Event{
				"host": "1.2.3.4",
				"hints": mapstr.M{
					"metrics": mapstr.M{
						"raw": "{\"enabled\":true,\"metricsets\":[\"default\"],\"module\":\"mockmoduledefaults\",\"period\":\"1m\",\"timeout\":\"3s\"}",
					},
				},
			},
			len: 1,
			result: []mapstr.M{
				{
					"module":     "mockmoduledefaults",
					"metricsets": []string{"default"},
					"timeout":    "3s",
					"period":     "1m",
					"enabled":    true,
				},
			},
		},
		{
			message: "Module, namespace, host hint should return valid config with port should return hosts for " +
				"docker host network scenario",
			event: bus.Event{
				"host": "1.2.3.4",
				"hints": mapstr.M{
					"metrics": mapstr.M{
						"module":    "mockmoduledefaults",
						"namespace": "test",
						"hosts":     "${data.host}:9090",
					},
				},
			},
			len: 1,
			result: []mapstr.M{
				{
					"module":     "mockmoduledefaults",
					"namespace":  "test",
					"metricsets": []string{"default"},
					"timeout":    "3s",
					"period":     "1m",
					"enabled":    true,
					"hosts":      []interface{}{"1.2.3.4:9090"},
					"processors": []interface{}{},
				},
			},
		},
		{
			message: "Module with processor config must return an module having the processor defined",
			event: bus.Event{
				"host": "1.2.3.4",
				"hints": mapstr.M{
					"metrics": mapstr.M{
						"module":    "mockmoduledefaults",
						"namespace": "test",
						"hosts":     "${data.host}:9090",
						"processors": mapstr.M{
							"add_locale": mapstr.M{
								"abbrevation": "MST",
							},
						},
					},
				},
			},
			len: 1,
			result: []mapstr.M{
				{
					"module":     "mockmoduledefaults",
					"namespace":  "test",
					"metricsets": []string{"default"},
					"timeout":    "3s",
					"period":     "1m",
					"enabled":    true,
					"hosts":      []interface{}{"1.2.3.4:9090"},
					"processors": []interface{}{
						map[string]interface{}{
							"add_locale": map[string]interface{}{
								"abbrevation": "MST",
							},
						},
					},
				},
			},
		},
		{
			message: "Module with data.host defined and a zero port should not return a config",
			event: bus.Event{
				"host": "1.2.3.4",
				"port": 0,
				"hints": mapstr.M{
					"metrics": mapstr.M{
						"module":    "mockmoduledefaults",
						"namespace": "test",
						"hosts":     "${data.host}:9090",
					},
				},
			},
			len: 0,
		},
		{
			message: "Module, namespace, host hint should return valid config",
			event: bus.Event{
				"host": "1.2.3.4",
				"port": 9090,
				"hints": mapstr.M{
					"metrics": mapstr.M{
						"module":    "mockmoduledefaults",
						"namespace": "test",
						"hosts":     "${data.host}:9090",
					},
				},
			},
			len: 1,
			result: []mapstr.M{
				{
					"module":     "mockmoduledefaults",
					"namespace":  "test",
					"metricsets": []string{"default"},
					"hosts":      []interface{}{"1.2.3.4:9090"},
					"timeout":    "3s",
					"period":     "1m",
					"enabled":    true,
					"processors": []interface{}{},
				},
			},
		},
		{
			message: "Module, namespace, host hint shouldn't return when port isn't the same has hint",
			event: bus.Event{
				"host": "1.2.3.4",
				"port": 80,
				"hints": []mapstr.M{
					{
						"metrics": mapstr.M{
							"module":    "mockmoduledefaults",
							"namespace": "test",
							"hosts":     "${data.host}:8080",
						},
					},
				},
			},
			len:    0,
			result: []mapstr.M{},
		},
		{
			message: "Non http URLs with valid host port combination should return a valid config",
			event: bus.Event{
				"host": "1.2.3.4",
				"port": 3306,
				"hints": mapstr.M{
					"metrics": mapstr.M{
						"module":    "mockmoduledefaults",
						"namespace": "test",
						"hosts":     "tcp(${data.host}:3306)/",
					},
				},
			},
			len: 1,
			result: []mapstr.M{
				{
					"module":     "mockmoduledefaults",
					"namespace":  "test",
					"metricsets": []string{"default"},
					"hosts":      []interface{}{"tcp(1.2.3.4:3306)/"},
					"timeout":    "3s",
					"period":     "1m",
					"enabled":    true,
					"processors": []interface{}{},
				},
			},
		},
		{
			message: "Named port in the hints should return the corresponding container port",
			event: bus.Event{
				"host":  "1.2.3.4",
				"ports": mapstr.M{"some": 3306},
				"hints": mapstr.M{
					"metrics": mapstr.M{
						"module":    "mockmoduledefaults",
						"namespace": "test",
						"hosts":     "${data.host}:${data.ports.some}",
					},
				},
			},
			len: 1,
			result: []mapstr.M{
				{
					"module":     "mockmoduledefaults",
					"namespace":  "test",
					"metricsets": []string{"default"},
					"hosts":      []interface{}{"1.2.3.4:3306"},
					"timeout":    "3s",
					"period":     "1m",
					"enabled":    true,
					"processors": []interface{}{},
				},
			},
		},
		{
			message: "Named port in the hints should return the corresponding container port for complex hosts",
			event: bus.Event{
				"host":  "1.2.3.4",
				"ports": mapstr.M{"prometheus": 3306},
				"hints": mapstr.M{
					"metrics": mapstr.M{
						"module":    "mockmoduledefaults",
						"namespace": "test",
						"hosts":     "http://${data.host}:${data.ports.prometheus}/metrics",
					},
				},
			},
			len: 1,
			result: []mapstr.M{
				{
					"module":     "mockmoduledefaults",
					"namespace":  "test",
					"metricsets": []string{"default"},
					"hosts":      []interface{}{"http://1.2.3.4:3306/metrics"},
					"timeout":    "3s",
					"period":     "1m",
					"enabled":    true,
					"processors": []interface{}{},
				},
			},
		},
		{
			message: "data.port in the hints should return the corresponding container port",
			event: bus.Event{
				"host":  "1.2.3.4",
				"port":  3306,
				"ports": mapstr.M{"prometheus": 3306},
				"hints": mapstr.M{
					"metrics": mapstr.M{
						"module":    "mockmoduledefaults",
						"namespace": "test",
						"hosts":     "${data.host}:${data.port}",
					},
				},
			},
			len: 1,
			result: []mapstr.M{
				{
					"module":     "mockmoduledefaults",
					"namespace":  "test",
					"metricsets": []string{"default"},
					"hosts":      []interface{}{"1.2.3.4:3306"},
					"timeout":    "3s",
					"period":     "1m",
					"enabled":    true,
					"processors": []interface{}{},
				},
			},
		},
		{
			message: "Module with multiple sets of hints must return the right configs",
			event: bus.Event{
				"host": "1.2.3.4",
				"hints": mapstr.M{
					"metrics": mapstr.M{
						"1": mapstr.M{
							"module":    "mockmoduledefaults",
							"namespace": "test",
							"hosts":     "${data.host}:9090",
						},
						"2": mapstr.M{
							"module":    "mockmoduledefaults",
							"namespace": "test1",
							"hosts":     "${data.host}:9090/fake",
						},
					},
				},
			},
			len: 2,
			result: []mapstr.M{
				{
					"module":     "mockmoduledefaults",
					"namespace":  "test",
					"metricsets": []string{"default"},
					"timeout":    "3s",
					"period":     "1m",
					"enabled":    true,
					"hosts":      []interface{}{"1.2.3.4:9090"},
					"processors": []interface{}{},
				},
				{
					"module":     "mockmoduledefaults",
					"namespace":  "test1",
					"metricsets": []string{"default"},
					"timeout":    "3s",
					"period":     "1m",
					"enabled":    true,
					"hosts":      []interface{}{"1.2.3.4:9090/fake"},
					"processors": []interface{}{},
				},
			},
		},
		{
			message: "Module with multiple hosts returns the right number of hints. Pod level hints need to be one per host",
			event: bus.Event{
				"host": "1.2.3.4",
				"hints": mapstr.M{
					"metrics": mapstr.M{
						"module":    "mockmoduledefaults",
						"namespace": "test",
						"hosts":     "${data.host}:9090, ${data.host}:9091",
					},
				},
			},
			len: 2,
			result: []mapstr.M{
				{
					"module":     "mockmoduledefaults",
					"namespace":  "test",
					"metricsets": []string{"default"},
					"timeout":    "3s",
					"period":     "1m",
					"enabled":    true,
					"hosts":      []interface{}{"1.2.3.4:9090"},
					"processors": []interface{}{},
				},
				{
					"module":     "mockmoduledefaults",
					"namespace":  "test",
					"metricsets": []string{"default"},
					"timeout":    "3s",
					"period":     "1m",
					"enabled":    true,
					"hosts":      []interface{}{"1.2.3.4:9091"},
					"processors": []interface{}{},
				},
			},
		},
		{
			message: "Module with multiple hosts and an exposed port creates a config for just the exposed port",
			event: bus.Event{
				"host": "1.2.3.4",
				"port": 9091,
				"hints": mapstr.M{
					"metrics": mapstr.M{
						"module":    "mockmoduledefaults",
						"namespace": "test",
						"hosts":     "${data.host}:9090, ${data.host}:9091",
					},
				},
			},
			len: 1,
			result: []mapstr.M{
				{
					"module":     "mockmoduledefaults",
					"namespace":  "test",
					"metricsets": []string{"default"},
					"timeout":    "3s",
					"period":     "1m",
					"enabled":    true,
					"hosts":      []interface{}{"1.2.3.4:9091"},
					"processors": []interface{}{},
				},
			},
		},
		{
			message: "exclude/exclude in metrics filters are parsed as a list",
			event: bus.Event{
				"host": "1.2.3.4",
				"hints": mapstr.M{
					"metrics": mapstr.M{
						"module":    "prometheus",
						"namespace": "test",
						"hosts":     "${data.host}:9090",
						"metrics_filters": mapstr.M{
							"exclude": "foo, bar",
							"include": "xxx, yyy",
						},
					},
				},
			},
			len: 1,
			result: []mapstr.M{
				{
					"module":     "prometheus",
					"namespace":  "test",
					"metricsets": []string{"collector"},
					"timeout":    "3s",
					"period":     "1m",
					"enabled":    true,
					"hosts":      []interface{}{"1.2.3.4:9090"},
					"processors": []interface{}{},
					"metrics_filters": map[string]interface{}{
						"exclude": []interface{}{"foo", "bar"},
						"include": []interface{}{"xxx", "yyy"},
					},
				},
			},
		},
	}
	for _, test := range tests {
		mockRegister := mb.NewRegister()
		mockRegister.MustAddMetricSet("mockmodule", "one", NewMockMetricSet, mb.DefaultMetricSet())
		mockRegister.MustAddMetricSet("mockmodule", "two", NewMockMetricSet, mb.DefaultMetricSet())
		mockRegister.MustAddMetricSet("mockmoduledefaults", "default", NewMockMetricSet, mb.DefaultMetricSet())
		mockRegister.MustAddMetricSet("mockmoduledefaults", "other", NewMockMetricSet)
		mockRegister.MustAddMetricSet("prometheus", "collector", NewMockMetricSet)

		m := metricHints{
			Key:      defaultConfig().Key,
			Registry: mockRegister,
			logger:   logp.NewLogger("hints.builder"),
		}
		cfgs := m.CreateConfig(test.event)
		assert.Equal(t, len(cfgs), test.len, test.message)

		// The check below helps skipping config validation if there is no config supposed to be emitted.
		if len(cfgs) == 0 {
			continue
		}
		configs := make([]mapstr.M, 0)
		for _, cfg := range cfgs {
			config := mapstr.M{}
			err := cfg.Unpack(&config)
			ok := assert.Nil(t, err, test.message)
			if !ok {
				break
			}
			// metricsets order is random, order it for tests
			if v, err := config.GetValue("metricsets"); err == nil {
				if msets, ok := v.([]interface{}); ok {
					metricsets := make([]string, len(msets))
					for i, v := range msets {
						var ok bool
						metricsets[i], ok = v.(string)
						assert.Truef(t, ok, "Failed to convert metricset: %d=%v", i, metricsets[i])
					}
					sort.Strings(metricsets)
					config["metricsets"] = metricsets
				}
			}
			configs = append(configs, config)
		}
		assert.Equal(t, test.result, configs, test.message)

	}
}

func TestGenerateHintsDoesNotAccessGlobalKeystore(t *testing.T) {
	path := getTemporaryKeystoreFile()
	defer os.Remove(path)
	// store the secret
	keystore := createAnExistingKeystore(t, path, "stored_secret")
	os.Setenv("PASSWORD", "env_secret")

	tests := []struct {
		message string
		event   bus.Event
		len     int
		result  mapstr.M
	}{
		{
			message: "Module, namespace, host hint should return valid config",
			event: bus.Event{
				"host": "1.2.3.4",
				"port": 9090,
				"hints": mapstr.M{
					"metrics": mapstr.M{
						"module":   "mockmoduledefaults",
						"hosts":    "${data.host}:9090",
						"password": "${PASSWORD}",
					},
				},
				"keystore": keystore,
			},
			len: 1,
			result: mapstr.M{
				"module":     "mockmoduledefaults",
				"metricsets": []string{"default"},
				"hosts":      []interface{}{"1.2.3.4:9090"},
				"timeout":    "3s",
				"period":     "1m",
				"enabled":    true,
				"password":   "env_secret",
				"processors": []interface{}{},
			},
		},
	}
	for _, test := range tests {
		mockRegister := mb.NewRegister()
		mockRegister.MustAddMetricSet("mockmoduledefaults", "default", NewMockMetricSet, mb.DefaultMetricSet())

		m := metricHints{
			Key:      defaultConfig().Key,
			Registry: mockRegister,
			logger:   logp.NewLogger("hints.builder"),
		}
		cfgs := m.CreateConfig(test.event)
		assert.Equal(t, len(cfgs), test.len)
		if len(cfgs) != 0 {
			config := mapstr.M{}
			err := cfgs[0].Unpack(&config)
			assert.Nil(t, err, test.message)

			// metricsets order is random, order it for tests
			if v, err := config.GetValue("metricsets"); err == nil {
				if msets, ok := v.([]interface{}); ok {
					metricsets := make([]string, len(msets))
					for i, v := range msets {
						var ok bool
						metricsets[i], ok = v.(string)
						assert.Truef(t, ok, "Failed to convert metricset: %d=%v", i, metricsets[i])
					}
					sort.Strings(metricsets)
					config["metricsets"] = metricsets
				}
			}

			assert.Equal(t, test.result, config, test.message)
		}

	}
}

type MockMetricSet struct {
	mb.BaseMetricSet
}

func NewMockMetricSet(base mb.BaseMetricSet) (mb.MetricSet, error) {
	return &MockMetricSet{}, nil
}

func (ms *MockMetricSet) Fetch(report mb.ReporterV2) {

}

type MockPrometheus struct {
	*MockMetricSet
}

func NewMockPrometheus(base mb.BaseMetricSet) (mb.MetricSet, error) {
	return &MockPrometheus{}, nil
}

// create a keystore with an existing key
// `PASSWORD` with the value of `secret` variable.
func createAnExistingKeystore(t *testing.T, path string, secret string) keystore.Keystore {
	t.Helper()
	keyStore, err := keystore.NewFileKeystore(path)
	// Fail fast in the test suite
	if err != nil {
		panic(err)
	}

	writableKeystore, err := keystore.AsWritableKeystore(keyStore)
	if err != nil {
		panic(err)
	}

	assert.NoError(t, writableKeystore.Store("PASSWORD", []byte(secret)))
	assert.NoError(t, writableKeystore.Save())
	return keyStore
}

// create a temporary file on disk to save the keystore.
func getTemporaryKeystoreFile() string {
	path, err := ioutils.TempDir("", "testing")
	if err != nil {
		panic(err)
	}
	return filepath.Join(path, "keystore")
}
