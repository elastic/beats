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
	"github.com/magefile/mage/mg"

	devtools "github.com/menderesk/beats/v7/dev-tools/mage"
)

const modulesConfigYml = "build/config.modules.yml.tmpl"

func configFileParams(moduleDirs ...string) devtools.ConfigFileParams {
	collectModuleConfig := func() error {
		return devtools.GenerateModuleReferenceConfig(modulesConfigYml, moduleDirs...)
	}
	mg.Deps(collectModuleConfig)

	p := devtools.DefaultConfigFileParams()
	p.Templates = append(p.Templates, devtools.OSSBeatDir("_meta/config/*.tmpl"), modulesConfigYml)
	p.ExtraVars = map[string]interface{}{
		"UseKubernetesMetadataProcessor": true,
	}
	return p
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
	args.Templates = append(args.Templates, "_meta/config/*.tmpl")
	return args
}
