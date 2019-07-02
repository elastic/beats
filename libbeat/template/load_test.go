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
	"fmt"
	"testing"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileLoader_Load(t *testing.T) {
	ver := "7.0.0"
	prefix := "mock"
	order := 1
	info := beat.Info{Version: ver, IndexPrefix: prefix}
	tmplName := fmt.Sprintf("%s-%s", prefix, ver)

	for name, test := range map[string]struct {
		settings TemplateSettings
		body     common.MapStr
	}{
		"load minimal config info": {
			body: common.MapStr{
				"index_patterns": []string{"mock-7.0.0-*"},
				"order":          order,
				"settings":       common.MapStr{"index": nil}},
		},
		"load minimal config with index settings": {
			settings: TemplateSettings{Index: common.MapStr{"code": "best_compression"}},
			body: common.MapStr{
				"index_patterns": []string{"mock-7.0.0-*"},
				"order":          order,
				"settings":       common.MapStr{"index": common.MapStr{"code": "best_compression"}}},
		},
		"load minimal config with source settings": {
			settings: TemplateSettings{Source: common.MapStr{"enabled": false}},
			body: common.MapStr{
				"index_patterns": []string{"mock-7.0.0-*"},
				"order":          order,
				"settings":       common.MapStr{"index": nil},
				"mappings": common.MapStr{
					"_source":           common.MapStr{"enabled": false},
					"_meta":             common.MapStr{"beat": prefix, "version": ver},
					"date_detection":    false,
					"dynamic_templates": nil,
					"properties":        nil,
				}},
		},
	} {
		t.Run(name, func(t *testing.T) {
			fc, err := newFileClient(ver)
			require.NoError(t, err)
			fl := NewFileLoader(fc)

			cfg := DefaultConfig()
			cfg.Settings = test.settings

			err = fl.Load(cfg, info, nil, false)
			require.NoError(t, err)
			assert.Equal(t, "template", fc.component)
			assert.Equal(t, tmplName, fc.name)
			assert.Equal(t, test.body.StringToPrint()+"\n", fc.body)
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
