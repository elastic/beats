// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package mage

import (
	"path/filepath"

	"github.com/magefile/mage/mg"

	devtools "github.com/elastic/beats/dev-tools/mage"
)

// Update target namespace.
type Update mg.Namespace

// Aliases stores aliases for the targets.
var Aliases = map[string]interface{}{
	"update": Update.All,
}

// All updates all generated content.
func (Update) All() {
	mg.Deps(Update.Fields, Update.IncludeFields, Update.Config, Update.FieldDocs)
}

// Config generates both the short and reference configs.
func (Update) Config() error {
	idxPrefix := devtools.BeatIndexPrefix
	for _, provider := range SelectedProviders {
		devtools.BeatIndexPrefix = idxPrefix + "-" + provider
		err := devtools.Config(devtools.ShortConfigType|devtools.ReferenceConfigType, XPackConfigFileParams(provider), provider)
		if err != nil {
			return err
		}
	}
	return nil
}

// Fields generates a fields.yml for the Beat.
func (Update) Fields() error {
	for _, provider := range getConfiguredProviders() {
		output := filepath.Join(devtools.CWD(), provider, "fields.yml")
		err := devtools.GenerateFieldsYAMLTo(output)
		if err != nil {
			return err
		}
	}
	return nil
}

// FieldDocs collects all fields by provider and generates documentation for them.
func (Update) FieldDocs() error {
	var inputs []string
	for _, provider := range SelectedProviders {
		inputs = append(inputs, provider)
	}
	output := devtools.CreateDir("build/fields/fields.all.yml")
	if err := devtools.GenerateFieldsYAMLTo(output, inputs...); err != nil {
		return err
	}
	return devtools.Docs.FieldDocs(output)
}

// IncludeFields generates include/fields.go by provider.
func (Update) IncludeFields() error {
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
