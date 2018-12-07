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
	"strings"

	"github.com/magefile/mage/sh"
	"github.com/pkg/errors"

	"github.com/elastic/beats/dev-tools/mage"
)

// CollectDocs collects documentation from modules.
//
// TODO: This needs to be updated to collect docs from x-pack/auditbeat.
func CollectDocs() error {
	// Generate config.yml files for each module.
	configFiles, err := mage.FindFiles(ConfigTemplateGlob)
	if err != nil {
		return errors.Wrap(err, "failed to find config templates")
	}

	var configs []string
	params := map[string]interface{}{
		"GOOS":      "linux",
		"GOARCH":    "amd64",
		"ArchBits":  archBits,
		"Reference": false,
	}
	for _, src := range configFiles {
		dst := strings.TrimSuffix(src, ".tmpl")
		configs = append(configs, dst)
		mage.MustExpandFile(src, dst, params)
	}
	defer mage.Clean(configs)

	// Remove old.
	if err = os.RemoveAll(mage.OSSBeatDir("docs/modules")); err != nil {
		return err
	}
	if err = os.MkdirAll(mage.OSSBeatDir("docs/modules"), 0755); err != nil {
		return err
	}

	// Run the docs_collector.py script.
	ve, err := mage.PythonVirtualenv()
	if err != nil {
		return err
	}

	python, err := mage.LookVirtualenvPath(ve, "python")
	if err != nil {
		return err
	}

	// TODO: Port this script to Go.
	return sh.Run(python,
		mage.OSSBeatDir("scripts/docs_collector.py"),
		"--beat", mage.BeatName)
}
