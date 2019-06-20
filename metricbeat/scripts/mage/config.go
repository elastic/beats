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
	"os"
	"regexp"

	"github.com/magefile/mage/mg"

	"github.com/elastic/beats/dev-tools/mage"
)

// ConfigOSS generates both the short and reference configs for OSS distribution
func ConfigOSS() {
	mg.Deps(shortConfig, referenceConfigOSS, dockerConfig, GenerateDirModulesD)
}

// ConfigXPack generates both the short and reference configs for Licensed distribution
func ConfigXPack() {
	mg.Deps(shortConfig, referenceConfigXPack, dockerConfig, GenerateDirModulesD)
}

func shortConfig() error {
	var configParts = []string{
		mage.OSSBeatDir("_meta/common.yml"),
		mage.OSSBeatDir("_meta/setup.yml"),
		"{{ elastic_beats_dir }}/libbeat/_meta/config.yml",
	}

	for i, f := range configParts {
		configParts[i] = mage.MustExpand(f)
	}

	configFile := mage.BeatName + ".yml"
	mage.MustFileConcat(configFile, 0640, configParts...)
	mage.MustFindReplace(configFile, regexp.MustCompile("beatname"), mage.BeatName)
	mage.MustFindReplace(configFile, regexp.MustCompile("beat-index-prefix"), mage.BeatIndexPrefix)
	return nil
}

func referenceConfigOSS() error {
	return referenceConfig("module")
}

func referenceConfigXPack() error {
	return referenceConfig(mage.OSSBeatDir("module"), "module")
}

func referenceConfig(dirs ...string) error {
	const modulesConfigYml = "build/config.modules.yml"
	err := mage.GenerateModuleReferenceConfig(modulesConfigYml, dirs...)
	if err != nil {
		return err
	}
	defer os.Remove(modulesConfigYml)

	var configParts = []string{
		mage.OSSBeatDir("_meta/common.reference.yml"),
		modulesConfigYml,
		"{{ elastic_beats_dir }}/libbeat/_meta/config.reference.yml",
	}

	for i, f := range configParts {
		configParts[i] = mage.MustExpand(f)
	}

	configFile := mage.BeatName + ".reference.yml"
	mage.MustFileConcat(configFile, 0640, configParts...)
	mage.MustFindReplace(configFile, regexp.MustCompile("beatname"), mage.BeatName)
	mage.MustFindReplace(configFile, regexp.MustCompile("beat-index-prefix"), mage.BeatIndexPrefix)
	return nil
}

func dockerConfig() error {
	var configParts = []string{
		mage.OSSBeatDir("_meta/beat.docker.yml"),
		mage.LibbeatDir("_meta/config.docker.yml"),
	}

	return mage.FileConcat(mage.BeatName+".docker.yml", 0600, configParts...)
}
