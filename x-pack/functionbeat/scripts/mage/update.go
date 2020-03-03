// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package mage

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/magefile/mage/mg"

	devtools "github.com/elastic/beats/v7/dev-tools/mage"
	"github.com/elastic/beats/v7/dev-tools/mage/gotool"
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
	for _, f := range []string{"pubsub", "storage"} {
		gcpVendorPath := filepath.Join("provider", "gcp", "build", f, "vendor")
		err := os.RemoveAll(gcpVendorPath)
		if err != nil {
			return err
		}

		deps, err := gotool.ListDeps("github.com/elastic/beats/v7/x-pack/functionbeat/provider/gcp/" + f)
		if err != nil {
			return err
		}

		for _, d := range deps {
			var in string
			if strings.HasPrefix(d, "github.com/elastic/beats/v7") {
				in = strings.ReplaceAll(d, "github.com/elastic/beats/v7/", "")
				in = filepath.Join("..", "..", in)
			} else {
				in = filepath.Join("..", "..", "vendor", d)
			}

			out := strings.ReplaceAll(d, "github.com/elastic/beats/v7/vendor", "")

			cp := &devtools.CopyTask{
				Source: in,
				Dest:   filepath.Join(gcpVendorPath, out),
				Mode:   0600,
				Exclude: []string{
					".*_test.go$",
					".*.yml",
				},
			}
			err := cp.Execute()
			if err != nil {
				return err
			}
		}

	}

	return nil
}
