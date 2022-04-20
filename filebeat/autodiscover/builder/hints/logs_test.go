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
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/bus"
	"github.com/elastic/beats/v7/libbeat/paths"
)

func TestGenerateHints(t *testing.T) {
	customCfg := common.MustNewConfigFrom(map[string]interface{}{
		"default_config": map[string]interface{}{
			"type": "docker",
			"containers": map[string]interface{}{
				"ids": []string{
					"${data.container.id}",
				},
			},
			"close_timeout": "true",
		},
	})

	defaultCfg := common.NewConfig()

	defaultDisabled := common.MustNewConfigFrom(map[string]interface{}{
		"default_config": map[string]interface{}{
			"enabled": "false",
		},
	})

	tests := []struct {
		msg    string
		config *common.Config
		event  bus.Event
		len    int
		result []common.MapStr
	}{
		{
			msg:    "Default config is correct",
			config: defaultCfg,
			event: bus.Event{
				"host": "1.2.3.4",
				"kubernetes": common.MapStr{
					"container": common.MapStr{
						"name": "foobar",
						"id":   "abc",
					},
				},
				"container": common.MapStr{
					"name": "foobar",
					"id":   "abc",
				},
			},
			len: 1,
			result: []common.MapStr{
				{
					"paths": []interface{}{"/var/lib/docker/containers/abc/*-json.log"},
					"type":  "container",
				},
			},
		},
		{
			msg:    "Config disabling works",
			config: defaultDisabled,
			event: bus.Event{
				"host": "1.2.3.4",
				"kubernetes": common.MapStr{
					"container": common.MapStr{
						"name": "foobar",
						"id":   "abc",
					},
				},
				"container": common.MapStr{
					"name": "foobar",
					"id":   "abc",
				},
			},
			len:    0,
			result: []common.MapStr{},
		},
		{
			msg:    "Hint to enable when disabled by default works",
			config: defaultDisabled,
			event: bus.Event{
				"host": "1.2.3.4",
				"kubernetes": common.MapStr{
					"container": common.MapStr{
						"name": "foobar",
						"id":   "abc",
					},
				},
				"container": common.MapStr{
					"name": "foobar",
					"id":   "abc",
				},
				"hints": common.MapStr{
					"logs": common.MapStr{
						"enabled":       "true",
						"exclude_lines": "^test2, ^test3",
					},
				},
			},
			len: 1,
			result: []common.MapStr{
				{
					"type":          "container",
					"paths":         []interface{}{"/var/lib/docker/containers/abc/*-json.log"},
					"exclude_lines": []interface{}{"^test2", "^test3"},
				},
			},
		},
		{
			msg:    "Hints without host should return nothing",
			config: customCfg,
			event: bus.Event{
				"hints": common.MapStr{
					"metrics": common.MapStr{
						"module": "prometheus",
					},
				},
			},
			len:    0,
			result: []common.MapStr{},
		},
		{
			msg:    "Hints with logs.disable should return nothing",
			config: customCfg,
			event: bus.Event{
				"hints": common.MapStr{
					"logs": common.MapStr{
						"disable": "true",
					},
				},
			},
			len:    0,
			result: []common.MapStr{},
		},
		{
			msg:    "Empty event hints should return default config",
			config: customCfg,
			event: bus.Event{
				"host": "1.2.3.4",
				"kubernetes": common.MapStr{
					"container": common.MapStr{
						"name": "foobar",
						"id":   "abc",
					},
				},
				"container": common.MapStr{
					"name": "foobar",
					"id":   "abc",
				},
			},
			len: 1,
			result: []common.MapStr{
				{
					"type": "docker",
					"containers": map[string]interface{}{
						"ids": []interface{}{"abc"},
					},
					"close_timeout": "true",
				},
			},
		},
		{
			msg:    "Hint with include|exclude_lines must be part of the input config",
			config: customCfg,
			event: bus.Event{
				"host": "1.2.3.4",
				"kubernetes": common.MapStr{
					"container": common.MapStr{
						"name": "foobar",
						"id":   "abc",
					},
				},
				"container": common.MapStr{
					"name": "foobar",
					"id":   "abc",
				},
				"hints": common.MapStr{
					"logs": common.MapStr{
						"include_lines": "^test, ^test1",
						"exclude_lines": "^test2, ^test3",
					},
				},
			},
			len: 1,
			result: []common.MapStr{
				{
					"type": "docker",
					"containers": map[string]interface{}{
						"ids": []interface{}{"abc"},
					},
					"include_lines": []interface{}{"^test", "^test1"},
					"exclude_lines": []interface{}{"^test2", "^test3"},
					"close_timeout": "true",
				},
			},
		},
		{
			msg:    "Hints with  two sets of include|exclude_lines must be part of the input config",
			config: customCfg,
			event: bus.Event{
				"host": "1.2.3.4",
				"kubernetes": common.MapStr{
					"container": common.MapStr{
						"name": "foobar",
						"id":   "abc",
					},
				},
				"container": common.MapStr{
					"name": "foobar",
					"id":   "abc",
				},
				"hints": common.MapStr{
					"logs": common.MapStr{
						"1": common.MapStr{
							"exclude_lines": "^test1, ^test2",
						},
						"2": common.MapStr{
							"include_lines": "^test1, ^test2",
						},
					},
				},
			},
			len: 2,
			result: []common.MapStr{
				{
					"type": "docker",
					"containers": map[string]interface{}{
						"ids": []interface{}{"abc"},
					},
					"exclude_lines": []interface{}{"^test1", "^test2"},
					"close_timeout": "true",
				},
				{
					"type": "docker",
					"containers": map[string]interface{}{
						"ids": []interface{}{"abc"},
					},
					"include_lines": []interface{}{"^test1", "^test2"},
					"close_timeout": "true",
				},
			},
		},
		{
			msg:    "Hint with multiline config must have a multiline in the input config",
			config: customCfg,
			event: bus.Event{
				"host": "1.2.3.4",
				"kubernetes": common.MapStr{
					"container": common.MapStr{
						"name": "foobar",
						"id":   "abc",
					},
				},
				"container": common.MapStr{
					"name": "foobar",
					"id":   "abc",
				},
				"hints": common.MapStr{
					"logs": common.MapStr{
						"multiline": common.MapStr{
							"pattern": "^test",
							"negate":  "true",
						},
					},
				},
			},
			len: 1,
			result: []common.MapStr{
				{
					"type": "docker",
					"containers": map[string]interface{}{
						"ids": []interface{}{"abc"},
					},
					"multiline": map[string]interface{}{
						"pattern": "^test",
						"negate":  "true",
					},
					"close_timeout": "true",
				},
			},
		},
		{
			msg:    "Hint with inputs config as json must be accepted",
			config: customCfg,
			event: bus.Event{
				"host": "1.2.3.4",
				"kubernetes": common.MapStr{
					"container": common.MapStr{
						"name": "foobar",
						"id":   "abc",
					},
				},
				"container": common.MapStr{
					"name": "foobar",
					"id":   "abc",
				},
				"hints": common.MapStr{
					"logs": common.MapStr{
						"raw": "[{\"containers\":{\"ids\":[\"${data.container.id}\"]},\"multiline\":{\"negate\":\"true\",\"pattern\":\"^test\"},\"type\":\"docker\"}]",
					},
				},
			},
			len: 1,
			result: []common.MapStr{
				{
					"type": "docker",
					"containers": map[string]interface{}{
						"ids": []interface{}{"abc"},
					},
					"multiline": map[string]interface{}{
						"pattern": "^test",
						"negate":  "true",
					},
				},
			},
		},
		{
			msg:    "Hint with processors config must have a processors in the input config",
			config: customCfg,
			event: bus.Event{
				"host": "1.2.3.4",
				"kubernetes": common.MapStr{
					"container": common.MapStr{
						"name": "foobar",
						"id":   "abc",
					},
				},
				"container": common.MapStr{
					"name": "foobar",
					"id":   "abc",
				},
				"hints": common.MapStr{
					"logs": common.MapStr{
						"processors": common.MapStr{
							"1": common.MapStr{
								"dissect": common.MapStr{
									"tokenizer": "%{key1} %{key2}",
								},
							},
							"drop_event": common.MapStr{},
						},
					},
				},
			},
			len: 1,
			result: []common.MapStr{
				{
					"type": "docker",
					"containers": map[string]interface{}{
						"ids": []interface{}{"abc"},
					},
					"close_timeout": "true",
					"processors": []interface{}{
						map[string]interface{}{
							"dissect": map[string]interface{}{
								"tokenizer": "%{key1} %{key2}",
							},
						},
						map[string]interface{}{
							"drop_event": nil,
						},
					},
				},
			},
		},
		{
			msg:    "Hint with module should attach input to its filesets",
			config: customCfg,
			event: bus.Event{
				"host": "1.2.3.4",
				"kubernetes": common.MapStr{
					"container": common.MapStr{
						"name": "foobar",
						"id":   "abc",
					},
				},
				"container": common.MapStr{
					"name": "foobar",
					"id":   "abc",
				},
				"hints": common.MapStr{
					"logs": common.MapStr{
						"module": "apache",
					},
				},
			},
			len: 1,
			result: []common.MapStr{
				{
					"module": "apache",
					"error": map[string]interface{}{
						"enabled": true,
						"input": map[string]interface{}{
							"type": "docker",
							"containers": map[string]interface{}{
								"stream": "all",
								"ids":    []interface{}{"abc"},
							},
							"close_timeout": "true",
						},
					},
					"access": map[string]interface{}{
						"enabled": true,
						"input": map[string]interface{}{
							"type": "docker",
							"containers": map[string]interface{}{
								"stream": "all",
								"ids":    []interface{}{"abc"},
							},
							"close_timeout": "true",
						},
					},
				},
			},
		},
		{
			msg:    "Hint with module should honor defined filesets",
			config: customCfg,
			event: bus.Event{
				"host": "1.2.3.4",
				"kubernetes": common.MapStr{
					"container": common.MapStr{
						"name": "foobar",
						"id":   "abc",
					},
				},
				"container": common.MapStr{
					"name": "foobar",
					"id":   "abc",
				},
				"hints": common.MapStr{
					"logs": common.MapStr{
						"module":  "apache",
						"fileset": "access",
					},
				},
			},
			len: 1,
			result: []common.MapStr{
				{
					"module": "apache",
					"access": map[string]interface{}{
						"enabled": true,
						"input": map[string]interface{}{
							"type": "docker",
							"containers": map[string]interface{}{
								"stream": "all",
								"ids":    []interface{}{"abc"},
							},
							"close_timeout": "true",
						},
					},
					"error": map[string]interface{}{
						"enabled": false,
						"input": map[string]interface{}{
							"type": "docker",
							"containers": map[string]interface{}{
								"stream": "all",
								"ids":    []interface{}{"abc"},
							},
							"close_timeout": "true",
						},
					},
				},
			},
		},
		{
			msg:    "Hint with module should honor defined filesets with streams",
			config: customCfg,
			event: bus.Event{
				"host": "1.2.3.4",
				"kubernetes": common.MapStr{
					"container": common.MapStr{
						"name": "foobar",
						"id":   "abc",
					},
				},
				"container": common.MapStr{
					"name": "foobar",
					"id":   "abc",
				},
				"hints": common.MapStr{
					"logs": common.MapStr{
						"module":         "apache",
						"fileset.stdout": "access",
						"fileset.stderr": "error",
					},
				},
			},
			len: 1,
			result: []common.MapStr{
				{
					"module": "apache",
					"access": map[string]interface{}{
						"enabled": true,
						"input": map[string]interface{}{
							"type": "docker",
							"containers": map[string]interface{}{
								"stream": "stdout",
								"ids":    []interface{}{"abc"},
							},
							"close_timeout": "true",
						},
					},
					"error": map[string]interface{}{
						"enabled": true,
						"input": map[string]interface{}{
							"type": "docker",
							"containers": map[string]interface{}{
								"stream": "stderr",
								"ids":    []interface{}{"abc"},
							},
							"close_timeout": "true",
						},
					},
				},
			},
		},
		{
			msg:    "Hint with module should attach input to its filesets",
			config: defaultCfg,
			event: bus.Event{
				"host": "1.2.3.4",
				"kubernetes": common.MapStr{
					"container": common.MapStr{
						"name": "foobar",
						"id":   "abc",
					},
				},
				"container": common.MapStr{
					"name": "foobar",
					"id":   "abc",
				},
				"hints": common.MapStr{
					"logs": common.MapStr{
						"module": "apache",
					},
				},
			},
			len: 1,
			result: []common.MapStr{
				{
					"module": "apache",
					"error": map[string]interface{}{
						"enabled": true,
						"input": map[string]interface{}{
							"type":   "container",
							"stream": "all",
							"paths": []interface{}{
								"/var/lib/docker/containers/abc/*-json.log",
							},
						},
					},
					"access": map[string]interface{}{
						"enabled": true,
						"input": map[string]interface{}{
							"type":   "container",
							"stream": "all",
							"paths": []interface{}{
								"/var/lib/docker/containers/abc/*-json.log",
							},
						},
					},
				},
			},
		},
		{
			msg:    "Hint with module should honor defined filesets",
			config: defaultCfg,
			event: bus.Event{
				"host": "1.2.3.4",
				"kubernetes": common.MapStr{
					"container": common.MapStr{
						"name": "foobar",
						"id":   "abc",
					},
				},
				"container": common.MapStr{
					"name": "foobar",
					"id":   "abc",
				},
				"hints": common.MapStr{
					"logs": common.MapStr{
						"module":  "apache",
						"fileset": "access",
					},
				},
			},
			len: 1,
			result: []common.MapStr{
				{
					"module": "apache",
					"access": map[string]interface{}{
						"enabled": true,
						"input": map[string]interface{}{
							"type":   "container",
							"stream": "all",
							"paths": []interface{}{
								"/var/lib/docker/containers/abc/*-json.log",
							},
						},
					},
					"error": map[string]interface{}{
						"enabled": false,
						"input": map[string]interface{}{
							"type":   "container",
							"stream": "all",
							"paths": []interface{}{
								"/var/lib/docker/containers/abc/*-json.log",
							},
						},
					},
				},
			},
		},
		{
			msg:    "Hint with module should honor defined filesets with streams",
			config: defaultCfg,
			event: bus.Event{
				"host": "1.2.3.4",
				"kubernetes": common.MapStr{
					"container": common.MapStr{
						"name": "foobar",
						"id":   "abc",
					},
				},
				"container": common.MapStr{
					"name": "foobar",
					"id":   "abc",
				},
				"hints": common.MapStr{
					"logs": common.MapStr{
						"module":         "apache",
						"fileset.stdout": "access",
						"fileset.stderr": "error",
					},
				},
			},
			len: 1,
			result: []common.MapStr{
				{
					"module": "apache",
					"access": map[string]interface{}{
						"enabled": true,
						"input": map[string]interface{}{
							"type":   "container",
							"stream": "stdout",
							"paths": []interface{}{
								"/var/lib/docker/containers/abc/*-json.log",
							},
						},
					},
					"error": map[string]interface{}{
						"enabled": true,
						"input": map[string]interface{}{
							"type":   "container",
							"stream": "stderr",
							"paths": []interface{}{
								"/var/lib/docker/containers/abc/*-json.log",
							},
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		// Configure path for modules access
		abs, _ := filepath.Abs("../../..")
		require.NoError(t, paths.InitPaths(&paths.Path{
			Home: abs,
		}))

		l, err := NewLogHints(test.config)
		if err != nil {
			t.Fatal(err)
		}

		cfgs := l.CreateConfig(test.event)
		assert.Equal(t, test.len, len(cfgs), test.msg)
		configs := make([]common.MapStr, 0)
		for _, cfg := range cfgs {
			config := common.MapStr{}
			err := cfg.Unpack(&config)
			ok := assert.Nil(t, err, test.msg)
			if !ok {
				break
			}
			configs = append(configs, config)
		}
		assert.Equal(t, test.result, configs, test.msg)
	}
}

func TestGenerateHintsWithPaths(t *testing.T) {
	tests := []struct {
		msg    string
		event  bus.Event
		path   string
		len    int
		result common.MapStr
	}{
		{
			msg: "Empty event hints should return default config",
			event: bus.Event{
				"host": "1.2.3.4",
				"kubernetes": common.MapStr{
					"container": common.MapStr{
						"name": "foobar",
						"id":   "abc",
					},
					"pod": common.MapStr{
						"name": "pod",
						"uid":  "12345",
					},
				},
				"container": common.MapStr{
					"name": "foobar",
					"id":   "abc",
				},
			},
			path: "/var/lib/docker/containers/${data.container.id}/*-json.log",
			len:  1,
			result: common.MapStr{
				"type": "docker",
				"containers": map[string]interface{}{
					"paths": []interface{}{"/var/lib/docker/containers/abc/*-json.log"},
				},
				"close_timeout": "true",
			},
		},
		{
			msg: "Empty event hints should return default config. Check for data.kubernetes.container.id instead",
			event: bus.Event{
				"host": "1.2.3.4",
				"kubernetes": common.MapStr{
					"container": common.MapStr{
						"name": "foobar",
						"id":   "abc-k8s",
					},
					"pod": common.MapStr{
						"name": "pod",
						"uid":  "12345",
					},
				},
				"container": common.MapStr{
					"name": "foobar",
					"id":   "abc",
				},
			},
			path: "/var/lib/docker/containers/${data.kubernetes.container.id}/*-json.log",
			len:  1,
			result: common.MapStr{
				"type": "docker",
				"containers": map[string]interface{}{
					"paths": []interface{}{"/var/lib/docker/containers/abc-k8s/*-json.log"},
				},
				"close_timeout": "true",
			},
		},
		{
			msg: "Hint with processors config must have a processors in the input config",
			event: bus.Event{
				"host": "1.2.3.4",
				"kubernetes": common.MapStr{
					"container": common.MapStr{
						"name": "foobar",
						"id":   "abc",
					},
					"pod": common.MapStr{
						"name": "pod",
						"uid":  "12345",
					},
				},
				"container": common.MapStr{
					"name": "foobar",
					"id":   "abc",
				},
				"hints": common.MapStr{
					"logs": common.MapStr{
						"processors": common.MapStr{
							"1": common.MapStr{
								"dissect": common.MapStr{
									"tokenizer": "%{key1} %{key2}",
								},
							},
							"drop_event": common.MapStr{},
						},
					},
				},
			},
			len:  1,
			path: "/var/log/pods/${data.kubernetes.pod.uid}/${data.kubernetes.container.name}/*.log",
			result: common.MapStr{
				"type": "docker",
				"containers": map[string]interface{}{
					"paths": []interface{}{"/var/log/pods/12345/foobar/*.log"},
				},
				"close_timeout": "true",
				"processors": []interface{}{
					map[string]interface{}{
						"dissect": map[string]interface{}{
							"tokenizer": "%{key1} %{key2}",
						},
					},
					map[string]interface{}{
						"drop_event": nil,
					},
				},
			},
		},
		{
			msg: "Hint with module should attach input to its filesets",
			event: bus.Event{
				"host": "1.2.3.4",
				"kubernetes": common.MapStr{
					"container": common.MapStr{
						"name": "foobar",
						"id":   "abc",
					},
					"pod": common.MapStr{
						"name": "pod",
						"uid":  "12345",
					},
				},
				"container": common.MapStr{
					"name": "foobar",
					"id":   "abc",
				},
				"hints": common.MapStr{
					"logs": common.MapStr{
						"module": "apache",
					},
				},
			},
			len:  1,
			path: "/var/log/pods/${data.kubernetes.pod.uid}/${data.kubernetes.container.name}/*.log",
			result: common.MapStr{
				"module": "apache",
				"error": map[string]interface{}{
					"enabled": true,
					"input": map[string]interface{}{
						"type": "docker",
						"containers": map[string]interface{}{
							"stream": "all",
							"paths":  []interface{}{"/var/log/pods/12345/foobar/*.log"},
						},
						"close_timeout": "true",
					},
				},
				"access": map[string]interface{}{
					"enabled": true,
					"input": map[string]interface{}{
						"type": "docker",
						"containers": map[string]interface{}{
							"stream": "all",
							"paths":  []interface{}{"/var/log/pods/12345/foobar/*.log"},
						},
						"close_timeout": "true",
					},
				},
			},
		},
		{
			msg: "Hint with module should honor defined filesets",
			event: bus.Event{
				"host": "1.2.3.4",
				"kubernetes": common.MapStr{
					"container": common.MapStr{
						"name": "foobar",
						"id":   "abc",
					},
					"pod": common.MapStr{
						"name": "pod",
						"uid":  "12345",
					},
				},
				"container": common.MapStr{
					"name": "foobar",
					"id":   "abc",
				},
				"hints": common.MapStr{
					"logs": common.MapStr{
						"module":  "apache",
						"fileset": "access",
					},
				},
			},
			len:  1,
			path: "/var/log/pods/${data.kubernetes.pod.uid}/${data.kubernetes.container.name}/*.log",
			result: common.MapStr{
				"module": "apache",
				"access": map[string]interface{}{
					"enabled": true,
					"input": map[string]interface{}{
						"type": "docker",
						"containers": map[string]interface{}{
							"stream": "all",
							"paths":  []interface{}{"/var/log/pods/12345/foobar/*.log"},
						},
						"close_timeout": "true",
					},
				},
				"error": map[string]interface{}{
					"enabled": false,
					"input": map[string]interface{}{
						"type": "docker",
						"containers": map[string]interface{}{
							"stream": "all",
							"paths":  []interface{}{"/var/log/pods/12345/foobar/*.log"},
						},
						"close_timeout": "true",
					},
				},
			},
		},
	}

	for _, test := range tests {
		cfg, _ := common.NewConfigFrom(map[string]interface{}{
			"default_config": map[string]interface{}{
				"type": "docker",
				"containers": map[string]interface{}{
					"paths": []string{
						test.path,
					},
				},
				"close_timeout": "true",
			},
		})

		// Configure path for modules access
		abs, _ := filepath.Abs("../../..")
		require.NoError(t, paths.InitPaths(&paths.Path{
			Home: abs,
		}))

		l, err := NewLogHints(cfg)
		if err != nil {
			t.Fatal(err)
		}

		cfgs := l.CreateConfig(test.event)
		require.Equal(t, test.len, len(cfgs), test.msg)
		if test.len != 0 {
			config := common.MapStr{}
			err := cfgs[0].Unpack(&config)
			assert.Nil(t, err, test.msg)

			assert.Equal(t, test.result, config, test.msg)
		}

	}
}
