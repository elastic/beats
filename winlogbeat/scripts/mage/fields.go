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

	"github.com/magefile/mage/mg"

	devtools "github.com/elastic/beats/dev-tools/mage"
)

var fb fieldsBuilder

var _ devtools.FieldsBuilder = fb

type fieldsBuilder struct{}

func (b fieldsBuilder) All() {
	mg.Deps(b.FieldsGo, b.FieldsYML, b.FieldsAllYML)
}

func (b fieldsBuilder) FieldsGo() error {
	switch SelectLogic {
	case devtools.OSSProject:
		return b.commonFieldsGo()
	case devtools.XPackProject:
		return devtools.GenerateModuleFieldsGo("module")
	default:
		panic(devtools.ErrUnknownProjectType)
	}
}

func (fieldsBuilder) FieldsYML() error {
	var modules []string
	switch SelectLogic {
	case devtools.OSSProject:
		// No OSS modules.
	case devtools.XPackProject:
		modules = append(modules, devtools.XPackBeatDir("module"))
	default:
		panic(devtools.ErrUnknownProjectType)
	}

	if err := devtools.GenerateFieldsYAMLTo(devtools.FieldsYML, modules...); err != nil {
		return err
	}
	return devtools.Copy(devtools.FieldsYML, devtools.FieldsYMLRoot)
}

func (fieldsBuilder) FieldsAllYML() error {
	return devtools.GenerateFieldsYAMLTo(devtools.FieldsAllYML, devtools.XPackBeatDir("module"))
}

func (b fieldsBuilder) commonFieldsGo() error {
	const file = "build/fields/fields.common.yml"
	if err := devtools.GenerateFieldsYAMLTo(file); err != nil {
		return err
	}
	defer os.Remove(file)
	return devtools.GenerateFieldsGo(file, "include/fields.go")
}
