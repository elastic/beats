// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package server

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/helper/server"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
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
			input: "counter1:11|c",
			expected: []statsdMetric{{
				name:       "counter1",
				metricType: "c",
				value:      "11",
			}},
		},
		{
			input: "counter2:15|c|@0.1",
			expected: []statsdMetric{{
				name:       "counter2",
				metricType: "c",
				value:      "15",
				sampleRate: "0.1",
			}},
		},
		{
			input: "decrement-counter:-15|c",
			expected: []statsdMetric{{
				name:       "decrement-counter",
				metricType: "c",
				value:      "-15",
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
			input: "histogram1:3|h",
			expected: []statsdMetric{{
				name:       "histogram1",
				metricType: "h",
				value:      "3",
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
			input: "counter2.1:1|c|@0.01\ncounter2.2:2|c|@0.01",
			expected: []statsdMetric{
				{
					name:       "counter2.1",
					metricType: "c",
					value:      "1",
					sampleRate: "0.01",
				},
				{
					name:       "counter2.2",
					metricType: "c",
					value:      "2",
					sampleRate: "0.01",
				},
			},
		},
		/// tags
		{
			input: "tags1:1|c|#k1:v1,k2:v2",
			expected: []statsdMetric{
				{
					name:       "tags1",
					metricType: "c",
					value:      "1",
					tags: map[string]string{
						"k1": "v1",
						"k2": "v2",
					},
				},
			},
		},
		{
			input: "tags2:2|m|@0.1|#k1:v1,k2:v2",
			expected: []statsdMetric{
				{
					name:       "tags2",
					metricType: "m",
					value:      "2",
					sampleRate: "0.1",
					tags: map[string]string{
						"k1": "v1",
						"k2": "v2",
					},
				},
			},
		},
		{ // Influx Statsd tags
			input: "tags2,k1=v1,k2=v2:1|c",
			expected: []statsdMetric{
				{
					name:       "tags2",
					metricType: "c",
					value:      "1",
					tags: map[string]string{
						"k1": "v1",
						"k2": "v2",
					},
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
		actual, err := parse([]byte(test.input))
		assert.Equal(t, test.err, err, test.input)
		assert.Equal(t, test.expected, actual, test.input)

		processor := newMetricProcessor(1000, time.Second)
		for _, e := range actual {
			err := processor.processSingle(e)

			assert.NoError(t, err)

		}
	}
}

type testUDPEvent struct {
	event common.MapStr
	meta  server.Meta
}

func (u *testUDPEvent) GetEvent() common.MapStr {
	return u.event
}

func (u *testUDPEvent) GetMeta() server.Meta {
	return u.meta
}

func process(t *testing.T, packets []string, ms *MetricSet) {
	for _, d := range packets {
		udpEvent := &testUDPEvent{
			event: common.MapStr{
				server.EventDataKey: []byte(d),
			},
			meta: server.Meta{
				"client_ip": "127.0.0.1",
			},
		}
		err := ms.processor.Process(udpEvent)
		require.NoError(t, err)
	}
}

func TestTagsGrouping(t *testing.T) {
	ms := mbtest.NewMetricSet(t, map[string]interface{}{"module": "statsd"}).(*MetricSet)
	testData := []string{
		"metric1:1.0|g|#k1:v1,k2:v2",
		"metric2:2|c|#k1:v1,k2:v2",

		"metric3:3|c|@0.1|#k1:v2,k2:v3",
		"metric4:4|ms|#k1:v2,k2:v3",
	}

	process(t, testData, ms)
	events := ms.getEvents()
	assert.Len(t, events, 2)

	actualTags := []common.MapStr{}
	for _, e := range events {
		actualTags = append(actualTags, e.RootFields)
	}

	expectedTags := []common.MapStr{
		common.MapStr{
			"labels": common.MapStr{
				"k1": "v1",
				"k2": "v2",
			},
		},
		common.MapStr{
			"labels": common.MapStr{
				"k1": "v2",
				"k2": "v3",
			},
		},
	}

	assert.ElementsMatch(t, expectedTags, actualTags)
}

func TestTagsCleanup(t *testing.T) {
	ms := mbtest.NewMetricSet(t, map[string]interface{}{"module": "statsd", "ttl": "1s"}).(*MetricSet)
	testData := []string{
		"metric1:1.0|g|#k1:v1,k2:v2",

		"metric2:3|c|@0.1|#k1:v2,k2:v3",
	}
	process(t, testData, ms)
	time.Sleep(1000 * time.Millisecond)

	// they will be reported at least once
	assert.Len(t, ms.getEvents(), 2)

	testData = []string{
		"metric1:2.0|g|#k1:v1,k2:v2",
	}
	// refresh metrics1
	process(t, testData, ms)
	time.Sleep(500 * time.Millisecond)

	// metrics2 should be out now
	events := ms.getEvents()
	assert.Len(t, events, 1)

	assert.Equal(t, events[0].MetricSetFields, common.MapStr{"metric1": map[string]interface{}{"value": 2.0}})
}

func TestData(t *testing.T) {
	ms := mbtest.NewMetricSet(t, map[string]interface{}{"module": "statsd"}).(*MetricSet)
	testData := []string{
		"metric1:1.0|g|#k1:v1,k2:v2",
		"metric2:2|c|#k1:v1,k2:v2",
		"metric3:3|c|@0.1|#k1:v1,k2:v2",
		"metric4:4|ms|#k1:v1,k2:v2",
		"metric5:5|h|#k1:v1,k2:v2",
		"metric6:6|h|#k1:v1,k2:v2",
		"metric7:7|ms|#k1:v1,k2:v2",
		"metric8,k1=v1,k2=v2:8|h",
		"metric9.with.dots,k1=v1,k2=v2:9|h",
	}
	process(t, testData, ms)

	events := ms.getEvents()
	assert.Len(t, events, 1)

	mbevent := mbtest.StandardizeEvent(ms, *events[0])
	mbtest.WriteEventToDataJSON(t, mbevent, "")
}
