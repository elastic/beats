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
	"github.com/elastic/beats/libbeat/common"
)

func Unpack(c *common.Config) (*TemplatesConfig, error) {
	var templatesRaw = struct {
		Templates []*common.Config `config:"templates"`
	}{}

	if err := c.Unpack(&templatesRaw); err != nil {
		return nil, err
	}

	var tc TemplatesConfig

	// use `settings.template.templates` if configured
	for _, t := range templatesRaw.Templates {
		var tmplCfg = DefaultTemplateCfg()
		t.Unpack(&tmplCfg)
		tc.Templates = append(tc.Templates, tmplCfg)
	}

	// fallback if no `settings.template.templates` was configured
	if len(tc.Templates) == 0 {
		var tmplCfg = DefaultTemplateCfg()
		c.Unpack(&tmplCfg)
		tc.Templates = append(tc.Templates, tmplCfg)
	}

	return &tc, nil
}

type TemplateConfig struct {
	AppendFields common.Fields `config:"append_fields"`

	Name    string `config:"name"`
	Pattern string `config:"pattern"`
	Fields  string `config:"fields"`
	Modules string `config:"modules"`
	JSON    struct {
		Enabled bool   `config:"enabled"`
		Path    string `config:"path"`
		Name    string `config:"name"`
	} `config:"json"`

	//TODO: check for overwrites
	Settings  TemplateSettings `config:"settings"`
	Enabled   bool             `config:"enabled"`
	Overwrite bool             `config:"overwrite"`
}

type TemplatesConfig struct {
	Templates []TemplateConfig `config:"-"`

	Settings TemplateSettings `config:"settings"`
}

type TemplateSettings struct {
	Index  map[string]interface{} `config:"index"`
	Source map[string]interface{} `config:"_source"`
}

var (
	// Defaults used in the template
	defaultDateDetection         = false
	defaultTotalFieldsLimit      = 10000
	defaultNumberOfRoutingShards = 30
)

func DefaultTemplateCfg() TemplateConfig {
	return TemplateConfig{
		Enabled:   true,
		Overwrite: false,
		Fields:    "",
	}
}
