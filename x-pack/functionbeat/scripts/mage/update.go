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
	mg.Deps(Update.Fields, Update.IncludeFields, Update.Config, Update.FieldDocs, Update.VendorBeats)
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

// VendorBeats collects the vendor folder required to deploy the function for GCP.
func (Update) VendorBeats() error {
	gcpVendorPath := filepath.Join("provider", "gcp", "build", "vendor")
	vendorPath := filepath.Join("..", "..", "vendor")
	beatsVendorPath := filepath.Join(gcpVendorPath, "github.com", "elastic", "beats")

	cp := &devtools.CopyTask{
		Source: vendorPath,
		Dest:   gcpVendorPath,
		Mode:   0600,
	}
	err := cp.Execute()
	if err != nil {
		return err
	}

	cp = &devtools.CopyTask{
		Source:  "../../libbeat",
		Dest:    filepath.Join(beatsVendorPath, "libbeat"),
		Mode:    0600,
		Exclude: []string{"build", "_meta", "libbeat.yml"},
	}
	err = cp.Execute()
	if err != nil {
		return err
	}

	cp = &devtools.CopyTask{
		Source:  "../../x-pack/functionbeat",
		Dest:    filepath.Join(beatsVendorPath, "x-pack", "functionbeat"),
		Mode:    0600,
		Exclude: []string{"build", "_meta", "functionbeat.yml"},
	}
	err = cp.Execute()
	if err != nil {
		return err
	}

	return nil
}
