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

// +build !integration

package fileset

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/outputs/elasticsearch"
)

func TestLoadPipelinesWithMultiPipelineFileset(t *testing.T) {
	cases := []struct {
		name          string
		esVersion     string
		isErrExpected bool
	}{
		{
			name:          "ES < 6.5.0",
			esVersion:     "6.4.1",
			isErrExpected: true,
		},
		{
			name:          "ES == 6.5.0",
			esVersion:     "6.5.0",
			isErrExpected: false,
		},
		{
			name:          "ES > 6.5.0",
			esVersion:     "6.6.0",
			isErrExpected: false,
		},
	}

	for _, test := range cases {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			testFilesetManifest := &manifest{
				Requires: struct {
					Processors []ProcessorRequirement `config:"processors"`
				}{
					Processors: []ProcessorRequirement{},
				},
				IngestPipeline: []string{"pipeline-plain.json", "pipeline-json.json"},
			}
			testFileset := &Fileset{
				name:       "fls",
				modulePath: "./test/mod",
				manifest:   testFilesetManifest,
				vars: map[string]interface{}{
					"builtin": map[string]interface{}{},
				},
				pipelineIDs: []string{"filebeat-7.0.0-mod-fls-pipeline-plain", "filebeat-7.0.0-mod-fls-pipeline-json"},
			}
			testRegistry := ModuleRegistry{
				registry: map[string]map[string]*Fileset{
					"mod": map[string]*Fileset{
						"fls": testFileset,
					},
				},
			}

			testESServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte("{\"version\":{\"number\":\"" + test.esVersion + "\"}}"))
			}))
			defer testESServer.Close()

			testESClient, err := elasticsearch.NewClient(elasticsearch.ClientSettings{
				URL: testESServer.URL,
			}, nil)
			assert.NoError(t, err)

			err = testESClient.Connect()
			assert.NoError(t, err)

			err = testRegistry.LoadPipelines(testESClient, false)
			if test.isErrExpected {
				assert.IsType(t, MultiplePipelineUnsupportedError{}, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSetEcsProcessors(t *testing.T) {
	cases := []struct {
		name          string
		esVersion     *common.Version
		content       map[string]interface{}
		expected      map[string]interface{}
		isErrExpected bool
	}{
		{
			name:      "ES < 6.7.0",
			esVersion: common.MustNewVersion("6.6.0"),
			content: map[string]interface{}{
				"processors": []interface{}{
					map[string]interface{}{
						"user_agent": map[string]interface{}{
							"field": "foo.http_user_agent",
						},
					},
				}},
			isErrExpected: true,
		},
		{
			name:      "ES == 6.7.0",
			esVersion: common.MustNewVersion("6.7.0"),
			content: map[string]interface{}{
				"processors": []interface{}{
					map[string]interface{}{
						"rename": map[string]interface{}{
							"field":        "foo.src_ip",
							"target_field": "source.ip",
						},
					},
					map[string]interface{}{
						"user_agent": map[string]interface{}{
							"field": "foo.http_user_agent",
						},
					},
				},
			},
			expected: map[string]interface{}{
				"processors": []interface{}{
					map[string]interface{}{
						"rename": map[string]interface{}{
							"field":        "foo.src_ip",
							"target_field": "source.ip",
						},
					},
					map[string]interface{}{
						"user_agent": map[string]interface{}{
							"field": "foo.http_user_agent",
							"ecs":   true,
						},
					},
				},
			},
			isErrExpected: false,
		},
		{
			name:      "ES >= 7.0.0",
			esVersion: common.MustNewVersion("7.0.0"),
			content: map[string]interface{}{
				"processors": []interface{}{
					map[string]interface{}{
						"rename": map[string]interface{}{
							"field":        "foo.src_ip",
							"target_field": "source.ip",
						},
					},
					map[string]interface{}{
						"user_agent": map[string]interface{}{
							"field": "foo.http_user_agent",
						},
					},
				},
			},
			expected: map[string]interface{}{
				"processors": []interface{}{
					map[string]interface{}{
						"rename": map[string]interface{}{
							"field":        "foo.src_ip",
							"target_field": "source.ip",
						},
					},
					map[string]interface{}{
						"user_agent": map[string]interface{}{
							"field": "foo.http_user_agent",
						},
					},
				},
			},
			isErrExpected: false,
		},
	}

	for _, test := range cases {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			err := setECSProcessors(*test.esVersion, "foo-pipeline", test.content)
			if test.isErrExpected {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, test.expected, test.content)
			}
		})
	}
}
