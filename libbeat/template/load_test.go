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

package template

import (
	"errors"
	"fmt"
	"testing"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileLoader_Load(t *testing.T) {
	ver := "7.0.0"
	prefix := "mock"
	info := beat.Info{Beat: "mock", Version: ver, IndexPrefix: prefix}
	tmplName := fmt.Sprintf("%s-%s", prefix, ver)

	for name, test := range map[string]struct {
		settings TemplateSettings
		body     common.MapStr
		fields   []byte
		want     common.MapStr
		wantErr  error
	}{
		"load minimal config info": {
			body: common.MapStr{
				"index_patterns": []string{"mock-7.0.0"},
				"data_stream":    struct{}{},
				"priority":       150,
				"template": common.MapStr{
					"settings": common.MapStr{"index": nil}},
			},
		},
		"load minimal config with index settings": {
			settings: TemplateSettings{Index: common.MapStr{"code": "best_compression"}},
			body: common.MapStr{
				"index_patterns": []string{"mock-7.0.0"},
				"data_stream":    struct{}{},
				"priority":       150,
				"template": common.MapStr{
					"settings": common.MapStr{"index": common.MapStr{"code": "best_compression"}}},
			},
		},
		"load minimal config with source settings": {
			settings: TemplateSettings{Source: common.MapStr{"enabled": false}},
			body: common.MapStr{
				"index_patterns": []string{"mock-7.0.0"},
				"data_stream":    struct{}{},
				"priority":       150,
				"template": common.MapStr{
					"settings": common.MapStr{"index": nil},
					"mappings": common.MapStr{
						"_source":           common.MapStr{"enabled": false},
						"_meta":             common.MapStr{"beat": prefix, "version": ver},
						"date_detection":    false,
						"dynamic_templates": nil,
						"properties":        nil,
					}},
			},
		},
		"load config and in-line analyzer fields": {
			body: common.MapStr{
				"index_patterns": []string{"mock-7.0.0"},
				"data_stream":    struct{}{},
				"priority":       150,
				"template": common.MapStr{
					"settings": common.MapStr{"index": nil}},
			},
			fields: []byte(`- key: test
  title: Test fields.yml with analyzer
  description: >
    Contains text fields with in-line analyzer for testing
  fields:
    - name: script_block_text
      type: text
      analyzer:
        test_powershell:
          type: pattern
          pattern: "[\\W&&[^-]]+"

    - name: code_block_text
      type: text
      analyzer:
        test_powershell:
          type: pattern
          pattern: "[\\W&&[^-]]+"

    - name: standard_text
      type: text
      analyzer: simple
`),
			want: common.MapStr{
				"index_patterns": []string{"mock-7.0.0"},
				"data_stream":    struct{}{},
				"priority":       150,
				"template": common.MapStr{
					"mappings": common.MapStr{
						"_meta": common.MapStr{
							"version": "7.0.0",
							"beat":    "mock",
						},
						"date_detection": false,
						"dynamic_templates": []common.MapStr{
							{
								"strings_as_keyword": common.MapStr{
									"mapping": common.MapStr{
										"ignore_above": 1024,
										"type":         "keyword",
									},
									"match_mapping_type": "string",
								},
							},
						},
						"properties": common.MapStr{
							"code_block_text": common.MapStr{
								"type":     "text",
								"norms":    false,
								"analyzer": "test_powershell",
							},
							"script_block_text": common.MapStr{
								"type":     "text",
								"norms":    false,
								"analyzer": "test_powershell",
							},
							"standard_text": common.MapStr{
								"type":     "text",
								"norms":    false,
								"analyzer": "simple",
							},
						},
					},
					"settings": common.MapStr{
						"index": common.MapStr{
							"refresh_interval": "5s",
							"mapping": common.MapStr{
								"total_fields": common.MapStr{
									"limit": 10000,
								},
							},
							"query": common.MapStr{
								"default_field": []string{
									"fields.*",
								},
							},
							"max_docvalue_fields_search": 200,
						},
						"analysis": common.MapStr{
							"analyzer": common.MapStr{
								"test_powershell": map[string]interface{}{
									"type":    "pattern",
									"pattern": "[\\W&&[^-]]+",
								},
							},
						},
					},
				},
			},
		},
		"load config and in-line analyzer fields with name collision": {
			body: common.MapStr{
				"index_patterns": []string{"mock-7.0.0"},
				"settings":       common.MapStr{"index": nil},
			},
			fields: []byte(`- key: test
  title: Test fields.yml with analyzer
  description: >
    Contains text fields with in-line analyzer for testing
  fields:
    - name: script_block_text
      type: text
      analyzer:
        test_powershell:
          type: pattern
          pattern: "[\\W&&[^-]]+"

    - name: code_block_text
      type: text
      analyzer:
        test_powershell:
          type: pattern
          pattern: "[\\W&&[^*-]]+"

    - name: standard_text
      type: text
      analyzer: simple
`),
			wantErr: fmt.Errorf(`error creating template: %w`, errors.New(`inconsistent definitions for analyzers with the name "test_powershell"`)),
		},
	} {
		t.Run(name, func(t *testing.T) {
			fc, err := newFileClient(ver)
			require.NoError(t, err)
			fl := NewFileLoader(fc)

			cfg := DefaultConfig(info)
			cfg.Settings = test.settings

			err = fl.Load(cfg, info, test.fields, false)
			require.Equal(t, test.wantErr, err)
			if err != nil {
				return
			}
			assert.Equal(t, "template", fc.component)
			assert.Equal(t, tmplName, fc.name)
			want := test.body
			if test.fields != nil {
				want = test.want
			}
			assert.Equal(t, want.StringToPrint()+"\n", fc.body)
		})
	}
}

type fileClient struct {
	component, name, body, ver string
}

func newFileClient(ver string) (*fileClient, error) {
	return &fileClient{ver: ver}, nil
}

func (c *fileClient) GetVersion() common.Version {
	return *common.MustNewVersion(c.ver)
}

func (c *fileClient) Write(component string, name string, body string) error {
	c.component, c.name, c.body = component, name, body
	return nil
}
