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

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/bus"
	"github.com/elastic/beats/libbeat/paths"
)

func TestGenerateHints(t *testing.T) {
	tests := []struct {
		msg    string
		event  bus.Event
		len    int
		result common.MapStr
	}{
		{
			msg: "Hints without host should return nothing",
			event: bus.Event{
				"hints": common.MapStr{
					"metrics": common.MapStr{
						"module": "prometheus",
					},
				},
			},
			len:    0,
			result: common.MapStr{},
		},
		{
			msg: "Empty event hints should return default config",
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
			result: common.MapStr{
				"type": "docker",
				"containers": map[string]interface{}{
					"ids": []interface{}{"abc"},
				},
				"close_timeout": "true",
			},
		},
		{
			msg: "Hint with include|exclude_lines must be part of the input config",
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
			result: common.MapStr{
				"type": "docker",
				"containers": map[string]interface{}{
					"ids": []interface{}{"abc"},
				},
				"include_lines": []interface{}{"^test", "^test1"},
				"exclude_lines": []interface{}{"^test2", "^test3"},
				"close_timeout": "true",
			},
		},
		{
			msg: "Hint with multiline config must have a multiline in the input config",
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
			result: common.MapStr{
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
		{
			msg: "Hint with inputs config as json must be accepted",
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
			result: common.MapStr{
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
		{
			msg: "Hint with processors config must have a processors in the input config",
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
			result: common.MapStr{
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
		{
			msg: "Hint with module should attach input to its filesets",
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
						"module": "apache2",
					},
				},
			},
			len: 1,
			result: common.MapStr{
				"module": "apache2",
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
		{
			msg: "Hint with module should honor defined filesets",
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
						"module":  "apache2",
						"fileset": "access",
					},
				},
			},
			len: 1,
			result: common.MapStr{
				"module": "apache2",
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
		{
			msg: "Hint with module should honor defined filesets with streams",
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
						"module":         "apache2",
						"fileset.stdout": "access",
						"fileset.stderr": "error",
					},
				},
			},
			len: 1,
			result: common.MapStr{
				"module": "apache2",
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
	}

	for _, test := range tests {
		cfg, _ := common.NewConfigFrom(map[string]interface{}{
			"config": map[string]interface{}{
				"type": "docker",
				"containers": map[string]interface{}{
					"ids": []string{
						"${data.container.id}",
					},
				},
				"close_timeout": "true",
			},
		})

		// Configure path for modules access
		abs, _ := filepath.Abs("../../..")
		err := paths.InitPaths(&paths.Path{
			Home: abs,
		})

		l, err := NewLogHints(cfg)
		if err != nil {
			t.Fatal(err)
		}

		cfgs := l.CreateConfig(test.event)
		assert.Equal(t, len(cfgs), test.len, test.msg)
		if test.len != 0 {
			config := common.MapStr{}
			err := cfgs[0].Unpack(&config)
			assert.Nil(t, err, test.msg)

			assert.Equal(t, test.result, config, test.msg)
		}

	}
}
