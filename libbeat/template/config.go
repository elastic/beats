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

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v7/libbeat/mapping"
)

const (
	IndexTemplateLegacy IndexTemplateType = iota
	IndexTemplateComponent
	IndexTemplateIndex
)

var (
	templateTypes = map[string]IndexTemplateType{
		"legacy":    IndexTemplateLegacy,
		"component": IndexTemplateComponent,
		"index":     IndexTemplateIndex,
	}
)

type IndexTemplateType uint8

// TemplateConfig holds config information about the Elasticsearch template
type TemplateConfig struct {
	Enabled bool   `config:"enabled"`
	Name    string `config:"name"`
	Pattern string `config:"pattern"`
	Fields  string `config:"fields"`
	JSON    struct {
		Enabled bool   `config:"enabled"`
		Path    string `config:"path"`
		Name    string `config:"name"`
	} `config:"json"`
	AppendFields mapping.Fields    `config:"append_fields"`
	Overwrite    bool              `config:"overwrite"`
	Settings     TemplateSettings  `config:"settings"`
	Order        int               `config:"order"`
	Priority     int               `config:"priority"`
	Type         IndexTemplateType `config:"type"`
}

// TemplateSettings are part of the Elasticsearch template and hold index and source specific information.
type TemplateSettings struct {
	Index  map[string]interface{} `config:"index"`
	Source map[string]interface{} `config:"_source"`
}

// DefaultConfig for index template
func DefaultConfig() TemplateConfig {
	return TemplateConfig{
		Enabled:  true,
		Fields:   "",
		Type:     IndexTemplateIndex,
		Order:    1,
		Priority: 150,
	}
}

func Unpack(c *common.Config) (TemplateConfig, error) {
	if c == nil {
		return DefaultConfig(), nil
	}

	jsonEnabled, _ := c.Bool("json.enabled", -1)
	config := DefaultConfig()
	if jsonEnabled {
		cfgwarn.Deprecate("8.0.0", "Please migrate your JSON templates from legacy template format to composable index template.")
		config.Type = IndexTemplateLegacy
	}
	var err error
	if c != nil {
		err = c.Unpack(&config)
	}
	return config, err

}

func (t *IndexTemplateType) Unpack(v string) error {
	if v == "" {
		*t = IndexTemplateIndex
		return nil
	}

	cfgwarn.Deprecate("8.0.0", "do not use setup.template.type, it is deprecated, data streams will be loaded automatically")

	var tt IndexTemplateType
	var ok bool
	if tt, ok = templateTypes[v]; !ok {
		return fmt.Errorf("unknown index template type: %s", v)
	}
	*t = tt

	return nil
}
