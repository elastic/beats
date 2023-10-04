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

package template

import (
	"strings"
	"testing"

	"github.com/elastic/beats/v7/libbeat/asset"
	"github.com/elastic/beats/v7/libbeat/template"
	"github.com/elastic/beats/v7/libbeat/version"
	libversion "github.com/elastic/elastic-agent-libs/version"
)

// MaxDefaultFieldLength is the limit on the number of default_field values
// allowed by the test. This is less that the 1024 limit of Elasticsearch to
// give a little room for custom fields and the expansion of `fields.*`.
const MaxDefaultFieldLength = 1000

// TestTemplate executes tests on the Beat's index template.
func TestTemplate(t *testing.T, beatName string, elasticLicensed bool) {
	t.Run("default_field length", testTemplateDefaultFieldLength(beatName, elasticLicensed))
}

// testTemplateDefaultFieldLength constructs a template based on the embedded
// fields.yml data verifies that the length is less than 1000.
func testTemplateDefaultFieldLength(beatName string, elasticLicensed bool) func(*testing.T) {
	return func(t *testing.T) {
		// 7.0 is when default_field was introduced.
		esVersion, err := libversion.New("7.0.0")
		if err != nil {
			t.Fatal(err)
		}

		// Generate a template based on the embedded fields.yml data.
		tmpl, err := template.New(false, version.GetDefaultVersion(), beatName, elasticLicensed, *esVersion, template.TemplateConfig{}, false)
		if err != nil {
			t.Fatal(err)
		}

		fieldsBytes, err := asset.GetFields(beatName)
		if err != nil {
			t.Fatal("Failed to get embedded fields.yml asset data:", err)
		}

		fields, err := tmpl.LoadBytes(fieldsBytes)
		if err != nil {
			t.Fatal("Failed to load template bytes:", err)
		}

		templateMap := tmpl.Generate(fields, nil, nil)

		v, _ := templateMap.GetValue("template.settings.index.query.default_field")
		defaultValue, ok := v.([]string)
		if !ok {
			t.Fatalf("template.settings.index.query.default_field value has an unexpected type: %T", v)
		}

		if len(defaultValue) > MaxDefaultFieldLength {
			t.Fatalf("Too many fields (%d>%d) in %v index template"+
				"settings.index.query.default_field for comfort. By default "+
				"Elasticsearch has a limit of 1024 fields in a query so we need "+
				"to keep the number of fields below 1024. Adding 'default_field: "+
				"false' to fields or groups in a fields.yml can be used to "+
				"reduce the number of text/keyword fields that end up in "+
				"default_field.",
				len(defaultValue), MaxDefaultFieldLength, strings.Title(beatName))
		}
		t.Logf("%v template has %d fields in default_field.", strings.Title(beatName), len(defaultValue))
	}
}
