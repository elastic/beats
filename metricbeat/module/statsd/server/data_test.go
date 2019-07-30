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

package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseMetrics(t *testing.T) {
	for _, test := range []struct {
		input    string
		err      error
		expected []statsdMetric
	}{
		{
			input: "gauge1:1.0|g",
			expected: []statsdMetric{{
				name:       "gauge1",
				metricType: "g",
				value:      "1.0",
			}},
		},
		{
			input: "counter1:1.1|c",
			expected: []statsdMetric{{
				name:       "counter1",
				metricType: "c",
				value:      "1.1",
			}},
		},
		{
			input: "counter2:1.15|c|@0.1",
			expected: []statsdMetric{{
				name:       "counter2",
				metricType: "c",
				value:      "1.15",
				sampleRate: "0.1",
			}},
		},
		{
			input: "timer1:1.2|ms",
			expected: []statsdMetric{{
				name:       "timer1",
				metricType: "ms",
				value:      "1.2",
			}},
		},
		{
			input: "histogram1:1.3|h",
			expected: []statsdMetric{{
				name:       "histogram1",
				metricType: "h",
				value:      "1.3",
			}},
		},
		{
			input: "meter1:1.4|m",
			expected: []statsdMetric{{
				name:       "meter1",
				metricType: "m",
				value:      "1.4",
			}},
		},
		{
			input: "lf-ended-meter1:1.5|m\n",
			expected: []statsdMetric{{
				name:       "lf-ended-meter1",
				metricType: "m",
				value:      "1.5",
			}},
		},
		{
			input: "meter1:1.6|m|k1=v1;k2=v2",
			expected: []statsdMetric{{
				name:       "meter1",
				metricType: "m",
				value:      "1.6",
				tags:       "k1=v1;k2=v2",
			}},
		},
		{
			input: "meter1:1.7|m|@0.01|k1=v1;k2=v2",
			expected: []statsdMetric{{
				name:       "meter1",
				metricType: "m",
				value:      "1.7",
				sampleRate: "0.01",
				tags:       "k1=v1;k2=v2",
			}},
		},
		{
			input: "counter2.1:2.1|c|@0.01\ncounter2.2:2.2|c|@0.01",
			expected: []statsdMetric{
				{
					name:       "counter2.1",
					metricType: "c",
					value:      "2.1",
					sampleRate: "0.01",
				},
				{
					name:       "counter2.2",
					metricType: "c",
					value:      "2.2",
					sampleRate: "0.01",
				},
			},
		},
		/// errors
		{
			input:    "meter1-1.4|m",
			expected: []statsdMetric{},
			err:      errInvalidPacket,
		},
		{
			input:    "meter1:1.4-m",
			expected: []statsdMetric{},
			err:      errInvalidPacket,
		},
	} {
		actual, err := Parse([]byte(test.input))
		assert.Equal(t, test.err, err, test.input)
		assert.Equal(t, test.expected, actual, test.input)
	}
}
