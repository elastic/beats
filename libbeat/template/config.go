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
	"github.com/elastic/beats/v8/libbeat/beat"
	"github.com/elastic/beats/v8/libbeat/mapping"
)

// TemplateConfig holds config information about the Elasticsearch template
type TemplateConfig struct {
	Enabled bool   `config:"enabled"`
	Name    string `config:"name"`
	Pattern string `config:"pattern"`
	Fields  string `config:"fields"`
	JSON    struct {
		Enabled      bool   `config:"enabled"`
		Path         string `config:"path"`
		Name         string `config:"name"`
		IsDataStream bool   `config:"data_stream"`
	} `config:"json"`
	AppendFields mapping.Fields   `config:"append_fields"`
	Overwrite    bool             `config:"overwrite"`
	Settings     TemplateSettings `config:"settings"`
	Priority     int              `config:"priority"`
}

// TemplateSettings are part of the Elasticsearch template and hold index and source specific information.
type TemplateSettings struct {
	Index  map[string]interface{} `config:"index"`
	Source map[string]interface{} `config:"_source"`
}

// DefaultConfig for index template
func DefaultConfig(info beat.Info) TemplateConfig {
	return TemplateConfig{
		Name:     info.Beat + "-" + info.Version,
		Pattern:  info.Beat + "-" + info.Version,
		Enabled:  true,
		Fields:   "",
		Priority: 150,
	}
}
