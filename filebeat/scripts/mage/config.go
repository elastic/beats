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
	"github.com/elastic/beats/dev-tools/mage"
)

const modulesConfigYml = "build/config.modules.yml"

func configFileParams(moduleDirs ...string) mage.ConfigFileParams {
	collectModuleConfig := func() error {
		return mage.GenerateModuleReferenceConfig(modulesConfigYml, moduleDirs...)
	}

	return mage.ConfigFileParams{
		ShortParts: []string{
			mage.OSSBeatDir("_meta/common.p1.yml"),
			mage.OSSBeatDir("_meta/common.p2.yml"),
			mage.LibbeatDir("_meta/config.yml"),
		},
		ReferenceDeps: []interface{}{collectModuleConfig},
		ReferenceParts: []string{
			mage.OSSBeatDir("_meta/common.reference.p1.yml"),
			modulesConfigYml,
			mage.OSSBeatDir("_meta/common.reference.inputs.yml"),
			mage.OSSBeatDir("_meta/common.reference.p2.yml"),
			mage.LibbeatDir("_meta/config.reference.yml"),
		},
		DockerParts: []string{
			mage.OSSBeatDir("_meta/beat.docker.yml"),
			mage.LibbeatDir("_meta/config.docker.yml"),
		},
	}
}

// OSSConfigFileParams returns the default ConfigFileParams for generating
// filebeat*.yml files.
func OSSConfigFileParams(moduleDirs ...string) mage.ConfigFileParams {
	return configFileParams(mage.OSSBeatDir("module"))
}

// XPackConfigFileParams returns the default ConfigFileParams for generating
// filebeat*.yml files.
func XPackConfigFileParams() mage.ConfigFileParams {
	args := configFileParams(mage.OSSBeatDir("module"), "module")
	args.ReferenceParts = []string{
		mage.OSSBeatDir("_meta/common.reference.p1.yml"),
		modulesConfigYml,
		mage.OSSBeatDir("_meta/common.reference.inputs.yml"),
		"_meta/common.reference.inputs.yml", // Added only to X-Pack.
		mage.OSSBeatDir("_meta/common.reference.p2.yml"),
		mage.LibbeatDir("_meta/config.reference.yml"),
	}
	return args
}
