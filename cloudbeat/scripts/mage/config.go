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

package mage

import (
	devtools "github.com/elastic/beats/v7/dev-tools/mage"
)

// ConfigFileParams returns the default ConfigFileParams for generating
// cloudbeat*.yml files.
func XPackConfigFileParams() devtools.ConfigFileParams {
	p := devtools.DefaultConfigFileParams()
	p.Templates = append(p.Templates, devtools.OSSBeatDir("_meta/config/*.tmpl"))
	//	p.ExtraVars = map[string]interface{}{}
	//  used to include extra vars not in _meta/config/*.tmpl
	return p
	// Todo cloudbeat fill out according to cloudbeats required config.
}
