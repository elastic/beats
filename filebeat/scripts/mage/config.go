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

const modulesConfigYml = "build/config.modules.yml"

func configFileParams(moduleDirs ...string) devtools.ConfigFileParams {
	collectModuleConfig := func() error {
		return devtools.GenerateModuleReferenceConfig(modulesConfigYml, moduleDirs...)
	}

	return devtools.ConfigFileParams{
		ShortParts: []string{
			devtools.OSSBeatDir("_meta/common.p1.yml"),
			devtools.OSSBeatDir("_meta/common.p2.yml"),
			devtools.LibbeatDir("_meta/config.yml.tmpl"),
		},
		ReferenceDeps: []interface{}{collectModuleConfig},
		ReferenceParts: []string{
			devtools.OSSBeatDir("_meta/common.reference.p1.yml"),
			modulesConfigYml,
			devtools.OSSBeatDir("_meta/common.reference.inputs.yml"),
			devtools.OSSBeatDir("_meta/common.reference.p2.yml"),
			devtools.LibbeatDir("_meta/config.reference.yml.tmpl"),
		},
		DockerParts: []string{
			devtools.OSSBeatDir("_meta/beat.docker.yml"),
			devtools.LibbeatDir("_meta/config.docker.yml"),
		},
	}
}

// OSSConfigFileParams returns the default ConfigFileParams for generating
// filebeat*.yml files.
func OSSConfigFileParams(moduleDirs ...string) devtools.ConfigFileParams {
	return configFileParams(devtools.OSSBeatDir("module"))
}

// XPackConfigFileParams returns the default ConfigFileParams for generating
// filebeat*.yml files.
func XPackConfigFileParams() devtools.ConfigFileParams {
	args := configFileParams(devtools.OSSBeatDir("module"), "module")
	args.ReferenceParts = []string{
		devtools.OSSBeatDir("_meta/common.reference.p1.yml"),
		modulesConfigYml,
		devtools.OSSBeatDir("_meta/common.reference.inputs.yml"),
		"_meta/common.reference.inputs.yml", // Added only to X-Pack.
		devtools.OSSBeatDir("_meta/common.reference.p2.yml"),
		devtools.LibbeatDir("_meta/config.reference.yml.tmpl"),
	}
	return args
}
