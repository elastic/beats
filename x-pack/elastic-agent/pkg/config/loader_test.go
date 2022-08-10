// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package config

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
)

func TestExternalConfigLoading(t *testing.T) {
	cases := map[string]struct {
		configs        []string
		inputsFolder   string
		expectedConfig map[string]interface{}
		err            bool
	}{
		"non-existent config files lead to error": {
			configs: []string{"no-such-configuration-file.yml"},
			err:     true,
		},
		"invalid configuration file in inputs folder lead to error": {
			configs: []string{
				filepath.Join("testdata", "inputs", "invalid-inputs.yml"),
			},
			inputsFolder: filepath.Join("testdata", "inputs", "*.yml"),
			err:          true,
		},
		"two standalone configs can be merged without inputs": {
			configs: []string{
				filepath.Join("testdata", "standalone1.yml"),
				filepath.Join("testdata", "standalone2.yml"),
			},
			inputsFolder: "",
			expectedConfig: map[string]interface{}{
				"outputs": map[string]interface{}{
					"default": map[string]interface{}{
						"type":    "elasticsearch",
						"hosts":   []interface{}{"127.0.0.1:9201"},
						"api-key": "my-secret-key",
					},
				},
				"agent": map[string]interface{}{
					"logging": map[string]interface{}{
						"level": "debug",
						"metrics": map[string]interface{}{
							"enabled": false,
						},
					},
				},
			},
		},
		"one external config, standalone config without inputs section": {
			configs: []string{
				filepath.Join("testdata", "standalone1.yml"),
				filepath.Join("testdata", "inputs", "log-inputs.yml"),
				filepath.Join("testdata", "inputs", "metrics-inputs.yml"),
			},
			inputsFolder: filepath.Join("testdata", "inputs", "*.yml"),
			expectedConfig: map[string]interface{}{
				"outputs": map[string]interface{}{
					"default": map[string]interface{}{
						"type":    "elasticsearch",
						"hosts":   []interface{}{"127.0.0.1:9201"},
						"api-key": "my-secret-key",
					},
				},
				"inputs": []interface{}{
					map[string]interface{}{
						"data_stream": map[string]interface{}{
							"dataset": "system.auth",
							"type":    "logs",
						},
						"exclude_files": []interface{}{".gz$"},
						"id":            "logfile-system.auth-my-id",
						"paths":         []interface{}{"/var/log/auth.log*", "/var/log/secure*"},
						"use_output":    "default",
					},
					map[string]interface{}{
						"data_stream": map[string]interface{}{
							"dataset": "system.syslog",
							"type":    "logs",
						},
						"type":          "logfile",
						"id":            "logfile-system.syslog-my-id",
						"exclude_files": []interface{}{".gz$"},
						"paths":         []interface{}{"/var/log/messages*", "/var/log/syslog*"},
						"use_output":    "default",
					},
					map[string]interface{}{
						"data_stream": map[string]interface{}{
							"dataset": "system.diskio",
							"type":    "metrics",
						},
						"id":         "system/metrics-system.diskio-my-id",
						"metricsets": []interface{}{"diskio"},
						"period":     "10s",
					},
					map[string]interface{}{
						"data_stream": map[string]interface{}{
							"dataset": "system.filesystem",
							"type":    "metrics",
						},
						"id":         "system/metrics-system.filesystem-my-id",
						"metricsets": []interface{}{"filesystem"},
						"period":     "30s",
					},
				},
			},
		},
		"inputs sections of all external and standalone configuration are merged to the result": {
			configs: []string{
				filepath.Join("testdata", "standalone-with-inputs.yml"),
				filepath.Join("testdata", "inputs", "log-inputs.yml"),
				filepath.Join("testdata", "inputs", "metrics-inputs.yml"),
			},
			inputsFolder: filepath.Join("testdata", "inputs", "*.yml"),
			expectedConfig: map[string]interface{}{
				"outputs": map[string]interface{}{
					"default": map[string]interface{}{
						"type":    "elasticsearch",
						"hosts":   []interface{}{"127.0.0.1:9201"},
						"api-key": "my-secret-key",
					},
				},
				"inputs": []interface{}{
					map[string]interface{}{
						"type":                  "system/metrics",
						"data_stream.namespace": "default",
						"use_output":            "default",
						"streams": []interface{}{
							map[string]interface{}{
								"metricset":           "cpu",
								"data_stream.dataset": "system.cpu",
							},
						},
					},
					map[string]interface{}{
						"data_stream": map[string]interface{}{
							"dataset": "system.auth",
							"type":    "logs",
						},
						"exclude_files": []interface{}{".gz$"},
						"id":            "logfile-system.auth-my-id",
						"paths":         []interface{}{"/var/log/auth.log*", "/var/log/secure*"},
						"use_output":    "default",
					},
					map[string]interface{}{
						"data_stream": map[string]interface{}{
							"dataset": "system.syslog",
							"type":    "logs",
						},
						"type":          "logfile",
						"id":            "logfile-system.syslog-my-id",
						"exclude_files": []interface{}{".gz$"},
						"paths":         []interface{}{"/var/log/messages*", "/var/log/syslog*"},
						"use_output":    "default",
					},
					map[string]interface{}{
						"data_stream": map[string]interface{}{
							"dataset": "system.diskio",
							"type":    "metrics",
						},
						"id":         "system/metrics-system.diskio-my-id",
						"metricsets": []interface{}{"diskio"},
						"period":     "10s",
					},
					map[string]interface{}{
						"data_stream": map[string]interface{}{
							"dataset": "system.filesystem",
							"type":    "metrics",
						},
						"id":         "system/metrics-system.filesystem-my-id",
						"metricsets": []interface{}{"filesystem"},
						"period":     "30s",
					},
				},
			},
		},
	}

	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			test := test

			l := mustNewLoader(test.inputsFolder)
			c, err := l.Load(test.configs)
			if test.err {
				require.NotNil(t, err)
				return
			}

			require.Nil(t, err)
			raw, err := c.ToMapStr()
			require.Nil(t, err)
			require.Equal(t, test.expectedConfig, raw)
		})
	}
}

func mustNewLoader(inputsFolder string) *Loader {
	log, err := logger.New("loader_test", true)
	if err != nil {
		panic(err)
	}
	return NewLoader(log, inputsFolder)
}
