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

package loader

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/logp"
)

func TestExternalConfigLoading(t *testing.T) {
	authCfg := map[string]interface{}{
		"data_stream": map[string]interface{}{
			"dataset": "system.auth",
			"type":    "logs",
		},
		"exclude_files": []interface{}{".gz$"},
		"id":            "logfile-system.auth-my-id",
		"paths":         []interface{}{"/var/log/auth.log*", "/var/log/secure*"},
		"use_output":    "default",
	}
	syslogCfg := map[string]interface{}{
		"data_stream": map[string]interface{}{
			"dataset": "system.syslog",
			"type":    "logs",
		},
		"type":          "logfile",
		"id":            "logfile-system.syslog-my-id",
		"exclude_files": []interface{}{".gz$"},
		"paths":         []interface{}{"/var/log/messages*", "/var/log/syslog*"},
		"use_output":    "default",
	}

	diskioCfg := map[string]interface{}{
		"data_stream": map[string]interface{}{
			"dataset": "system.diskio",
			"type":    "metrics",
		},
		"id":         "system/metrics-system.diskio-my-id",
		"metricsets": []interface{}{"diskio"},
		"period":     "10s",
	}
	filesystemCfg := map[string]interface{}{
		"data_stream": map[string]interface{}{
			"dataset": "system.filesystem",
			"type":    "metrics",
		},
		"id":         "system/metrics-system.filesystem-my-id",
		"metricsets": []interface{}{"filesystem"},
		"period":     "30s",
	}

	outputCfg := map[string]interface{}{
		"default": map[string]interface{}{
			"type":    "elasticsearch",
			"hosts":   []interface{}{"127.0.0.1:9201"},
			"api-key": "my-secret-key",
		},
	}

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
				"outputs": outputCfg,
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
					authCfg,
					syslogCfg,
					diskioCfg,
					filesystemCfg,
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
				"outputs": outputCfg,
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
					authCfg,
					syslogCfg,
					diskioCfg,
					filesystemCfg,
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
	log := logp.L()
	return NewLoader(log, inputsFolder)
}
