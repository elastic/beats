// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package mage

import (
	"github.com/magefile/mage/mg"

	devtools "github.com/elastic/beats/v8/dev-tools/mage"
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
	return devtools.Config(devtools.ShortConfigType|devtools.ReferenceConfigType, XPackConfigFileParams(), ".")
}

// Fields generates a fields.yml for the Beat.
func (Update) Fields() error {
	return devtools.GenerateFieldsYAML()
}

// FieldDocs collects all fields by provider and generates documentation for them.
func (Update) FieldDocs() error {
	mg.Deps(Update.Fields)

	return devtools.Docs.FieldDocs("fields.yml")
}

// IncludeFields generates include/fields.go by provider.
func (Update) IncludeFields() error {
	mg.Deps(Update.Fields)

	return devtools.GenerateAllInOneFieldsGo()
}
