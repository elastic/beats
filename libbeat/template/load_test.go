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
	"github.com/elastic/beats/libbeat/version"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileLoader_Load(t *testing.T) {
	ver := "7.0.0"
	prefix := "mock"
	info := beat.Info{Version: ver, IndexPrefix: prefix}

	for name, test := range map[string]struct {
		cfg       TemplateConfig
		fields    []byte
		migration bool

		name string
	}{
		"default config": {
			cfg:       DefaultConfig(),
			name:      fmt.Sprintf("%s-%s", prefix, ver),
			migration: false,
		},
		"default config with migration": {
			cfg:       DefaultConfig(),
			name:      fmt.Sprintf("%s-%s", prefix, ver),
			migration: true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			fc, err := newFileClient(ver)
			require.NoError(t, err)
			fl := NewFileLoader(fc)
			err = fl.Load(test.cfg, info, test.fields, false)
			require.NoError(t, err)

			tmpl, err := New(ver, prefix, *common.MustNewVersion(ver), test.cfg, test.migration)
			require.NoError(t, err)
			body, err := buildBody(tmpl, test.cfg, test.fields)
			require.NoError(t, err)
			assert.Equal(t, common.MapStr{test.name: body}.StringToPrint()+"\n", fc.body)
		})
	}
}

type fileClient struct {
	ver  common.Version
	body string
}

func newFileClient(ver string) (*fileClient, error) {
	if ver == "" {
		ver = version.GetDefaultVersion()
	}
	v, err := common.NewVersion(ver)
	if err != nil {
		return nil, err
	}
	return &fileClient{ver: *v}, nil
}

func (c *fileClient) GetVersion() common.Version {
	return c.ver
}

func (c *fileClient) Write(name string, body string) error {
	c.body = body
	return nil
}
