// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package mage

import (
	"os"

	"github.com/magefile/mage/mg"

	"github.com/elastic/beats/dev-tools/mage"
)

var fb fieldsBuilder

var _ mage.FieldsBuilder = fb

type fieldsBuilder struct{}

func (b fieldsBuilder) All() {
	mg.Deps(b.FieldsGo, b.FieldsYML, b.FieldsAllYML)
}

func (b fieldsBuilder) FieldsGo() error { return b.commonFieldsGo() }

func (b fieldsBuilder) FieldsYML() error {
	if err := mage.GenerateFieldsYAMLTo(mage.FieldsYML); err != nil {
		return err
	}
	return mage.Copy(mage.FieldsYML, mage.FieldsYMLRoot)
}

func (b fieldsBuilder) FieldsAllYML() error {
	return mage.GenerateFieldsYAMLTo(mage.FieldsAllYML)
}

func (b fieldsBuilder) commonFieldsGo() error {
	const file = "build/fields/fields.common.yml"
	if err := mage.GenerateFieldsYAMLTo(file); err != nil {
		return err
	}
	defer os.Remove(file)
	return mage.GenerateFieldsGo(file, "include/fields.go")
}
