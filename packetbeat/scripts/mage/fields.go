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
	"go.uber.org/multierr"

	"github.com/elastic/beats/dev-tools/mage"
)

var fb fieldsBuilder

var _ mage.FieldsBuilder = fb

type fieldsBuilder struct{}

func (b fieldsBuilder) All() {
	mg.Deps(b.FieldsGo, b.FieldsYML, b.FieldsAllYML)
}

func (b fieldsBuilder) FieldsGo() error {
	switch SelectLogic {
	case mage.OSSProject:
		return multierr.Combine(
			b.commonFieldsGo(),
			b.protosFieldsGo(),
		)
	case mage.XPackProject:
		return nil
	default:
		panic(mage.ErrUnknownProjectType)
	}
}

func (b fieldsBuilder) FieldsYML() error {
	var modules []string
	switch SelectLogic {
	case mage.OSSProject, mage.XPackProject:
		modules = append(modules, mage.OSSBeatDir("protos"))
	default:
		panic(mage.ErrUnknownProjectType)
	}

	if err := mage.GenerateFieldsYAMLTo(mage.FieldsYML, modules...); err != nil {
		return err
	}
	return mage.Copy(mage.FieldsYML, mage.FieldsYMLRoot)
}

func (b fieldsBuilder) FieldsAllYML() error {
	return mage.GenerateFieldsYAMLTo(mage.FieldsAllYML,
		mage.OSSBeatDir("protos"),
	)
}

func (b fieldsBuilder) commonFieldsGo() error {
	const file = "build/fields/fields.common.yml"
	if err := mage.GenerateFieldsYAMLTo(file); err != nil {
		return err
	}
	defer os.Remove(file)
	return mage.GenerateFieldsGo(file, "include/fields.go")
}

// protosFieldsGo generates a fields.go for each protocol.
func (fieldsBuilder) protosFieldsGo() error {
	return mage.GenerateModuleFieldsGo(mage.OSSBeatDir("protos"))
}
