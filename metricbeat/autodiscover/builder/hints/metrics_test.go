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

	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/libbeat/common/bus"
	"github.com/elastic/beats/v8/libbeat/keystore"
	"github.com/elastic/beats/v8/libbeat/logp"
	"github.com/elastic/beats/v8/metricbeat/mb"
)

func TestGenerateHints(t *testing.T) {
	tests := []struct {
		message string
		event   bus.Event
		len     int
		result  []common.MapStr
	}{
		{
			message: "Empty event hints should return empty config",
			event: bus.Event{
				"host": "1.2.3.4",
				"kubernetes": common.MapStr{
					"container": common.MapStr{
						"name": "foobar",
						"id":   "abc",
					},
				},
				"docker": common.MapStr{
					"container": common.MapStr{
						"name": "foobar",
						"id":   "abc",
					},
				},
			},
			len:    0,
			result: []common.MapStr{},
		},
		{
			message: "Hints without host should return nothing",
			event: bus.Event{
				"hints": common.MapStr{
					"metrics": common.MapStr{
						"module": "mockmodule",
					},
				},
			},
			len:    0,
			result: []common.MapStr{},
		},
		{
			message: "Hints without matching port should return nothing",
			event: bus.Event{
				"host": "1.2.3.4",
				"port": 9090,
				"hints": common.MapStr{
					"metrics": common.MapStr{
						"module": "mockmoduledefaults",
						"hosts":  "${data.host}:8888",
					},
				},
			},
			len:    0,
			result: []common.MapStr{},
		},
		{
			message: "Hints with multiple hosts return only the matching one",
			event: bus.Event{
				"host": "1.2.3.4",
				"port": 9090,
				"hints": common.MapStr{
					"metrics": common.MapStr{
						"module": "mockmoduledefaults",
						"hosts":  "${data.host}:8888,${data.host}:9090",
					},
				},
			},
			len: 1,
			result: []common.MapStr{
				{
					"module":     "mockmoduledefaults",
					"metricsets": []string{"default"},
					"timeout":    "3s",
					"period":     "1m",
					"enabled":    true,
					"hosts":      []interface{}{"1.2.3.4:9090"},
				},
			},
		},
		{
			message: "Hints with multiple hosts return only the one with the template",
			event: bus.Event{
				"host": "1.2.3.4",
				"port": 9090,
				"hints": common.MapStr{
					"metrics": common.MapStr{
						"module": "mockmoduledefaults",
						"hosts":  "${data.host}:8888,${data.host}:${data.port}",
					},
				},
			},
			len: 1,
			result: []common.MapStr{
				{
					"module":     "mockmoduledefaults",
					"metricsets": []string{"default"},
					"timeout":    "3s",
					"period":     "1m",
					"enabled":    true,
					"hosts":      []interface{}{"1.2.3.4:9090"},
				},
			},
		},
		{
			message: "Only module hint should return all metricsets",
			event: bus.Event{
				"host": "1.2.3.4",
				"hints": common.MapStr{
					"metrics": common.MapStr{
						"module": "mockmodule",
					},
				},
			},
			len: 1,
			result: []common.MapStr{
				{
					"module":     "mockmodule",
					"metricsets": []string{"one", "two"},
					"timeout":    "3s",
					"period":     "1m",
					"enabled":    true,
				},
			},
		},
		{
			message: "Metricsets hint works",
			event: bus.Event{
				"host": "1.2.3.4",
				"hints": common.MapStr{
					"metrics": common.MapStr{
						"module":     "mockmodule",
						"metricsets": "one",
					},
				},
			},
			len: 1,
			result: []common.MapStr{
				{
					"module":     "mockmodule",
					"metricsets": []string{"one"},
					"timeout":    "3s",
					"period":     "1m",
					"enabled":    true,
				},
			},
		},
		{
			message: "Only module, it should return defaults",
			event: bus.Event{
				"host": "1.2.3.4",
				"hints": common.MapStr{
					"metrics": common.MapStr{
						"module": "mockmoduledefaults",
					},
				},
			},
			len: 1,
			result: []common.MapStr{
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
			message: "Module defined in modules as a JSON string should return a config",
			event: bus.Event{
				"host": "1.2.3.4",
				"hints": common.MapStr{
					"metrics": common.MapStr{
						"raw": "{\"enabled\":true,\"metricsets\":[\"default\"],\"module\":\"mockmoduledefaults\",\"period\":\"1m\",\"timeout\":\"3s\"}",
					},
				},
			},
			len: 1,
			result: []common.MapStr{
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
				"hints": common.MapStr{
					"metrics": common.MapStr{
						"module":    "mockmoduledefaults",
						"namespace": "test",
						"hosts":     "${data.host}:9090",
					},
				},
			},
			len: 1,
			result: []common.MapStr{
				{
					"module":     "mockmoduledefaults",
					"namespace":  "test",
					"metricsets": []string{"default"},
					"timeout":    "3s",
					"period":     "1m",
					"enabled":    true,
					"hosts":      []interface{}{"1.2.3.4:9090"},
				},
			},
		},
		{
			message: "Module with processor config must return an module having the processor defined",
			event: bus.Event{
				"host": "1.2.3.4",
				"hints": common.MapStr{
					"metrics": common.MapStr{
						"module":    "mockmoduledefaults",
						"namespace": "test",
						"hosts":     "${data.host}:9090",
						"processors": common.MapStr{
							"add_locale": common.MapStr{
								"abbrevation": "MST",
							},
						},
					},
				},
			},
			len: 1,
			result: []common.MapStr{
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
				"hints": common.MapStr{
					"metrics": common.MapStr{
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
				"hints": common.MapStr{
					"metrics": common.MapStr{
						"module":    "mockmoduledefaults",
						"namespace": "test",
						"hosts":     "${data.host}:9090",
					},
				},
			},
			len: 1,
			result: []common.MapStr{
				{
					"module":     "mockmoduledefaults",
					"namespace":  "test",
					"metricsets": []string{"default"},
					"hosts":      []interface{}{"1.2.3.4:9090"},
					"timeout":    "3s",
					"period":     "1m",
					"enabled":    true,
				},
			},
		},
		{
			message: "Module, namespace, host hint shouldn't return when port isn't the same has hint",
			event: bus.Event{
				"host": "1.2.3.4",
				"port": 80,
				"hints": []common.MapStr{
					{
						"metrics": common.MapStr{
							"module":    "mockmoduledefaults",
							"namespace": "test",
							"hosts":     "${data.host}:8080",
						},
					},
				},
			},
			len:    0,
			result: []common.MapStr{},
		},
		{
			message: "Non http URLs with valid host port combination should return a valid config",
			event: bus.Event{
				"host": "1.2.3.4",
				"port": 3306,
				"hints": common.MapStr{
					"metrics": common.MapStr{
						"module":    "mockmoduledefaults",
						"namespace": "test",
						"hosts":     "tcp(${data.host}:3306)/",
					},
				},
			},
			len: 1,
			result: []common.MapStr{
				{
					"module":     "mockmoduledefaults",
					"namespace":  "test",
					"metricsets": []string{"default"},
					"hosts":      []interface{}{"tcp(1.2.3.4:3306)/"},
					"timeout":    "3s",
					"period":     "1m",
					"enabled":    true,
				},
			},
		},
		{
			message: "Named port in the hints should return the corresponding container port",
			event: bus.Event{
				"host":  "1.2.3.4",
				"ports": common.MapStr{"some": 3306},
				"hints": common.MapStr{
					"metrics": common.MapStr{
						"module":    "mockmoduledefaults",
						"namespace": "test",
						"hosts":     "${data.host}:${data.ports.some}",
					},
				},
			},
			len: 1,
			result: []common.MapStr{
				{
					"module":     "mockmoduledefaults",
					"namespace":  "test",
					"metricsets": []string{"default"},
					"hosts":      []interface{}{"1.2.3.4:3306"},
					"timeout":    "3s",
					"period":     "1m",
					"enabled":    true,
				},
			},
		},
		{
			message: "Named port in the hints should return the corresponding container port for complex hosts",
			event: bus.Event{
				"host":  "1.2.3.4",
				"ports": common.MapStr{"prometheus": 3306},
				"hints": common.MapStr{
					"metrics": common.MapStr{
						"module":    "mockmoduledefaults",
						"namespace": "test",
						"hosts":     "http://${data.host}:${data.ports.prometheus}/metrics",
					},
				},
			},
			len: 1,
			result: []common.MapStr{
				{
					"module":     "mockmoduledefaults",
					"namespace":  "test",
					"metricsets": []string{"default"},
					"hosts":      []interface{}{"http://1.2.3.4:3306/metrics"},
					"timeout":    "3s",
					"period":     "1m",
					"enabled":    true,
				},
			},
		},
		{
			message: "data.port in the hints should return the corresponding container port",
			event: bus.Event{
				"host":  "1.2.3.4",
				"port":  3306,
				"ports": common.MapStr{"prometheus": 3306},
				"hints": common.MapStr{
					"metrics": common.MapStr{
						"module":    "mockmoduledefaults",
						"namespace": "test",
						"hosts":     "${data.host}:${data.port}",
					},
				},
			},
			len: 1,
			result: []common.MapStr{
				{
					"module":     "mockmoduledefaults",
					"namespace":  "test",
					"metricsets": []string{"default"},
					"hosts":      []interface{}{"1.2.3.4:3306"},
					"timeout":    "3s",
					"period":     "1m",
					"enabled":    true,
				},
			},
		},
		{
			message: "Module with mutliple sets of hints must return the right configs",
			event: bus.Event{
				"host": "1.2.3.4",
				"hints": common.MapStr{
					"metrics": common.MapStr{
						"1": common.MapStr{
							"module":    "mockmoduledefaults",
							"namespace": "test",
							"hosts":     "${data.host}:9090",
						},
						"2": common.MapStr{
							"module":    "mockmoduledefaults",
							"namespace": "test1",
							"hosts":     "${data.host}:9090/fake",
						},
					},
				},
			},
			len: 2,
			result: []common.MapStr{
				{
					"module":     "mockmoduledefaults",
					"namespace":  "test",
					"metricsets": []string{"default"},
					"timeout":    "3s",
					"period":     "1m",
					"enabled":    true,
					"hosts":      []interface{}{"1.2.3.4:9090"},
				},
				{
					"module":     "mockmoduledefaults",
					"namespace":  "test1",
					"metricsets": []string{"default"},
					"timeout":    "3s",
					"period":     "1m",
					"enabled":    true,
					"hosts":      []interface{}{"1.2.3.4:9090/fake"},
				},
			},
		},
		{
			message: "Module with multiple hosts returns the right number of hints. Pod level hints need to be one per host",
			event: bus.Event{
				"host": "1.2.3.4",
				"hints": common.MapStr{
					"metrics": common.MapStr{
						"module":    "mockmoduledefaults",
						"namespace": "test",
						"hosts":     "${data.host}:9090, ${data.host}:9091",
					},
				},
			},
			len: 2,
			result: []common.MapStr{
				{
					"module":     "mockmoduledefaults",
					"namespace":  "test",
					"metricsets": []string{"default"},
					"timeout":    "3s",
					"period":     "1m",
					"enabled":    true,
					"hosts":      []interface{}{"1.2.3.4:9090"},
				},
				{
					"module":     "mockmoduledefaults",
					"namespace":  "test",
					"metricsets": []string{"default"},
					"timeout":    "3s",
					"period":     "1m",
					"enabled":    true,
					"hosts":      []interface{}{"1.2.3.4:9091"},
				},
			},
		},
		{
			message: "Module with multiple hosts and an exposed port creates a config for just the exposed port",
			event: bus.Event{
				"host": "1.2.3.4",
				"port": 9091,
				"hints": common.MapStr{
					"metrics": common.MapStr{
						"module":    "mockmoduledefaults",
						"namespace": "test",
						"hosts":     "${data.host}:9090, ${data.host}:9091",
					},
				},
			},
			len: 1,
			result: []common.MapStr{
				{
					"module":     "mockmoduledefaults",
					"namespace":  "test",
					"metricsets": []string{"default"},
					"timeout":    "3s",
					"period":     "1m",
					"enabled":    true,
					"hosts":      []interface{}{"1.2.3.4:9091"},
				},
			},
		},
		{
			message: "exclude/exclude in metrics filters are parsed as a list",
			event: bus.Event{
				"host": "1.2.3.4",
				"hints": common.MapStr{
					"metrics": common.MapStr{
						"module":    "prometheus",
						"namespace": "test",
						"hosts":     "${data.host}:9090",
						"metrics_filters": common.MapStr{
							"exclude": "foo, bar",
							"include": "xxx, yyy",
						},
					},
				},
			},
			len: 1,
			result: []common.MapStr{
				{
					"module":     "prometheus",
					"namespace":  "test",
					"metricsets": []string{"collector"},
					"timeout":    "3s",
					"period":     "1m",
					"enabled":    true,
					"hosts":      []interface{}{"1.2.3.4:9090"},
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
		configs := make([]common.MapStr, 0)
		for _, cfg := range cfgs {
			config := common.MapStr{}
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
						metricsets[i] = v.(string)
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
	keystore := createAnExistingKeystore(path, "stored_secret")
	os.Setenv("PASSWORD", "env_secret")

	tests := []struct {
		message string
		event   bus.Event
		len     int
		result  common.MapStr
	}{
		{
			message: "Module, namespace, host hint should return valid config",
			event: bus.Event{
				"host": "1.2.3.4",
				"port": 9090,
				"hints": common.MapStr{
					"metrics": common.MapStr{
						"module":   "mockmoduledefaults",
						"hosts":    "${data.host}:9090",
						"password": "${PASSWORD}",
					},
				},
				"keystore": keystore,
			},
			len: 1,
			result: common.MapStr{
				"module":     "mockmoduledefaults",
				"metricsets": []string{"default"},
				"hosts":      []interface{}{"1.2.3.4:9090"},
				"timeout":    "3s",
				"period":     "1m",
				"enabled":    true,
				"password":   "env_secret",
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
			config := common.MapStr{}
			err := cfgs[0].Unpack(&config)
			assert.Nil(t, err, test.message)

			// metricsets order is random, order it for tests
			if v, err := config.GetValue("metricsets"); err == nil {
				if msets, ok := v.([]interface{}); ok {
					metricsets := make([]string, len(msets))
					for i, v := range msets {
						metricsets[i] = v.(string)
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

func (ms *MockMetricSet) Fetch(report mb.Reporter) {

}

type MockPrometheus struct {
	*MockMetricSet
}

func NewMockPrometheus(base mb.BaseMetricSet) (mb.MetricSet, error) {
	return &MockPrometheus{}, nil
}

// create a keystore with an existing key
// `PASSWORD` with the value of `secret` variable.
func createAnExistingKeystore(path string, secret string) keystore.Keystore {
	keyStore, err := keystore.NewFileKeystore(path)
	// Fail fast in the test suite
	if err != nil {
		panic(err)
	}

	writableKeystore, err := keystore.AsWritableKeystore(keyStore)
	if err != nil {
		panic(err)
	}

	writableKeystore.Store("PASSWORD", []byte(secret))
	writableKeystore.Save()
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
