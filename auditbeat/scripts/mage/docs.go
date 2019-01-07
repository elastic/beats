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
	"path/filepath"
	"strings"

	"github.com/magefile/mage/sh"
	"github.com/pkg/errors"

	"github.com/elastic/beats/dev-tools/mage"
)

// CollectDocs collects documentation from modules.
func CollectDocs(basePaths ...string) error {

	// Generate config.yml files for each module.
	var configFiles []string
	for _, path := range basePaths {
		files, err := mage.FindFiles(filepath.Join(path, ConfigTemplateGlob))
		if err != nil {
			return errors.Wrap(err, "failed to find config templates")
		}

		configFiles = append(configFiles, files...)
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
	for _, path := range basePaths {
		if err := os.RemoveAll(filepath.Join(path, "docs/modules")); err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Join(path, "docs/modules"), 0755); err != nil {
			return err
		}
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
	args := []string{mage.OSSBeatDir("scripts/docs_collector.py"), "--base-paths"}
	args = append(args, basePaths...)

	err = sh.Run(python, args...)
	if err != nil {
		return err
	}

	esBeats, err := mage.ElasticBeatsDir()
	if err != nil {
		return err
	}

	return sh.Run(python, mage.LibbeatDir("scripts/generate_fields_docs.py"),
		XpackBeatDir(), mage.BeatName, esBeats, "--output_path", mage.OSSBeatDir())
}

// XpackBeatDir returns the x-pack/{beatname} directory for a Beat.
func XpackBeatDir() string {
	return mage.OSSBeatDir("../x-pack", mage.BeatName)
}
