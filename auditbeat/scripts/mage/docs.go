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
	"os"
	"path/filepath"
	"strings"

	"github.com/magefile/mage/sh"

	devtools "github.com/elastic/beats/v7/dev-tools/mage"
)

// ModuleDocs collects documentation from modules (both OSS and X-Pack).
func ModuleDocs() error {
	dirsWithModules := []string{
		devtools.OSSBeatDir(),
		devtools.XPackBeatDir(),
	}

	// Generate config.yml files for each module.
	var configFiles []string
	for _, path := range dirsWithModules {
		files, err := devtools.FindFiles(filepath.Join(path, configTemplateGlob))
		if err != nil {
			return fmt.Errorf("failed to find config templates: %w", err)
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
		devtools.MustExpandFile(src, dst, params)
	}
	defer devtools.Clean(configs)

	// Remove old.
	for _, path := range dirsWithModules {
		if err := os.RemoveAll(filepath.Join(path, "docs/modules")); err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Join(path, "docs/modules"), 0o755); err != nil {
			return err
		}
	}

	// Run the docs_collector.py script.
	ve, err := devtools.PythonVirtualenv()
	if err != nil {
		return err
	}

	python, err := devtools.LookVirtualenvPath(ve, "python")
	if err != nil {
		return err
	}

	// TODO: Port this script to Go.
	args := []string{devtools.OSSBeatDir("scripts/docs_collector.py"), "--base-paths"}
	args = append(args, dirsWithModules...)

	return sh.Run(python, args...)
}

// FieldDocs generates docs/fields.asciidoc containing all fields
// (including x-pack).
func FieldDocs() error {
	inputs := []string{
		devtools.OSSBeatDir("module"),
		devtools.XPackBeatDir("module"),
	}
	output := devtools.CreateDir("build/fields/fields.all.yml")
	if err := devtools.GenerateFieldsYAMLTo(output, inputs...); err != nil {
		return err
	}
	return devtools.Docs.FieldDocs(output)
}
