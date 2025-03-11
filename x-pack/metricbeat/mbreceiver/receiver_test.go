// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package mbreceiver

import (
	"testing"

	"github.com/elastic/beats/v7/libbeat/otelbeat/oteltest"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestNewReceiver(t *testing.T) {
	config := Config{
		Beatconfig: map[string]interface{}{
			"metricbeat": map[string]interface{}{
				"modules": []map[string]interface{}{
					{
						"module":     "system",
						"enabled":    true,
						"period":     "1s",
						"processes":  []string{".*"},
						"metricsets": []string{"cpu"},
					},
				},
			},
			"output": map[string]interface{}{
				"otelconsumer": map[string]interface{}{},
			},
			"logging": map[string]interface{}{
				"level": "debug",
				"selectors": []string{
					"*",
				},
			},
			"path.home": t.TempDir(),
		},
	}

	oteltest.CheckReceivers(oteltest.CheckReceiversParams{
		T:       t,
		Factory: NewFactory(),
		Receivers: []oteltest.ReceiverConfig{
			{
				Name:   "r1",
				Config: &config,
			},
		},
		AssertFunc: func(t *testing.T, logs map[string][]mapstr.M, zapLogs []byte) bool {
			_ = zapLogs
			return len(logs["r1"]) > 0
		},
	})
}

func TestMultipleReceivers(t *testing.T) {
	config := Config{
		Beatconfig: map[string]interface{}{
			"metricbeat": map[string]interface{}{
				"modules": []map[string]interface{}{
					{
						"module":     "system",
						"enabled":    true,
						"period":     "1s",
						"processes":  []string{".*"},
						"metricsets": []string{"cpu"},
					},
				},
			},
			"output": map[string]interface{}{
				"otelconsumer": map[string]interface{}{},
			},
			"logging": map[string]interface{}{
				"level": "debug",
				"selectors": []string{
					"*",
				},
			},
			"path.home": t.TempDir(),
		},
	}

	oteltest.CheckReceivers(oteltest.CheckReceiversParams{
		T:       t,
		Factory: NewFactory(),
		Receivers: []oteltest.ReceiverConfig{
			{
				Name:   "r1",
				Config: &config,
			},
			{
				Name:   "r2",
				Config: &config,
			},
		},
		AssertFunc: func(t *testing.T, logs map[string][]mapstr.M, zapLogs []byte) bool {
			_ = zapLogs
			return len(logs["r1"]) > 0 && len(logs["r2"]) > 0
		},
	})
}
