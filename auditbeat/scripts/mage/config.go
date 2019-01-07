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
	"regexp"

	"github.com/pkg/errors"

	"github.com/elastic/beats/dev-tools/mage"
)

// -----------------------------------------------------------------------------
// Customizations specific to Auditbeat.
// - Config files are Go templates.

const (
	// ConfigTemplateGlob matches Auditbeat modules' config file templates.
	ConfigTemplateGlob      = "module/*/_meta/config*.yml.tmpl"
	shortConfigTemplate     = "build/auditbeat.yml.tmpl"
	referenceConfigTemplate = "build/auditbeat.reference.yml.tmpl"
)

func makeConfigTemplates(globs ...string) error {
	configFiles, err := mage.FindFiles(globs...)
	if err != nil {
		return errors.Wrap(err, "failed to find config templates")
	}

	var shortIn []string
	shortIn = append(shortIn, mage.OSSBeatDir("_meta/common.p1.yml"))
	shortIn = append(shortIn, configFiles...)
	shortIn = append(shortIn, mage.OSSBeatDir("_meta/common.p2.yml"))
	shortIn = append(shortIn, mage.LibbeatDir("_meta/config.yml"))
	if !mage.IsUpToDate(shortConfigTemplate, shortIn...) {
		fmt.Println(">> Building", shortConfigTemplate)
		mage.MustFileConcat(shortConfigTemplate, 0600, shortIn...)
		mage.MustFindReplace(shortConfigTemplate, regexp.MustCompile("beatname"), "{{.BeatName}}")
		mage.MustFindReplace(shortConfigTemplate, regexp.MustCompile("beat-index-prefix"), "{{.BeatIndexPrefix}}")
	}

	var referenceIn []string
	referenceIn = append(referenceIn, mage.OSSBeatDir("_meta/common.reference.yml"))
	referenceIn = append(referenceIn, configFiles...)
	referenceIn = append(referenceIn, mage.LibbeatDir("_meta/config.reference.yml"))
	if !mage.IsUpToDate(referenceConfigTemplate, referenceIn...) {
		fmt.Println(">> Building", referenceConfigTemplate)
		mage.MustFileConcat(referenceConfigTemplate, 0644, referenceIn...)
		mage.MustFindReplace(referenceConfigTemplate, regexp.MustCompile("beatname"), "{{.BeatName}}")
		mage.MustFindReplace(referenceConfigTemplate, regexp.MustCompile("beat-index-prefix"), "{{.BeatIndexPrefix}}")
	}

	return nil
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

// Config generates the auditbeat.yml and auditbeat.reference.yml config files.
// Set DEV_OS and DEV_ARCH to change the target host for the generated configs.
// Defaults to linux/amd64.
func Config(configTemplateGlobs ...string) error {
	if err := makeConfigTemplates(configTemplateGlobs...); err != nil {
		return errors.Wrap(err, "failed making config templates")
	}

	params := map[string]interface{}{
		"GOOS":      mage.EnvOr("DEV_OS", "linux"),
		"GOARCH":    mage.EnvOr("DEV_ARCH", "amd64"),
		"ArchBits":  archBits,
		"Reference": false,
	}
	fmt.Printf(">> Building auditbeat.yml for %v/%v\n", params["GOOS"], params["GOARCH"])
	mage.MustExpandFile(shortConfigTemplate, "auditbeat.yml", params)

	params["Reference"] = true
	fmt.Printf(">> Building auditbeat.reference.yml for %v/%v\n", params["GOOS"], params["GOARCH"])
	mage.MustExpandFile(referenceConfigTemplate, "auditbeat.reference.yml", params)
	return nil
}
