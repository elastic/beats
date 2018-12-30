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
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/elastic/beats/dev-tools/mage"
)

const (
	// configTemplateGlob matches Auditbeat modules' config file templates.
	configTemplateGlob = "module/*/_meta/config*.yml.tmpl"
)

// OSSConfigFileParams returns the parameters for generating OSS config.
func OSSConfigFileParams() mage.ConfigFileParams {
	params, err := configFileParams(mage.OSSBeatDir())
	if err != nil {
		panic(err)
	}
	return params
}

// XPackConfigFileParams returns the parameters for generating X-Pack config.
func XPackConfigFileParams() mage.ConfigFileParams {
	params, err := configFileParams(mage.OSSBeatDir(), mage.XPackBeatDir())
	if err != nil {
		panic(err)
	}
	return params
}

func configFileParams(dirs ...string) (mage.ConfigFileParams, error) {
	var globs []string
	for _, dir := range dirs {
		globs = append(globs, filepath.Join(dir, configTemplateGlob))
	}

	configFiles, err := mage.FindFiles(globs...)
	if err != nil {
		return mage.ConfigFileParams{}, errors.Wrap(err, "failed to find config templates")
	}
	if len(configFiles) == 0 {
		return mage.ConfigFileParams{}, errors.Errorf("no config files found in %v", globs)
	}

	return mage.ConfigFileParams{
		ShortParts: join(
			mage.OSSBeatDir("_meta/common.p1.yml"),
			configFiles,
			mage.OSSBeatDir("_meta/common.p2.yml"),
			mage.LibbeatDir("_meta/config.yml"),
		),
		ReferenceParts: join(
			mage.OSSBeatDir("_meta/common.reference.yml"),
			configFiles,
			mage.LibbeatDir("_meta/config.reference.yml"),
		),
		DockerParts: []string{
			mage.OSSBeatDir("_meta/beat.docker.yml"),
			mage.LibbeatDir("_meta/config.docker.yml"),
		},
		ExtraVars: map[string]interface{}{
			"ArchBits": archBits,
		},
	}, nil
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

func join(items ...interface{}) []string {
	var out []string
	for _, item := range items {
		switch v := item.(type) {
		case string:
			out = append(out, v)
		case []string:
			out = append(out, v...)
		}
	}
	return out
}
