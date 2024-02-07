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

package kafka

import (
	"testing"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/management"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestBuildTopicSelector(t *testing.T) {
	testCases := []struct {
		name       string
		topic      string
		expected   string
		underAgent bool
	}{
		{
			name:       "static topic",
			topic:      "a test",
			expected:   "a test",
			underAgent: true,
		},
		{
			name:       "dynamic topic under agent",
			topic:      "%{[foo]}",
			expected:   "%{[foo]}",
			underAgent: true,
		},
		{
			name:       "dynamic topic standalone",
			topic:      "%{[foo]}",
			expected:   "bar",
			underAgent: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			topicCfg := struct {
				Topic string `config:"topic" yaml:"topic"`
			}{
				Topic: tc.topic,
			}

			configC := config.MustNewConfigFrom(topicCfg)
			if tc.underAgent {
				previous := management.UnderAgent()
				management.SetUnderAgent(true)
				defer management.SetUnderAgent(previous)
			}

			selector, err := buildTopicSelector(configC)
			if err != nil {
				t.Fatalf("could not build topic selector: %s", err)
			}

			event := beat.Event{Fields: mapstr.M{"foo": "bar"}}
			topic, err := selector.Select(&event)
			if err != nil {
				t.Fatalf("could not use selector: %s", err)
			}

			if topic != tc.expected {
				t.Fatalf("expecting topic to be '%s', got '%s' instead", tc.expected, topic)
			}
		})
	}

	t.Run("fail unpacking config", func(t *testing.T) {
		_, err := buildTopicSelector(nil)
		if err == nil {
			t.Error("unpack must fail with a nil *config.C")
		}
	})
}
