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
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/elastic-agent-autodiscover/bus"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestGenerateHints(t *testing.T) {
	tests := []struct {
		message string
		event   bus.Event
		len     int
		result  mapstr.M
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
			result: mapstr.M{},
		},
		{
			message: "Hints without host should return nothing",
			event: bus.Event{
				"hints": mapstr.M{
					"monitor": mapstr.M{
						"type": "http",
					},
				},
			},
			len:    0,
			result: mapstr.M{},
		},
		{
			message: "Hints without port should return nothing if ${data.port} is used",
			event: bus.Event{
				"host": "1.2.3.4",
				"hints": mapstr.M{
					"monitor": mapstr.M{
						"type":  "http",
						"hosts": "${data.host}:${data.port},test:${data.port}",
					},
				},
			},
			len:    0,
			result: mapstr.M{},
		},
		{
			message: "Hints with multiple hosts returns all with the template",
			event: bus.Event{
				"host": "1.2.3.4",
				"port": 9090,
				"hints": mapstr.M{
					"monitor": mapstr.M{
						"type":  "http",
						"hosts": "${data.host}:8888,${data.host}:${data.port}",
					},
				},
			},
			len: 1,
			result: mapstr.M{
				"type":     "http",
				"schedule": "@every 5s",
				"hosts":    []string{"1.2.3.4:8888", "1.2.3.4:9090"},
			},
		},
		{
			message: "Monitor defined in monitors as a JSON string should return a config",
			event: bus.Event{
				"host": "1.2.3.4",
				"hints": mapstr.M{
					"monitor": mapstr.M{
						"raw": "{\"enabled\":true,\"type\":\"http\",\"schedule\":\"@every 20s\",\"timeout\":\"3s\"}",
					},
				},
			},
			len: 1,
			result: mapstr.M{
				"type":     "http",
				"timeout":  "3s",
				"schedule": "@every 20s",
				"enabled":  true,
			},
		},
		{
			message: "Monitor with processor config must return an module having the processor defined",
			event: bus.Event{
				"host": "1.2.3.4",
				"port": 9090,
				"hints": mapstr.M{
					"monitor": mapstr.M{
						"type":  "http",
						"hosts": "${data.host}:9090",
						"processors": mapstr.M{
							"add_locale": mapstr.M{
								"abbrevation": "MST",
							},
						},
					},
				},
			},
			len: 1,
			result: mapstr.M{
				"type":     "http",
				"hosts":    []string{"1.2.3.4:9090"},
				"schedule": "@every 5s",
				"processors": []interface{}{
					map[string]interface{}{
						"add_locale": map[string]interface{}{
							"abbrevation": "MST",
						},
					},
				},
			},
		},
		{
			message: "Hints with multiple monitors should return multiple",
			event: bus.Event{
				"host": "1.2.3.4",
				"port": 9090,
				"hints": mapstr.M{
					"monitor": mapstr.M{
						"1": mapstr.M{
							"type":  "http",
							"hosts": "${data.host}:8888,${data.host}:9090",
						},
						"2": mapstr.M{
							"type":  "http",
							"hosts": "${data.host}:8888,${data.host}:9090",
						},
					},
				},
			},
			len: 2,
			result: mapstr.M{
				"type":     "http",
				"schedule": "@every 5s",
				"hosts":    []string{"1.2.3.4:8888", "1.2.3.4:9090"},
			},
		},
		{
			message: "Hints for ICMP with port should return nothing",
			event: bus.Event{
				"host": "1.2.3.4",
				"port": 9090,
				"hints": mapstr.M{
					"monitor": mapstr.M{
						"1": mapstr.M{
							"type":  "icmp",
							"hosts": "${data.host}:9090",
						},
						"2": mapstr.M{
							"type":  "icmp",
							"hosts": "${data.host}:${data.port}",
						},
					},
				},
			},
			len:    0,
			result: mapstr.M{},
		},
	}
	for _, test := range tests {

		m := heartbeatHints{
			config: defaultConfig(),
			logger: logp.L(),
		}
		cfgs := m.CreateConfig(test.event)
		assert.Equal(t, test.len, len(cfgs), test.message)

		if len(cfgs) != 0 {
			config := mapstr.M{}
			err := cfgs[0].Unpack(&config)
			assert.Nil(t, err, test.message)

			// Autodiscover can return configs with different sort orders here, which is irrelevant
			// To make tests pass consistently we sort the host list
			hostStrs := []string{}
			if hostsSlice, ok := config["hosts"].([]interface{}); ok && len(hostsSlice) > 0 {
				for _, hi := range hostsSlice {
					hostStrs = append(hostStrs, hi.(string))
				}
				sort.Strings(hostStrs)
				config["hosts"] = hostStrs
			}

			assert.Equal(t, test.result, config, test.message)
		}

	}
}
