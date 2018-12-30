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

import "github.com/elastic/beats/libbeat/common"

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
	AppendFields common.Fields    `config:"append_fields"`
	Overwrite    bool             `config:"overwrite"`
	Settings     TemplateSettings `config:"settings"`
}

type TemplateSettings struct {
	Index  map[string]interface{} `config:"index"`
	Source map[string]interface{} `config:"_source"`
}

var (
	// DefaultConfig for index template
	DefaultConfig = TemplateConfig{
		Enabled: true,
		Fields:  "",
	}
)
