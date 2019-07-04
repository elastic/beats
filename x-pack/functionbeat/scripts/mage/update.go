// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package mage

import (
	"path/filepath"

	"github.com/magefile/mage/mg"

	devtools "github.com/elastic/beats/dev-tools/mage"
	"github.com/elastic/beats/dev-tools/mage/target/common"
	"github.com/elastic/beats/dev-tools/mage/target/docs"
)

func init() {
	common.RegisterCheckDeps(Update.All)

	docs.RegisterDeps(Update.FieldDocs)
}

// Update target namespace.
type Update mg.Namespace

var Aliases = map[string]interface{}{
	"update": Update.All,
}

// All updates all generated content.
func (Update) All() {
	mg.Deps(Update.Fields, Update.Config, Update.FieldDocs)
}

// Config generates both the short and reference configs.
func (Update) Config() error {
	for _, provider := range SelectedProviders {
		devtools.BeatIndexPrefix += "-" + provider
		err := devtools.Config(devtools.ShortConfigType|devtools.ReferenceConfigType, XPackConfigFileParams(provider), provider)
		if err != nil {
			return err
		}
	}
	return nil
}

// Fields generates a fields.yml for the Beat.
func (Update) Fields() error {
	for _, provider := range SelectedProviders {
		output := filepath.Join(devtools.CWD(), provider, "fields.yml")
		err := devtools.GenerateFieldsYAMLTo(output)
		if err != nil {
			return err
		}
	}
	return nil
}

func (Update) FieldDocs() error {
	mg.Deps(includeFields)
	for _, provider := range SelectedProviders {
		fieldsYml := filepath.Join(devtools.CWD(), provider, "fields.yml")
		err := devtools.Docs.FieldDocs(fieldsYml)
		if err != nil {
			return err
		}
	}
	return nil
}

func includeFields() error {
	mg.Deps(Update.Fields)
	for _, provider := range SelectedProviders {
		input := filepath.Join(provider, "fields.yml")
		output := filepath.Join(provider, "include", "fields.go")
		err := devtools.GenerateFieldsGo(devtools.BeatName+"-"+provider, input, output)
		if err != nil {
			return err
		}
	}
	return nil
}
