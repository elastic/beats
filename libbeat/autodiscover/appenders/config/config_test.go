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

package config

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/bus"
)

func TestGenerateAppender(t *testing.T) {
	tests := []struct {
		name        string
		eventConfig common.MapStr
		event       bus.Event
		result      common.MapStr
		config      string
	}{
		{
			name:  "Appender without a condition should apply the config regardless",
			event: bus.Event{},
			result: common.MapStr{
				"test":  "bar",
				"test1": "foo",
			},
			eventConfig: common.MapStr{
				"test": "bar",
			},
			config: `
config:
  test1: foo`,
		},
		{
			name: "Appender with a condition check that fails",
			event: bus.Event{
				"field": "notbar",
			},
			result: common.MapStr{
				"test": "bar",
			},
			eventConfig: common.MapStr{
				"test": "bar",
			},
			config: `
config: 
  test2: foo 
condition.equals:
  field: bar`,
		},
		{
			name: "Appender with a condition check that passes. It should get appended",
			event: bus.Event{
				"field": "bar",
			},
			result: common.MapStr{
				"test":  "bar",
				"test2": "foo",
			},
			eventConfig: common.MapStr{
				"test": "bar",
			},
			config: `
config: 
  test2: foo 
condition.equals:
  field: bar`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config, err := common.NewConfigWithYAML([]byte(test.config), "")
			if err != nil {
				t.Fatal(err)
			}

			appender, err := NewConfigAppender(config)
			assert.Nil(t, err)
			assert.NotNil(t, appender)

			eveConfig, err := common.NewConfigFrom(&test.eventConfig)
			assert.Nil(t, err)

			test.event["config"] = []*common.Config{eveConfig}
			appender.Append(test.event)

			cfgs, _ := test.event["config"].([]*common.Config)
			assert.Equal(t, len(cfgs), 1)

			out := common.MapStr{}
			cfgs[0].Unpack(&out)

			assert.Equal(t, out, test.result)
		})

	}
}
