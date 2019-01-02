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

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"github.com/pkg/errors"

	"github.com/elastic/beats/dev-tools/mage"
	"github.com/elastic/beats/dev-tools/mage/target/build"
	"github.com/elastic/beats/dev-tools/mage/target/common"
	"github.com/elastic/beats/dev-tools/mage/target/dashboards"
	"github.com/elastic/beats/dev-tools/mage/target/docs"
	"github.com/elastic/beats/dev-tools/mage/target/integtest"
	"github.com/elastic/beats/dev-tools/mage/target/unittest"
)

func init() {
	common.RegisterCheckDeps(Update.All)

	dashboards.RegisterImportDeps(build.Build, Update.Dashboards)

	docs.RegisterDeps(Update.FieldDocs, Update.ModuleDocs)

	unittest.RegisterGoTestDeps(Update.Fields)
	unittest.RegisterPythonTestDeps(Update.Fields)

	integtest.RegisterPythonTestDeps(Update.Fields, Update.Dashboards)
}

var (
	// SelectLogic configures the types of project logic to use (OSS vs X-Pack).
	SelectLogic mage.ProjectType
)

// Update target namespace.
type Update mg.Namespace

// All updates all generated content.
func (Update) All() {
	mg.Deps(Update.Fields, Update.Dashboards, Update.Config,
		Update.Includes, Update.ModuleDocs, Update.FieldDocs)
}

// Fields updates all fields files (.go, .yml).
func (Update) Fields() {
	mg.Deps(fb.All)
}

// Includes updates include/list.go.
func (Update) Includes() error {
	return mage.GenerateModuleIncludeListGo()
}

// Config updates the Beat's config files.
func (Update) Config() error {
	return config()
}

// Dashboards collects all the dashboards and generates index patterns.
func (Update) Dashboards() error {
	mg.Deps(fb.FieldsYML)
	switch SelectLogic {
	case mage.OSSProject:
		return mage.KibanaDashboards("module")
	case mage.XPackProject:
		return mage.KibanaDashboards(mage.OSSBeatDir("module"), "module")
	default:
		panic(mage.ErrUnknownProjectType)
	}
}

// FieldDocs generates docs/fields.asciidoc containing all fields (including
// x-pack).
func (Update) FieldDocs() error {
	mg.Deps(fb.FieldsAllYML)
	return mage.Docs.FieldDocs(mage.FieldsAllYML)
}

// ModuleDocs collects documentation from modules (both OSS and X-Pack).
func (Update) ModuleDocs() error {
	dirsWithModules := []string{
		mage.OSSBeatDir(),
		mage.XPackBeatDir(),
	}

	// Generate config.yml files for each module.
	var configFiles []string
	for _, path := range dirsWithModules {
		files, err := mage.FindFiles(filepath.Join(path, configTemplateGlob))
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
	for _, path := range dirsWithModules {
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
	args = append(args, dirsWithModules...)

	return sh.Run(python, args...)
}
