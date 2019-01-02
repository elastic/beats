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

	"github.com/elastic/beats/dev-tools/mage"
)

var fb fieldsBuilder

var _ mage.FieldsBuilder = fb

type fieldsBuilder struct{}

func (b fieldsBuilder) All() {
	mg.Deps(b.FieldsGo, b.FieldsYML, b.FieldsAllYML)
}

func (fieldsBuilder) FieldsGo() error {
	// TODO: Currently libbeat does not have any fields.go files because each
	// Beat is incorporating this into its include/fields.go file.
	return nil
}

func (fieldsBuilder) FieldsYML() error {
	var modules []string
	switch SelectLogic {
	case mage.OSSProject, mage.XPackProject:
		modules = append(modules, mage.OSSBeatDir("processors"))
	default:
		panic(mage.ErrUnknownProjectType)
	}

	if err := mage.GenerateFieldsYAMLTo(mage.FieldsYML, modules...); err != nil {
		return err
	}
	return mage.Copy(mage.FieldsYML, mage.FieldsYMLRoot)
}

func (fieldsBuilder) FieldsAllYML() error {
	// This isn't used because we don't generate any fields docs for libbeat.
	return nil
}
