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
	devtools "github.com/elastic/beats/dev-tools/mage"
)

// ConfigFileParams returns the default ConfigFileParams for generating
// packetbeat*.yml files.
func ConfigFileParams() devtools.ConfigFileParams {
	return devtools.ConfigFileParams{
		ShortParts: []string{
			devtools.OSSBeatDir("_meta/beat.yml"),
			devtools.LibbeatDir("_meta/config.yml.tmpl"),
		},
		ReferenceParts: []string{
			devtools.OSSBeatDir("_meta/beat.reference.yml"),
			devtools.LibbeatDir("_meta/config.reference.yml.tmpl"),
		},
		DockerParts: []string{
			devtools.OSSBeatDir("_meta/beat.docker.yml"),
			devtools.LibbeatDir("_meta/config.docker.yml"),
		},
		ExtraVars: map[string]interface{}{
			"UseObserverProcessor": true,
			"ExcludeDashboards":    true,
		},
	}
}
