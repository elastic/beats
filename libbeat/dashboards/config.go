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

package dashboards

import "time"

// Config represents the config values for dashboards
type Config struct {
	Enabled            bool              `config:"enabled"`
	KibanaIndex        string            `config:"kibana_index"`
	Index              string            `config:"index"`
	Dir                string            `config:"directory"`
	File               string            `config:"file"`
	Beat               string            `config:"beat"`
	URL                string            `config:"url"`
	OnlyDashboards     bool              `config:"only_dashboards"`
	OnlyIndex          bool              `config:"only_index"`
	AlwaysKibana       bool              `config:"always_kibana"`
	Retry              *Retry            `config:"retry"`
	StringReplacements map[string]string `config:"string_replacements"`
}

// Retry handles query retries
type Retry struct {
	Enabled  bool          `config:"enabled"`
	Interval time.Duration `config:"interval"`
	Maximum  uint          `config:"maximum"`
}

var defaultConfig = Config{
	KibanaIndex:  ".kibana",
	AlwaysKibana: false,
	Retry: &Retry{
		Enabled:  false,
		Interval: time.Second,
		Maximum:  0,
	},
}
var (
	defaultDirectory = "kibana"
)
