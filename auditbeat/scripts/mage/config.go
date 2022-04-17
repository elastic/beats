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
	"fmt"
	"path/filepath"

	devtools "github.com/menderesk/beats/v7/dev-tools/mage"
)

const (
	// configTemplateGlob matches Auditbeat modules' config file templates.
	configTemplateGlob = "module/*/_meta/config*.yml.tmpl"
)

// OSSConfigFileParams returns the parameters for generating OSS config.
func OSSConfigFileParams() devtools.ConfigFileParams {
	params, err := configFileParams(devtools.OSSBeatDir())
	if err != nil {
		panic(err)
	}
	return params
}

// XPackConfigFileParams returns the parameters for generating X-Pack config.
func XPackConfigFileParams() devtools.ConfigFileParams {
	params, err := configFileParams(devtools.OSSBeatDir(), devtools.XPackBeatDir())
	if err != nil {
		panic(err)
	}
	return params
}

func configFileParams(dirs ...string) (devtools.ConfigFileParams, error) {
	var globs []string
	for _, dir := range dirs {
		globs = append(globs, filepath.Join(dir, configTemplateGlob))
	}

	configFiles, err := devtools.FindFiles(globs...)
	if err != nil {
		return devtools.ConfigFileParams{}, fmt.Errorf("failed to find config templates: %w", err)
	}
	if len(configFiles) == 0 {
		return devtools.ConfigFileParams{}, fmt.Errorf("no config files found in %v", globs)
	}
	devtools.MustFileConcat("build/config.modules.yml.tmpl", 0o644, configFiles...)

	p := devtools.DefaultConfigFileParams()
	p.Templates = append(p.Templates, devtools.OSSBeatDir("_meta/config/*.tmpl"))
	p.Templates = append(p.Templates, "build/config.modules.yml.tmpl")
	p.ExtraVars = map[string]interface{}{
		"ArchBits": archBits,
	}
	return p, nil
}

// archBits returns the number of bit width of the GOARCH architecture value.
// This function is used by the auditd module configuration templates to
// generate architecture specific audit rules.
func archBits(goarch string) int {
	switch goarch {
	case "386", "arm":
		return 32
	default:
		return 64
	}
}
