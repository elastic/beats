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
	"github.com/menderesk/beats/v7/dev-tools/mage/target/build"
	"github.com/menderesk/beats/v7/dev-tools/mage/target/common"
	"github.com/menderesk/beats/v7/dev-tools/mage/target/dashboards"
	"github.com/menderesk/beats/v7/dev-tools/mage/target/docs"
)

func init() {
	common.RegisterCheckDeps(Update.All)

	dashboards.RegisterImportDeps(build.Build, Update.Dashboards)

	docs.RegisterDeps(Update.FieldDocs, Update.ModuleDocs)
}

// SelectLogic configures the types of project logic to use (OSS vs X-Pack).
var SelectLogic devtools.ProjectType

// Update target namespace.
type Update mg.Namespace

// All updates all generated content.
func (Update) All() {
	mg.Deps(Update.Fields, Update.Dashboards, Update.Config, Update.FieldDocs, Update.ModuleDocs, Update.Includes)
}

// Config updates the Beat's config files.
func (Update) Config() error {
	return config()
}

// Dashboards collects all the dashboards and generates index patterns.
func (Update) Dashboards() error {
	mg.Deps(fb.FieldsYML)
	switch SelectLogic {
	case devtools.XPackProject:
		return devtools.KibanaDashboards(devtools.OSSBeatDir("module"), devtools.XPackBeatDir("module"))
	default:
		return devtools.KibanaDashboards(devtools.OSSBeatDir("module"))
	}
}

// Fields updates all fields files (.go, .yml).
func (Update) Fields() {
	mg.Deps(fb.All)
}

// FieldDocs updates the field documentation.
func (Update) FieldDocs() error {
	mg.Deps(fb.FieldsAllYML)
	return devtools.Docs.FieldDocs(devtools.FieldsAllYML)
}

// ModuleDocs collects and updates the module documentation.
func (Update) ModuleDocs() error {
	return moduleDocs()
}

// Includes generates the include/list.go file.
func (Update) Includes() error {
	switch SelectLogic {
	case devtools.XPackProject:
		return devtools.GenerateModuleIncludeListGo()
	default:
		return nil
	}
}
