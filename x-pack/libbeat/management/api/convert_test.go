// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package api

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConvertAPI(t *testing.T) {
	tests := map[string]struct {
		t        string
		config   map[string]interface{}
		expected map[string]interface{}
		err      bool
	}{
		"output": {
			t: "output",
			config: map[string]interface{}{
				"_sub_type": "elasticsearch",
				"username":  "foobar",
			},
			expected: map[string]interface{}{
				"elasticsearch": map[string]interface{}{
					"username": "foobar",
				},
			},
		},
		"filebeat inputs": {
			t: "filebeat.inputs",
			config: map[string]interface{}{
				"type": "log",
				"paths": []string{
					"/var/log/message.log",
					"/var/log/system.log",
				},
			},
			expected: map[string]interface{}{
				"type": "log",
				"paths": []string{
					"/var/log/message.log",
					"/var/log/system.log",
				},
			},
		},
		"filebeat modules": {
			t: "filebeat.modules",
			config: map[string]interface{}{
				"_sub_type": "system",
			},
			expected: map[string]interface{}{
				"module": "system",
			},
		},
		"metricbeat modules": {
			t: "metricbeat.modules",
			config: map[string]interface{}{
				"_sub_type": "logstash",
			},
			expected: map[string]interface{}{
				"module": "logstash",
			},
		},
		"badly formed output": {
			err: true,
			t:   "output",
			config: map[string]interface{}{
				"nosubtype": "logstash",
			},
		},
		"badly formed filebeat module": {
			err: true,
			t:   "filebeat.modules",
			config: map[string]interface{}{
				"nosubtype": "logstash",
			},
		},
		"badly formed metricbeat module": {
			err: true,
			t:   "metricbeat.modules",
			config: map[string]interface{}{
				"nosubtype": "logstash",
			},
		},
		"unknown type is passthrough": {
			t: "unkown",
			config: map[string]interface{}{
				"nosubtype": "logstash",
			},
			expected: map[string]interface{}{
				"nosubtype": "logstash",
			},
		},
	}

	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			converter := selectConverter(test.t)
			newMap, err := converter(test.config)
			if !assert.Equal(t, test.err, err != nil) {
				return
			}
			assert.True(t, reflect.DeepEqual(newMap, test.expected))
		})
	}
}
